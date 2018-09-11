package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func challengeStatusHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func statusHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}
