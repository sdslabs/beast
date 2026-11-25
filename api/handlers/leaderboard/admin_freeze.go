package leaderboard

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// FreezeLeaderboardHandler handles PUT /api/v1/admin/leaderboard/freeze
func FreezeLeaderboardHandler(c *gin.Context) {
	var req dto.AdminFreezeLeaderboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement freezing the public scoreboard
	var resp dto.LeaderboardFreezeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// UnfreezeLeaderboardHandler handles DELETE /api/v1/admin/leaderboard/freeze
func UnfreezeLeaderboardHandler(c *gin.Context) {
	// TODO: implement unfreezing the public scoreboard
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
