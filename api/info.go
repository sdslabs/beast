package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	cfg "github.com/sdslabs/beastv4/core/config"
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

func availableImagesHandler(c *gin.Context) {
	jsonString, _ := json.Marshal(cfg.Cfg.AllowedBaseImages)
	c.JSON(http.StatusOK, gin.H{
		"message": "Available docker images",
		"images":  string(jsonString),
	})
}
