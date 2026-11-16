package leaderboard

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetLeaderboardHandler handles GET /api/v1/leaderboard
func GetLeaderboardHandler(c *gin.Context) {
	// TODO: implement current leaderboard calculation
	httputil.RespondNotImplemented(c, "GetLeaderboardHandler")
}

// GetLeaderboardHistoryHandler handles GET /api/v1/leaderboard/history
func GetLeaderboardHistoryHandler(c *gin.Context) {
	// TODO: implement historical score progression timeline
	httputil.RespondNotImplemented(c, "GetLeaderboardHistoryHandler")
}
