package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/handlers/hint"
	"github.com/sdslabs/beastv4/api/handlers/instance"
	"github.com/sdslabs/beastv4/api/handlers/submission"
	"github.com/sdslabs/beastv4/api/handlers/team"
	"github.com/sdslabs/beastv4/api/middleware"
)

// RegisterTeamRoutes registers team affiliation, hint unlocking, flag submissions, dynamic instances, and leader endpoints.
func RegisterTeamRoutes(userGroup *gin.RouterGroup) {
	// User without a team
	noTeamGroup := userGroup.Group("", middleware.WithoutTeamAuthorize)
	{
		noTeamGroup.POST("/teams", team.CreateTeamHandler)
		noTeamGroup.POST("/teams/join", team.JoinTeamHandler)
	}

	// Team member
	teamGroup := userGroup.Group("", middleware.TeamAuthorize)
	{
		teamGroup.POST("/teams/leave", team.LeaveTeamHandler)

		teamGroup.POST("/challenges/:challenge_slug/hints/:hint_id/unlock", middleware.IdempotencyMiddleware, hint.UnlockHintHandler)
		teamGroup.POST("/challenges/:challenge_slug/submissions", middleware.IdempotencyMiddleware, submission.SubmitFlagHandler)

		instanceGroup := teamGroup.Group("/challenges/:challenge_slug/instance")
		{
			instanceGroup.POST("/spawn", middleware.IdempotencyMiddleware, instance.SpawnInstanceHandler)
			instanceGroup.GET("", instance.GetInstanceHandler)
			instanceGroup.POST("/renew", middleware.IdempotencyMiddleware, instance.RenewInstanceHandler)
			instanceGroup.POST("/stop", middleware.IdempotencyMiddleware, instance.StopInstanceHandler)
		}

		// Team leader
		leaderGroup := teamGroup.Group("/teams/:team_id", middleware.LeaderAuthorize)
		{
			leaderGroup.DELETE("/members/:user_id", team.RemoveTeamMemberHandler)
			leaderGroup.PATCH("", team.UpdateTeamHandler)
			leaderGroup.POST("/leader", team.TransferTeamLeadershipHandler)
			leaderGroup.GET("/invitation", team.GetTeamInvitationHandler)
			leaderGroup.POST("/invitation/rotate", team.RotateTeamInvitationHandler)
		}
	}
}
