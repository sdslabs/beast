package api

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
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
		authGroup.POST("/reset-password", authorize, resetPasswordHandler)
	}

	// For serving static files
	router.Use(static.Serve("/api/info/logo", static.LocalFile(
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_LOGO_DIR),
		false)),
	)
	router.GET("/api/info/competition-info", competitionInfoHandler)

	// API routes group
	apiGroup := router.Group("/api", authorize)
	{
		// Deploy route group
		manageGroup := apiGroup.Group("/manage", managerAuthorize)
		{
			manageGroup.POST("/deploy/local/", deployLocalChallengeHandler)
			manageGroup.POST("/challenge/", manageChallengeHandler)
			manageGroup.POST("/challenge/multiple/", manageMultipleChallengeHandlerNameBased)
			manageGroup.POST("/multiple/:action", manageMultipleChallengeHandlerTagBased)
			manageGroup.POST("/static/:action", beastStaticContentHandler)
			manageGroup.POST("/commit/", commitChallenge)
			manageGroup.POST("/challenge/verify", verifyHandler)
			manageGroup.POST("/schedule/:action", manageScheduledAction)
			manageGroup.POST("/challenge/upload", manageUploadHandler)
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
			infoGroup.GET("/challenge/:name", challengeInfoHandler)
			infoGroup.GET("/challenges", challengesInfoHandler)
			infoGroup.GET("/images/available", availableImagesHandler)
			infoGroup.GET("/ports/used", usedPortsInfoHandler)
			infoGroup.GET("/logs", challengeLogsHandler)
			infoGroup.GET("/user/:username", userInfoHandler)
			infoGroup.GET("/users", getAllUsersInfoHandler)
			infoGroup.GET("/submissions", submissionsHandler)
		}

		// Notification route group
		notificationGroup := apiGroup.Group("/notification")
		{
			notificationGroup.GET("/available", availableNotificationHandler)
			notificationGroup.POST("/add", adminAuthorize, addNotification)
			notificationGroup.PUT("/update", adminAuthorize, updateNotifications)
			notificationGroup.DELETE("/delete", adminAuthorize, removeNotification)
		}

		remoteGroup := apiGroup.Group("/remote", adminAuthorize)
		{
			remoteGroup.POST("/sync", syncBeastGitRemote)
			remoteGroup.POST("/reset", resetBeastGitRemote)
		}

		configGroup := apiGroup.Group("/config", adminAuthorize)
		{
			configGroup.PATCH("/reload", reloadBeastConfig)
			configGroup.POST("/competition-info", updateCompetitionInfoHandler)
			configGroup.POST("/challenge-info", updateChallengeInfoHandler)
		}

		submitGroup := apiGroup.Group("/submit")
		{
			submitGroup.POST("/challenge", submitFlagHandler)
		}

		adminPanelGroup := apiGroup.Group("/admin", adminAuthorize)
		{
			adminPanelGroup.POST("/users/:action/:id", banUserHandler)
			adminPanelGroup.GET("/statistics", getUsersStatisticsHandler)
		}
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	return router
}
