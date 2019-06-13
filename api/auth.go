package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/auth"
	"github.com/sdslabs/beastv4/core/config"
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
// @Summary Handles solution check and token production
// @Description JWT can be received by sending back correct answer to challenge
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} api.HTTPAuthorizeResp
// @Failure 401 {object} api.HTTPPlainResp
// @Router /auth/:username [post]
func getJWT(c *gin.Context) {
	username := c.Param("username")
	decrMess := c.PostForm("decrmess")

	jwt, err := auth.GenerateJWT(username, decrMess)

	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, HTTPAuthorizeResp{
		Token:   jwt,
		Message: "Expires in 6 hours\nTo access APIs send the token in header as \"Authorization: Bearer <token>\"",
	})
	return
}

// Handles route related to getting user challenge for authorization
// @Summary Handles challenge generation
// @Description Sends challenge for authorization of user
// @Tags auth
// @Produce json
// @Success 200 {object} api.AuthorizationChallengeResp
// @Failure 406 {object} api.HTTPPlainResp
// @Router /auth/:username [get]
func getAuthChallenge(c *gin.Context) {
	username := c.Param("username")

	challenge, err := auth.GenerateAuthChallenge(username)

	if err != nil {
		c.JSON(http.StatusNotAcceptable, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AuthorizationChallengeResp{
		Challenge: []byte(challenge),
		Message:   "Solve the above challenge and POST to this route to get AUTHORIZATION KEY",
	})
	return
}
