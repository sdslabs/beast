package api

import (
	"crypto/sha256"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/auth"
	"github.com/sdslabs/beastv4/database"
)

func signUpHandler(c *gin.Context) {
	username := c.Param("username")
	useremail := c.Param("useremail")
	password := c.Param("password")
	encryptedPass := sha256.Sum256([]byte(password))
	NewUserEntry := database.UserDetail{
		UserName:   username,
		UserEmail:  useremail,
		Password:   encryptedPass,
		TotalScore: 0,
	}
	err := database.AddUser(&NewUserEntry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while adding User or UserName exists.",
		})
		return
	}
}

func signInHandler(c *gin.Context) {
	username := c.Param("username")
	password := c.Param("password")
	decrMess := c.PostForm("decrmess")
	encryptedPass := sha256.Sum256([]byte(password))
	if username == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Name of the challenge is a required parameter to process request.",
		})
		return
	}
	userdetail, err := database.QueryUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	if userdetail[0].Password == encryptedPass {
		jwt, err := auth.GenerateUserJWT(username, decrMess)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":   jwt,
			"message": "Expires in 6 hours\nTo access APIs send the token in header as \"Authorization: Bearer <token>\"",
		})
		return

	} else {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Name of the challenge is a required parameter to process request.",
		})
		return
	}
}
