package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func dummyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": WIP_TEXT,
	})
}

func initGinRouter() *gin.Engine {
	router := gin.New()

	// API routes group
	apiGroup := router.Group("/api")
	{
		// Deploy route group
		manageGroup := apiGroup.Group("/manage")
		{
			manageGroup.GET("/all/:action", dummyHandler)
			manageGroup.POST("/deploy/local/", deployLocalChallengeHandler)
			manageGroup.POST("/challenge/", manageChallengeHandler)
			manageGroup.POST("/static/:action", beastStaticContentHandler)
		}

		// Status route group
		statusGroup := apiGroup.Group("/status")
		{
			statusGroup.GET("/challenge/:id", challengeStatusHandler)
			statusGroup.GET("/all/", statusHandler)
		}

		// Info route group
		infoGroup := apiGroup.Group("/info")
		{
			infoGroup.GET("/challenge/:id", challengeInfoHandler)
			infoGroup.GET("/available/", availableChallengeInfoHandler)
		}

		remoteGroup := apiGroup.Group("/remote")
		{
			remoteGroup.GET("/sync", syncBeastGitRemote)
			remoteGroup.GET("/reset", resetBeastGitRemote)
		}
	}

	return router
}
