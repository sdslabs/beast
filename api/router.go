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

	// Authorization routes group
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/:username", getAuthChallenge)
		authGroup.POST("/:username", getJWT)
	}

	// API routes group
	apiGroup := router.Group("/api", authorize)
	{
		// Deploy route group
		manageGroup := apiGroup.Group("/manage")
		{
			manageGroup.POST("/deploy/local/", deployLocalChallengeHandler)
			manageGroup.POST("/challenge/", manageChallengeHandler)
			manageGroup.POST("/multiple/:action", manageMultipleChallengeHandler)
			manageGroup.POST("/static/:action", beastStaticContentHandler)
			manageGroup.POST("/commit/", commitChallenge)
			manageGroup.GET("/logs/", challengeLogsHandler)
		}

		// Status route group
		statusGroup := apiGroup.Group("/status")
		{
			statusGroup.GET("/challenge/:name", challengeStatusHandler)
			statusGroup.GET("/all", statusHandler)
			statusGroup.GET("/all/:filter", statusHandler)
			statusGroup.GET("/challenge/description", challengeDescriptionHandler)
		}

		// Info route group
		infoGroup := apiGroup.Group("/info")
		{
			infoGroup.GET("/challenge/:id", challengeInfoHandler)
			infoGroup.GET("/available", availableChallengeInfoHandler)
			infoGroup.GET("/images/available", availableImagesHandler)
			infoGroup.GET("/ports/used", usedPortsInfoHandler)
		}

		remoteGroup := apiGroup.Group("/remote")
		{
			remoteGroup.POST("/sync", syncBeastGitRemote)
			remoteGroup.POST("/reset", resetBeastGitRemote)
		}

		configGroup := apiGroup.Group("/config")
		{
			configGroup.PATCH("/reload", reloadBeastConfig)
		}
	}

	return router
}
