package leaderboard

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminLeaderboardHandler handles GET /api/v1/admin/leaderboard
func AdminLeaderboardHandler(c *gin.Context) {
	// TODO: implement administrative unfrozen leaderboard inspection
	httputil.RespondNotImplemented(c, "AdminLeaderboardHandler")
}

// AdminLeaderboardHistoryHandler handles GET /api/v1/admin/leaderboard/history
func AdminLeaderboardHistoryHandler(c *gin.Context) {
	// TODO: implement administrative unfrozen historical leaderboard data
	httputil.RespondNotImplemented(c, "AdminLeaderboardHistoryHandler")
}
