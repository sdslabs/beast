package team

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListTeamsHandler handles GET /api/v1/teams
func ListTeamsHandler(c *gin.Context) {
	var query dto.ListTeamsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement public teams list
	var resp dto.ListTeamsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetTeamHandler handles GET /api/v1/teams/:team_id
func GetTeamHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	// TODO: implement team profile inspection
	var resp dto.TeamProfileResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// CreateTeamHandler handles POST /api/v1/teams
func CreateTeamHandler(c *gin.Context) {
	var req dto.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement team creation by unaffiliated user
	var resp dto.TeamProfileResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}
