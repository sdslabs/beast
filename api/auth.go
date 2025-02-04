package api

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
)

// Acts as a middleware to authorize user
// @Summary Handles authorization of user
// @Description Authorizes user by checking if JWT token exists and is valid
// @Tags auth
// @Accept json
// @Produce json
// @Failure 401 {object} api.HTTPPlainResp
// @Security ApiKeyAuth
func authorize(c *gin.Context) {
	if config.SkipAuthorization {
		return
	}

	authHeader := c.GetHeader("Authorization")

	values := strings.Split(authHeader, " ")

	if len(values) < 2 || values[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "No Token Provided",
		})
		c.Abort()
		return
	}

	err := auth.Authorize(values[1], core.MANAGER|core.ADMIN|core.USER)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
		c.Abort()
		return
	}

	c.Next()
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
	if config.SkipAuthorization {
		return
	}

	authHeader := c.GetHeader("Authorization")

	values := strings.Split(authHeader, " ")

	if len(values) < 2 || values[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "No Token Provided",
		})
		c.Abort()
		return
	}

	err := auth.Authorize(values[1], core.MANAGER|core.ADMIN)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
		c.Abort()
		return
	}

	c.Next()
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
	if config.SkipAuthorization {
		return
	}

	authHeader := c.GetHeader("Authorization")

	values := strings.Split(authHeader, " ")

	if len(values) < 2 || values[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "No Token Provided",
		})
		c.Abort()
		return
	}

	err := auth.Authorize(values[1], core.ADMIN)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
		c.Abort()
		return
	}

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

	if username == "" || password == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Username and password can not be empty",
		})
	}

	userEntry, err := database.QueryFirstUserEntry("username", username)

	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
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

	jwt, err := auth.Authenticate(username, password, userEntry.AuthModel)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
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
// @Param ssh-key formData string false "User's ssh-key"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 406 {object} api.HTTPPlainResp
// @Router /auth/register [post]
func register(c *gin.Context) {
	name := c.PostForm("name")
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	sshKey := c.PostForm("ssh-key")

	if username == "" || password == "" || email == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Username ,password and email can not be empty",
		})
		return
	}

	if len(username) > 12 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Username cannot be greater than 12 characters",
		})
		return
	}

	re := regexp.MustCompile(`^.*@.*iitr\.ac\.in$`)
	isIITR := re.MatchString(email)

	if !isIITR {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Email should be of IITR domain",
		})
		return
	}

	userEntry := database.User{
		Name:      name,
		AuthModel: auth.CreateModel(username, password, core.USER_ROLES["contestant"]),
		Email:     email,
		SshKey:    sshKey,
	}

	otpEntry, err := database.QueryOTPEntry(email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "OTP not found, email not verified",
			})
			return
		} else {
			log.Println("Failed to query OTP:", err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "Failed to send OTP",
			})
			return
		}
	}

	if !otpEntry.Verified {
		c.JSON(http.StatusNotAcceptable, HTTPErrorResp{
			Error: "Email not verified, cannot register user",
		})
		return
	}

	err = database.CreateUserEntry(&userEntry)

	if err != nil {
		c.JSON(http.StatusNotAcceptable, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

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

	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized user",
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized user",
		})
	}

	authModel := auth.CreateModel(username, newPass, user.Role)

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
