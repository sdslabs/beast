package team

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// JoinTeamHandler handles POST /api/v1/teams/join
func JoinTeamHandler(c *gin.Context) {
	// TODO: implement joining team via invitation code
	httputil.RespondNotImplemented(c, "JoinTeamHandler")
}

// LeaveTeamHandler handles POST /api/v1/teams/leave
func LeaveTeamHandler(c *gin.Context) {
	// TODO: implement leaving current team
	httputil.RespondNotImplemented(c, "LeaveTeamHandler")
}

// RemoveTeamMemberHandler handles DELETE /api/v1/teams/:team_id/members/:user_id
func RemoveTeamMemberHandler(c *gin.Context) {
	// TODO: implement member removal by team leader
	httputil.RespondNotImplemented(c, "RemoveTeamMemberHandler")
}
