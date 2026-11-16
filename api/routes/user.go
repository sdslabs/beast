package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/handlers/challenge"
	"github.com/sdslabs/beastv4/api/handlers/leaderboard"
	"github.com/sdslabs/beastv4/api/handlers/notification"
	"github.com/sdslabs/beastv4/api/handlers/submission"
	"github.com/sdslabs/beastv4/api/handlers/tag"
	"github.com/sdslabs/beastv4/api/handlers/team"
	"github.com/sdslabs/beastv4/api/handlers/user"
)

// RegisterUserRoutes registers participant read-only routes and personal notification endpoints.
func RegisterUserRoutes(userGroup *gin.RouterGroup) {
	userGroup.GET("/users", user.ListUsersHandler)
	userGroup.GET("/users/:user_id", user.GetUserHandler)

	userGroup.GET("/teams", team.ListTeamsHandler)
	userGroup.GET("/teams/:team_id", team.GetTeamHandler)

	userGroup.GET("/challenges", challenge.ListChallengesHandler)
	userGroup.GET("/challenges/:challenge_slug", challenge.GetChallengeHandler)
	userGroup.GET("/challenges/:challenge_slug/files/:file_id", challenge.GetChallengeFileHandler)
	userGroup.GET("/tags", tag.ListTagsHandler)

	userGroup.GET("/solves", submission.ListSolvesHandler)
	userGroup.GET("/leaderboard", leaderboard.GetLeaderboardHandler)
	userGroup.GET("/leaderboard/history", leaderboard.GetLeaderboardHistoryHandler)

	notificationsGroup := userGroup.Group("/notifications")
	{
		notificationsGroup.GET("", notification.ListNotificationsHandler)
		notificationsGroup.GET("/stream", notification.StreamNotificationsHandler)
		notificationsGroup.POST("/read-all", notification.MarkAllNotificationsReadHandler)
		notificationsGroup.GET("/:notification_id", notification.GetNotificationHandler)
		notificationsGroup.PUT("/:notification_id/read", notification.MarkNotificationReadHandler)
	}
}
