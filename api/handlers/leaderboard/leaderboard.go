package leaderboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetLeaderboardHandler handles GET /api/v1/leaderboard
func GetLeaderboardHandler(c *gin.Context) {
	// TODO: implement current leaderboard calculation
	var resp dto.LeaderboardResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetLeaderboardHistoryHandler handles GET /api/v1/leaderboard/history
func GetLeaderboardHistoryHandler(c *gin.Context) {
	var query dto.LeaderboardHistoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement historical score progression timeline
	var resp dto.LeaderboardHistoryResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
