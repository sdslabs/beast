package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func challengeInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": WIP_TEXT,
	})
}

func availableChallengeInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": WIP_TEXT,
	})
}
