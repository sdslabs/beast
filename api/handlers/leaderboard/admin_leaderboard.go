package leaderboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminLeaderboardHandler handles GET /api/v1/admin/leaderboard
func AdminLeaderboardHandler(c *gin.Context) {
	var query dto.AdminLeaderboardQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative unfrozen leaderboard inspection
	var resp dto.LeaderboardResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminLeaderboardHistoryHandler handles GET /api/v1/admin/leaderboard/history
func AdminLeaderboardHistoryHandler(c *gin.Context) {
	var query dto.LeaderboardHistoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative unfrozen historical leaderboard data
	var resp dto.LeaderboardHistoryResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
