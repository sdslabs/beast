package team

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetTeamInvitationHandler handles GET /api/v1/teams/:team_id/invitation
func GetTeamInvitationHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	// TODO: implement inspection of current team invitation code
	var resp dto.TeamInvitationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// RotateTeamInvitationHandler handles POST /api/v1/teams/:team_id/invitation/rotate
func RotateTeamInvitationHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	// TODO: implement team invitation code rotation
	var resp dto.TeamInvitationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
