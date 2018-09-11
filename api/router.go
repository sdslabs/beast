package api

import "github.com/gin-gonic/gin"

func initGinRouter() *gin.Engine {
	router := gin.New()

	// Depoy routes group
	deployGroup := router.Group("/deploy")
	{
		deployGroup.POST("/all/:action", deployAllHandler)
		deployGroup.POST("/challenge/:id/:action", deployChallengeHandler)
		deployGroup.POST("/local", deployLocalChallengeHandler)
	}

	// Status route group
	statusGroup := router.Group("/status")
	{
		statusGroup.GET("/challenge/:id", challengeStatusHandler)
		statusGroup.GET("/all", statusHandler)
	}

	// Status route group
	infoGroup := router.Group("/info")
	{
		infoGroup.GET("/challenge/:id", challengeInfoHandler)
		infoGroup.GET("/available", availableChallengeInfoHandler)
	}

	return router
}
