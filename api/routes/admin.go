package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/handlers/challenge"
	"github.com/sdslabs/beastv4/api/handlers/competition"
	"github.com/sdslabs/beastv4/api/handlers/deployment"
	"github.com/sdslabs/beastv4/api/handlers/hint"
	"github.com/sdslabs/beastv4/api/handlers/leaderboard"
	"github.com/sdslabs/beastv4/api/handlers/node"
	"github.com/sdslabs/beastv4/api/handlers/notification"
	"github.com/sdslabs/beastv4/api/handlers/submission"
	"github.com/sdslabs/beastv4/api/handlers/system"
	"github.com/sdslabs/beastv4/api/handlers/tag"
	"github.com/sdslabs/beastv4/api/handlers/team"
	"github.com/sdslabs/beastv4/api/handlers/user"
	"github.com/sdslabs/beastv4/api/middleware"
)

// RegisterAdminRoutes registers all administrative and management endpoints.
func RegisterAdminRoutes(adminGroup *gin.RouterGroup) {
	// Challenges management
	adminChallengeGroup := adminGroup.Group("/challenges/:challenge_slug")
	{
		adminChallengeGroup.GET("", challenge.GetManagedChallengeHandler)
		adminChallengeGroup.PATCH("", challenge.UpdateManagedChallengeHandler)
		adminChallengeGroup.DELETE("", challenge.DeleteManagedChallengeHandler)

		adminChallengeGroup.GET("/runtime", challenge.GetChallengeRuntimeHandler)

		adminChallengeGroup.GET("/instance-policy", challenge.GetChallengeInstancePolicyHandler)
		adminChallengeGroup.PUT("/instance-policy", challenge.ReplaceChallengeInstancePolicyHandler)

		adminChallengeGroup.GET("/hints", hint.ListManagedHintsHandler)
		adminChallengeGroup.POST("/hints", hint.CreateChallengeHintHandler)
		adminChallengeGroup.PATCH("/hints/:hint_id", hint.UpdateChallengeHintHandler)
		adminChallengeGroup.DELETE("/hints/:hint_id", hint.DeleteChallengeHintHandler)

		adminChallengeGroup.POST("/deployments", middleware.IdempotencyMiddleware, deployment.LaunchSharedDeploymentHandler)
		adminChallengeGroup.POST("/deployments/:deployment_id/stop", deployment.StopChallengeDeploymentHandler)
	}

	// Tags
	adminGroup.POST("/tags", tag.CreateTagHandler)
	adminGroup.PATCH("/tags/:tag_id", tag.UpdateTagHandler)
	adminGroup.DELETE("/tags/:tag_id", tag.DeleteTagHandler)

	// Deployments
	adminGroup.GET("/deployments", deployment.ListManagedDeploymentsHandler)
	adminGroup.GET("/deployments/:deployment_id", deployment.GetManagedDeploymentHandler)
	adminGroup.POST("/deployments/:deployment_id/stop", deployment.StopManagedDeploymentHandler)
	adminGroup.POST("/deployments/:deployment_id/replacements", middleware.IdempotencyMiddleware, deployment.ReplaceManagedDeploymentHandler)
	adminGroup.GET("/deployments/:deployment_id/logs", deployment.GetManagedDeploymentLogsHandler)

	// Competition
	adminGroup.GET("/competition", competition.GetAdminCompetitionHandler)
	adminGroup.PATCH("/competition", competition.UpdateAdminCompetitionHandler)
	adminGroup.PUT("/competition/logo", competition.UploadCompetitionLogoHandler)
	adminGroup.DELETE("/competition/logo", competition.DeleteCompetitionLogoHandler)

	// Users
	adminGroup.GET("/users", user.AdminListUsersHandler)
	adminGroup.POST("/users", user.AdminCreateUserHandler)
	adminGroup.GET("/users/:user_id", user.AdminGetUserHandler)
	adminGroup.PATCH("/users/:user_id", user.AdminUpdateUserHandler)

	// Teams
	adminGroup.GET("/teams", team.AdminListTeamsHandler)
	adminGroup.GET("/teams/:team_id", team.AdminGetTeamHandler)
	adminGroup.PATCH("/teams/:team_id", team.AdminUpdateTeamHandler)

	// Submissions, Stats & Audit
	adminGroup.GET("/submissions", submission.AdminListSubmissionsHandler)
	adminGroup.GET("/submissions/:submission_id", submission.AdminGetSubmissionHandler)
	adminGroup.GET("/statistics", system.AdminStatisticsHandler)
	adminGroup.GET("/audit-events", system.AdminAuditEventsHandler)

	// Leaderboard & Freeze
	adminGroup.GET("/leaderboard", leaderboard.AdminLeaderboardHandler)
	adminGroup.GET("/leaderboard/history", leaderboard.AdminLeaderboardHistoryHandler)
	adminGroup.PUT("/leaderboard/freeze", leaderboard.FreezeLeaderboardHandler)
	adminGroup.DELETE("/leaderboard/freeze", leaderboard.UnfreezeLeaderboardHandler)

	// Notifications
	adminGroup.GET("/notifications", notification.AdminListNotificationsHandler)
	adminGroup.POST("/notifications", notification.AdminCreateNotificationHandler)
	adminGroup.GET("/notifications/:notification_id", notification.AdminGetNotificationHandler)
	adminGroup.PATCH("/notifications/:notification_id", notification.AdminUpdateNotificationHandler)
	adminGroup.DELETE("/notifications/:notification_id", notification.AdminDeleteNotificationHandler)

	// Nodes
	adminGroup.GET("/nodes", node.ListNodesHandler)
	adminGroup.POST("/nodes", node.RegisterNodeHandler)
	adminGroup.GET("/nodes/:node_id", node.GetNodeHandler)
	adminGroup.PATCH("/nodes/:node_id", node.UpdateNodeHandler)
	adminGroup.POST("/nodes/:node_id/checks", node.CheckNodeHandler)
	adminGroup.POST("/nodes/:node_id/reconciliations", node.ReconcileNodeHandler)

	// System & Config
	adminGroup.GET("/system", system.AdminSystemStatusHandler)
	adminGroup.POST("/config/reload", system.ReloadConfigHandler)
}
