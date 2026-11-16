package team

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetTeamInvitationHandler handles GET /api/v1/teams/:team_id/invitation
func GetTeamInvitationHandler(c *gin.Context) {
	// TODO: implement inspection of current team invitation code
	httputil.RespondNotImplemented(c, "GetTeamInvitationHandler")
}

// RotateTeamInvitationHandler handles POST /api/v1/teams/:team_id/invitation/rotate
func RotateTeamInvitationHandler(c *gin.Context) {
	// TODO: implement team invitation code rotation
	httputil.RespondNotImplemented(c, "RotateTeamInvitationHandler")
}
