package team

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// UpdateTeamHandler handles PATCH /api/v1/teams/:team_id
func UpdateTeamHandler(c *gin.Context) {
	// TODO: implement team profile update by leader
	httputil.RespondNotImplemented(c, "UpdateTeamHandler")
}

// TransferTeamLeadershipHandler handles POST /api/v1/teams/:team_id/leader
func TransferTeamLeadershipHandler(c *gin.Context) {
	// TODO: implement leadership transfer
	httputil.RespondNotImplemented(c, "TransferTeamLeadershipHandler")
}
