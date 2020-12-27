package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func dummyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: WIP_TEXT,
	})
}

func initGinRouter() *gin.Engine {
	router := gin.New()

	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Cookie"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// Authorization routes group
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", register)
		authGroup.POST("/login", login)
	}

	// API routes group
	apiGroup := router.Group("/api", authorize)
	{
		// Deploy route group
		manageGroup := apiGroup.Group("/manage", managerAuthorize)
		{
			manageGroup.POST("/deploy/local/", deployLocalChallengeHandler)
			manageGroup.POST("/challenge/", manageChallengeHandler)
			manageGroup.POST("/multiple/:action", manageMultipleChallengeHandler)
			manageGroup.POST("/static/:action", beastStaticContentHandler)
			manageGroup.POST("/commit/", commitChallenge)
			manageGroup.POST("/challenge/verify", verifyHandler)
			manageGroup.POST("/schedule/:action", manageScheduledAction)
		}

		// Status route group
		statusGroup := apiGroup.Group("/status")
		{
			statusGroup.GET("/challenge/:name", challengeStatusHandler)
			statusGroup.GET("/all", statusHandler)
			statusGroup.GET("/all/:filter", statusHandler)
		}

		// Info route group
		infoGroup := apiGroup.Group("/info")
		{
			infoGroup.POST("/challenge/info", challengeInfoHandler)
			infoGroup.POST("/available", availableChallengeInfoHandler)
			infoGroup.GET("/images/available", availableImagesHandler)
			infoGroup.GET("/ports/used", usedPortsInfoHandler)
			infoGroup.GET("/logs", challengeLogsHandler)
			infoGroup.GET("/challenges", challengesInfoHandler)
			infoGroup.GET("/challenges/available", availableChallengeHandler)
			infoGroup.POST("/user", userInfoHandler)
			infoGroup.GET("/user/available", getAllUsersInfoHandler)
			infoGroup.POST("/submissions", submissionsHandler)
		}

		// Notification route group
		notificationGroup := apiGroup.Group("/notification", adminAuthorize)
		{
			notificationGroup.POST("/add", addNotification)
			notificationGroup.POST("/delete", removeNotification)
			notificationGroup.POST("/update", updateNotifications)
			notificationGroup.POST("/available", availableNotificationHandler)
		}

		remoteGroup := apiGroup.Group("/remote", adminAuthorize)
		{
			remoteGroup.POST("/sync", syncBeastGitRemote)
			remoteGroup.POST("/reset", resetBeastGitRemote)
		}

		configGroup := apiGroup.Group("/config", adminAuthorize)
		{
			configGroup.PATCH("/reload", reloadBeastConfig)
		}

		submitGroup := apiGroup.Group("/submit")
		{
			submitGroup.POST("/challenge", submitFlagHandler)
		}

		adminPanelGroup := apiGroup.Group("/admin", adminAuthorize)
		{
			adminPanelGroup.POST("/users/:action/:id", banUserHandler)
			adminPanelGroup.POST("/statistics", getUsersStatisticsHandler)
		}
	}

	return router
}
