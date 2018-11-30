package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/auth"
)

func authorize(c *gin.Context) {

	authHeader := c.GetHeader("Authorization")

	values := strings.Split(authHeader, " ")

	if len(values) < 2 || values[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": fmt.Errorf("No Token Provided"),
		})
		c.Abort()
		return
	}

	err := auth.Authorize(values[1])

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		c.Abort()
		return
	}

	c.Next()
}

func getJWT(c *gin.Context) {
	username := c.Param("username")
	decrMess := c.PostForm("decrmess")

	jwt, err := auth.GenerateJWT(username, decrMess)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":   jwt,
		"message": "Expires in 6 hours\tTo access APIs send the token in header as \"Authorization: Bearer <token>\"",
	})
	return
}

func getAuthChallenge(c *gin.Context) {
	username := c.Param("username")

	challenge, err := auth.GenerateAuthChallenge(username)

	if err != nil {
		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"challenge": challenge,
	})
	return
}
