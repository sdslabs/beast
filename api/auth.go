package api

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
)

var contestantUsernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,11}$`)

func validatePassword(password string) error {
	if len(password) < 12 || len(password) > 128 {
		return errors.New("password must contain between 12 and 128 bytes")
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("password cannot contain only whitespace")
	}
	return nil
}

// Acts as a middleware to authorize user
// @Summary Handles authorization of user
// @Description Authorizes user by checking if JWT token exists and is valid
// @Tags auth
// @Accept json
// @Produce json
// @Failure 401 {object} api.HTTPPlainResp
// @Security ApiKeyAuth
func authorizeRoles(c *gin.Context, roles int) {
	values := strings.Fields(c.GetHeader("Authorization"))
	if len(values) != 2 || values[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "No Token Provided",
		})
		c.Abort()
		return
	}

	claims, err := auth.AuthorizeClaims(values[1], roles)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
		c.Abort()
		return
	}
	if database.Db == nil || database.DBMux == nil {
		c.JSON(http.StatusServiceUnavailable, HTTPPlainResp{Message: "Authorization could not be verified"})
		c.Abort()
		return
	}
	user, err := database.QueryFirstUserEntry("username", claims.User)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusUnauthorized
		}
		c.JSON(status, HTTPPlainResp{Message: "Authorization could not be verified"})
		c.Abort()
		return
	}
	if user.Status != 0 || user.Role != claims.Role {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{Message: "Authorization is no longer valid"})
		c.Abort()
		return
	}
	c.Set("authClaims", claims)
	c.Next()
}

func authorize(c *gin.Context) {
	authorizeRoles(c, core.MANAGER|core.ADMIN|core.USER)
}

// Acts as a middleware to authorize manager roles
// @Summary Handles authorization of manager roles
// @Description Authorizes authors and admin by checking if JWT token exists and is valid
// @Tags auth
// @Accept json
// @Produce json
// @Failure 401 {object} api.HTTPPlainResp
// @Security ApiKeyAuth
func managerAuthorize(c *gin.Context) {
	authorizeRoles(c, core.MANAGER|core.ADMIN)
}

// Acts as a middleware to authorize admin roles
// @Summary Handles authorization of admin roles
// @Description Authorizes admin by checking if JWT token exists and is valid
// @Tags auth
// @Accept json
// @Produce json
// @Failure 401 {object} api.HTTPPlainResp
// @Security ApiKeyAuth
func adminAuthorize(c *gin.Context) {
	authorizeRoles(c, core.ADMIN)
}

func resetPasswordAuthorize(c *gin.Context) {
	values := strings.Fields(c.GetHeader("Authorization"))
	if len(values) != 2 || values[0] != "Bearer" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, HTTPPlainResp{Message: "No Token Provided"})
		return
	}
	claims, err := auth.AuthorizeClaims(values[1], core.MANAGER|core.ADMIN|core.USER)
	if err != nil {
		claims, err = auth.AuthorizePasswordResetClaims(values[1])
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, HTTPPlainResp{Message: "Invalid reset token"})
		return
	}
	c.Set("authClaims", claims)
	c.Next()
}

// Handles route related to receive JWT token
// @Summary Handles signin and token production
// @Description JWT can be received by signing in
// @Tags auth
// @Accept json
// @Produce json
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Success 200 {object} api.HTTPAuthorizeResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 401 {object} api.HTTPPlainResp
// @Failure 403 {object} api.HTTPPlainResp
// @Router /auth/login [post]
func login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	username = strings.TrimSpace(strings.ToLower(username))

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Username and password can not be empty",
		})
		return
	}
	if !enforceLoginRateLimit(c, username) {
		return
	}

	userEntry, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_, _ = auth.Authenticate(username, password, auth.AuthModel{
				Salt:     make([]byte, 16),
				Password: make([]byte, core.HASH_LENGTH),
			})
			c.JSON(http.StatusUnauthorized, HTTPPlainResp{Message: "The username or password is invalid"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, HTTPPlainResp{Message: "Authentication service unavailable"})
		return
	}

	jwt, err := auth.Authenticate(username, password, userEntry.AuthModel)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}
	if userEntry.Status == 1 {
		c.JSON(http.StatusForbidden, HTTPPlainResp{
			Message: "The user has been banned from this competition. Please contact competition admin for more information",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPAuthorizeResp{
		Token:   jwt,
		Role:    userEntry.Role,
		Message: "Expires in 6 hours. To access APIs send the token in header as \"Authorization: Bearer <token>\"",
	})
	return
}

// Signup
// @Summary Signup for the user
// @Description Signup route for the user
// @Tags auth
// @Produce json
// @Param name formData string false "User's name"
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Param email formData string true "User's email id"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 406 {object} api.HTTPPlainResp
// @Router /auth/register [post]
func register(c *gin.Context) {
	name := c.PostForm("name")
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")

	name = strings.TrimSpace(name)
	username = strings.TrimSpace(strings.ToLower(username))
	email = strings.TrimSpace(strings.ToLower(email))

	if username == "" || password == "" || email == "" {

		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Username, password and email can not be empty",
		})
		return
	}

	if !contestantUsernamePattern.MatchString(username) {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Username must be 3-12 lowercase letters, digits, dots, underscores, or hyphens",
		})
		return
	}
	if err := validatePassword(password); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: err.Error()})
		return
	}
	if canonical, err := canonicalMailbox(email); err != nil || canonical != email {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: "A valid email address is required"})
		return
	}

	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort
	if smtpHost == "" || smtpPort == "" {
		c.JSON(http.StatusServiceUnavailable, HTTPErrorResp{Error: "SMTP not configured"})
		return
	}
	otpEntry, err := database.QueryOTPEntry(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{Error: "OTP not found, email not verified"})
		} else {
			log.Println("Failed to query OTP:", err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to verify OTP"})
		}
		return
	}
	if !otpEntry.Verified || otpEntry.Purpose != otpPurposeRegistration || time.Now().After(otpEntry.Expiry) {
		c.JSON(http.StatusNotAcceptable, HTTPErrorResp{Error: "Email not verified, cannot register user"})
		return
	}
	authModel, err := auth.CreateModel(username, password, core.USER_ROLES["contestant"])
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to secure user credentials"})
		return
	}
	userEntry := database.User{
		Name:      name,
		AuthModel: authModel,
		Email:     email,
	}
	err = database.CreateUserEntry(&userEntry)

	if err != nil {
		c.JSON(http.StatusNotAcceptable, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}
	if err := database.DeleteOTPEntry(email); err != nil {
		log.Printf("Failed to consume registration OTP: %v", err)
	}

	markLeaderboardCachesStale()

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User created successfully",
	})
}

// ResetPasswordHandler
// @Summary Resets password for the user
// @Description Resets password for the user
// @Tags auth
// @Produce json
// @Param new_pass formData string true "New Password"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 401 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /auth/reset-password [post]
func resetPasswordHandler(c *gin.Context) {
	newPass := c.PostForm("new_pass")
	if err := validatePassword(newPass); err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{Message: err.Error()})
		return
	}

	claimsValue, exists := c.Get("authClaims")
	claims, ok := claimsValue.(*auth.CustomClaims)
	if !exists || !ok {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized user",
		})
		return
	}
	username := claims.User

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized user",
		})
		return
	}

	authModel, err := auth.CreateModel(username, newPass, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{Message: "Failed to secure user credentials"})
		return
	}
	if claims.TokenUse == auth.PasswordResetTokenUse {
		if err := database.ConsumeVerifiedOTP(user.Email, otpPurposePasswordReset, time.Now()); err != nil {
			c.JSON(http.StatusUnauthorized, HTTPPlainResp{Message: "Password reset grant is invalid or already used"})
			return
		}
	}

	err = database.UpdateUser(&user, map[string]interface{}{"Password": authModel.Password, "Salt": authModel.Salt})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Password changed successfully",
	})
}
