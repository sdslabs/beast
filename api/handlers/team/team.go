package team

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListTeamsHandler handles GET /api/v1/teams
func ListTeamsHandler(c *gin.Context) {
	// TODO: implement public teams list
	httputil.RespondNotImplemented(c, "ListTeamsHandler")
}

// GetTeamHandler handles GET /api/v1/teams/:team_id
func GetTeamHandler(c *gin.Context) {
	// TODO: implement team profile inspection
	httputil.RespondNotImplemented(c, "GetTeamHandler")
}

// CreateTeamHandler handles POST /api/v1/teams
func CreateTeamHandler(c *gin.Context) {
	// TODO: implement team creation by unaffiliated user
	httputil.RespondNotImplemented(c, "CreateTeamHandler")
}
