package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
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

	err := auth.Authorize(values[1])

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
// @Router /auth/signin [post]
func signin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	userEntry, err := database.QueryFirstUserEntry("username", username)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
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
// @Router /auth/signup [post]
func signup(c *gin.Context) {

	name := c.PostForm("name")
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	sshKey := c.PostForm("ssh-key")

	userEntry := database.User{
		Name:      name,
		AuthModel: auth.CreateModel(username, password, core.USER_ROLES["maintainer"]),
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
