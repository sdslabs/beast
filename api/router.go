package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func initGinRouter() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(requestIDMiddleware())
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Skip: func(c *gin.Context) bool {
			return !isPayloadLoggingEnabled() && c.Writer.Status() < 400
		},
	}))
	router.Use(requestResponseLogger())

	corsConfig := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Cookie",
			"Idempotency-Key",
			"Last-Event-ID",
			"X-Request-ID",
		},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	router.GET("/healthz", healthzHandler)
	router.GET("/readyz", operatorAuthorize, readyzHandler)
	router.GET("/openapi.json", docsAuthorize, openAPISpecHandler)
	router.GET("/docs", docsAuthorize, docsUIHandler)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/competition", getCompetitionHandler)
		v1.GET("/competition/logo", getCompetitionLogoHandler)

		authPublic := v1.Group("/auth")
		{
			authPublic.POST("/register", registerHandler)
			authPublic.POST("/login", loginHandler)
			authPublic.POST("/refresh", refreshTokenHandler)
			authPublic.POST("/password-reset/request", requestPasswordResetHandler)
			authPublic.POST("/password-reset/confirm", confirmPasswordResetHandler)
		}

		accountGroup := v1.Group("", accountAuthorize)
		{
			authAccount := accountGroup.Group("/auth")
			{
				authAccount.POST("/logout", logoutHandler)
				authAccount.POST("/email-verification/request", requestEmailVerificationHandler)
				authAccount.POST("/email-verification/confirm", confirmEmailVerificationHandler)
			}

			meAccount := accountGroup.Group("/me")
			{
				meAccount.GET("", getMeHandler)
				meAccount.PATCH("", updateMeHandler)
				meAccount.POST("/password", changePasswordHandler)
			}

			userGroup := accountGroup.Group("", userAuthorize)
			{
				userGroup.GET("/users", listUsersHandler)
				userGroup.GET("/users/:user_id", getUserHandler)

				userGroup.GET("/teams", listTeamsHandler)
				userGroup.GET("/teams/:team_id", getTeamHandler)

				userGroup.GET("/challenges", listChallengesHandler)
				userGroup.GET("/challenges/:challenge_slug", getChallengeHandler)
				userGroup.GET("/challenges/:challenge_slug/files/:file_id", getChallengeFileHandler)
				userGroup.GET("/tags", listTagsHandler)

				userGroup.GET("/solves", listSolvesHandler)
				userGroup.GET("/leaderboard", getLeaderboardHandler)
				userGroup.GET("/leaderboard/history", getLeaderboardHistoryHandler)

				notificationsGroup := userGroup.Group("/notifications")
				{
					notificationsGroup.GET("", listNotificationsHandler)
					notificationsGroup.GET("/stream", streamNotificationsHandler)
					notificationsGroup.POST("/read-all", markAllNotificationsReadHandler)
					notificationsGroup.GET("/:notification_id", getNotificationHandler)
					notificationsGroup.PUT("/:notification_id/read", markNotificationReadHandler)
				}

				noTeamGroup := userGroup.Group("", withoutTeamAuthorize)
				{
					noTeamGroup.POST("/teams", createTeamHandler)
					noTeamGroup.POST("/teams/join", joinTeamHandler)
				}

				teamGroup := userGroup.Group("", teamAuthorize)
				{
					teamGroup.POST("/teams/leave", leaveTeamHandler)

					teamGroup.POST("/challenges/:challenge_slug/hints/:hint_id/unlock", idempotencyMiddleware, unlockHintHandler)
					teamGroup.POST("/challenges/:challenge_slug/submissions", idempotencyMiddleware, submitFlagHandler)

					instanceGroup := teamGroup.Group("/challenges/:challenge_slug/instance")
					{
						instanceGroup.POST("/spawn", idempotencyMiddleware, spawnInstanceHandler)
						instanceGroup.GET("", getInstanceHandler)
						instanceGroup.POST("/renew", idempotencyMiddleware, renewInstanceHandler)
						instanceGroup.POST("/stop", idempotencyMiddleware, stopInstanceHandler)
					}

					leaderGroup := teamGroup.Group("/teams/:team_id", leaderAuthorize)
					{
						leaderGroup.DELETE("/members/:user_id", removeTeamMemberHandler)
						leaderGroup.PATCH("", updateTeamHandler)
						leaderGroup.POST("/leader", transferTeamLeadershipHandler)
						leaderGroup.GET("/invitation", getTeamInvitationHandler)
						leaderGroup.POST("/invitation/rotate", rotateTeamInvitationHandler)
					}
				}
			}

			adminGroup := accountGroup.Group("/admin", adminAuthorize)
			{
				adminChallengeGroup := adminGroup.Group("/challenges/:challenge_slug")
				{
					adminChallengeGroup.GET("", getManagedChallengeHandler)
					adminChallengeGroup.PATCH("", updateManagedChallengeHandler)
					adminChallengeGroup.DELETE("", deleteManagedChallengeHandler)

					adminChallengeGroup.GET("/runtime", getChallengeRuntimeHandler)

					adminChallengeGroup.GET("/instance-policy", getChallengeInstancePolicyHandler)
					adminChallengeGroup.PUT("/instance-policy", replaceChallengeInstancePolicyHandler)

					adminChallengeGroup.GET("/hints", listManagedHintsHandler)
					adminChallengeGroup.POST("/hints", createChallengeHintHandler)
					adminChallengeGroup.PATCH("/hints/:hint_id", updateChallengeHintHandler)
					adminChallengeGroup.DELETE("/hints/:hint_id", deleteChallengeHintHandler)

					adminChallengeGroup.POST("/deployments", idempotencyMiddleware, launchSharedDeploymentHandler)
					adminChallengeGroup.POST("/deployments/:deployment_id/stop", stopChallengeDeploymentHandler)
				}

				adminGroup.POST("/tags", createTagHandler)
				adminGroup.PATCH("/tags/:tag_id", updateTagHandler)
				adminGroup.DELETE("/tags/:tag_id", deleteTagHandler)

				adminGroup.GET("/deployments", listManagedDeploymentsHandler)
				adminGroup.GET("/deployments/:deployment_id", getManagedDeploymentHandler)
				adminGroup.POST("/deployments/:deployment_id/stop", stopManagedDeploymentHandler)
				adminGroup.POST("/deployments/:deployment_id/replacements", idempotencyMiddleware, replaceManagedDeploymentHandler)
				adminGroup.GET("/deployments/:deployment_id/logs", getManagedDeploymentLogsHandler)

				adminGroup.GET("/competition", getAdminCompetitionHandler)
				adminGroup.PATCH("/competition", updateAdminCompetitionHandler)
				adminGroup.PUT("/competition/logo", uploadCompetitionLogoHandler)
				adminGroup.DELETE("/competition/logo", deleteCompetitionLogoHandler)

				adminGroup.GET("/users", adminListUsersHandler)
				adminGroup.POST("/users", adminCreateUserHandler)
				adminGroup.GET("/users/:user_id", adminGetUserHandler)
				adminGroup.PATCH("/users/:user_id", adminUpdateUserHandler)

				adminGroup.GET("/teams", adminListTeamsHandler)
				adminGroup.GET("/teams/:team_id", adminGetTeamHandler)
				adminGroup.PATCH("/teams/:team_id", adminUpdateTeamHandler)

				adminGroup.GET("/submissions", adminListSubmissionsHandler)
				adminGroup.GET("/submissions/:submission_id", adminGetSubmissionHandler)
				adminGroup.GET("/statistics", adminStatisticsHandler)
				adminGroup.GET("/audit-events", adminAuditEventsHandler)

				adminGroup.GET("/leaderboard", adminLeaderboardHandler)
				adminGroup.GET("/leaderboard/history", adminLeaderboardHistoryHandler)
				adminGroup.PUT("/leaderboard/freeze", freezeLeaderboardHandler)
				adminGroup.DELETE("/leaderboard/freeze", unfreezeLeaderboardHandler)

				adminGroup.GET("/notifications", adminListNotificationsHandler)
				adminGroup.POST("/notifications", adminCreateNotificationHandler)
				adminGroup.GET("/notifications/:notification_id", adminGetNotificationHandler)
				adminGroup.PATCH("/notifications/:notification_id", adminUpdateNotificationHandler)
				adminGroup.DELETE("/notifications/:notification_id", adminDeleteNotificationHandler)

				adminGroup.GET("/nodes", listNodesHandler)
				adminGroup.POST("/nodes", registerNodeHandler)
				adminGroup.GET("/nodes/:node_id", getNodeHandler)
				adminGroup.PATCH("/nodes/:node_id", updateNodeHandler)
				adminGroup.POST("/nodes/:node_id/checks", checkNodeHandler)
				adminGroup.POST("/nodes/:node_id/reconciliations", reconcileNodeHandler)

				adminGroup.GET("/system", adminSystemStatusHandler)
				adminGroup.POST("/config/reload", reloadConfigHandler)
			}
		}
	}

	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":       "PAGE_NOT_FOUND",
			"message":    "Page not found",
			"request_id": c.GetString("request_id"),
		})
	})

	return router
}
