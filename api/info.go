package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func challengeInfoHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func availableChallengeInfoHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}
