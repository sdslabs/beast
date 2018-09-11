package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func deployAllHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func deployChallengeHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func deployLocalChallengeHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}
