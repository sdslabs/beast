package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/auth"
)

func authorize(c *gin.Context) {

	cookie, err := c.Cookie("JWT")

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		c.Abort()
	}

	err = auth.Authorize(cookie)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		c.Abort()
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
		"message": "Expires in 30 mins\tTo access APIs send the token as cookie \"JWT\"",
	})
	return
}

func getRandomMessage(c *gin.Context) {
	username := c.Param("username")

	encrmess, err := auth.GenerateEncryptedMessage(username)

	if err != nil {
		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"encrmess": encrmess,
	})
	return
}
