package team

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminListTeamsHandler handles GET /api/v1/admin/teams
func AdminListTeamsHandler(c *gin.Context) {
	var query dto.AdminListTeamsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative teams listing
	var resp dto.AdminListTeamsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminGetTeamHandler handles GET /api/v1/admin/teams/:team_id
func AdminGetTeamHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	// TODO: implement administrative team details retrieval
	var resp dto.AdminTeamResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminUpdateTeamHandler handles PATCH /api/v1/admin/teams/:team_id
func AdminUpdateTeamHandler(c *gin.Context) {
	teamID := c.Param("team_id")
	_ = teamID

	var req dto.AdminUpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative team status, ban, or visibility modification
	var resp dto.AdminTeamResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
