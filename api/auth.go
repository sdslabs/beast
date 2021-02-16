package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
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
// @Success 200 {object} api.HTTPAuthorizeResp
// @Failure 401 {object} api.HTTPPlainResp
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
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
	}

	if userEntry.Status == 1 {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
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
		Message: "Expires in 6 hours. To access APIs send the token in header as \"Authorization: Bearer <token>\"",
	})
	return
}

// Signup
// @Summary Signup for the user
// @Description Signup route for the user
// @Tags auth
// @Produce json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 406 {object} api.HTTPPlainResp
// @Router /auth/register [post]
func register(c *gin.Context) {

	name := c.PostForm("name")
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	sshKey := c.PostForm("ssh-key")

	if username == "" || password == "" || email == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Username ,password and email can not be empty",
		})
	}

	userEntry := database.User{
		Name:      name,
		AuthModel: auth.CreateModel(username, password, core.USER_ROLES["contestant"]),
		Email:     email,
		SshKey:    sshKey,
	}

	err := database.CreateUserEntry(&userEntry)

	if err != nil {
		c.JSON(http.StatusNotAcceptable, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User created successfully",
	})
	return
}

// ResetPasswordHandler
// @Summary Resets password for the user
// @Description Resets password for the user
// @Tags auth
// @Produce json
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
	return
}
