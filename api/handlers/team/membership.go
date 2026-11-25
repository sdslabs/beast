package team

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// JoinTeamHandler handles POST /api/v1/teams/join
func JoinTeamHandler(c *gin.Context) {
	var req dto.JoinTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement joining team via invitation code
	var resp dto.TeamProfileResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// LeaveTeamHandler handles POST /api/v1/teams/leave
func LeaveTeamHandler(c *gin.Context) {
	// TODO: implement leaving current team
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// RemoveTeamMemberHandler handles DELETE /api/v1/teams/:team_id/members/:user_id
func RemoveTeamMemberHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	userID := c.Param("user_id")
	_ = teamID
	_ = userID

	// TODO: implement member removal by team leader
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
