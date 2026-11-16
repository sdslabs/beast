package leaderboard

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// FreezeLeaderboardHandler handles PUT /api/v1/admin/leaderboard/freeze
func FreezeLeaderboardHandler(c *gin.Context) {
	// TODO: implement freezing the public scoreboard
	httputil.RespondNotImplemented(c, "FreezeLeaderboardHandler")
}

// UnfreezeLeaderboardHandler handles DELETE /api/v1/admin/leaderboard/freeze
func UnfreezeLeaderboardHandler(c *gin.Context) {
	// TODO: implement unfreezing the public scoreboard
	httputil.RespondNotImplemented(c, "UnfreezeLeaderboardHandler")
}
