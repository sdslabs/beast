package team

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// UpdateTeamHandler handles PATCH /api/v1/teams/:team_id
func UpdateTeamHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	var req dto.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement team profile update by leader
	var resp dto.TeamProfileResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// TransferTeamLeadershipHandler handles POST /api/v1/teams/:team_id/leader
func TransferTeamLeadershipHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	var req dto.TransferLeadershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement leadership transfer
	var resp dto.TeamProfileResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
