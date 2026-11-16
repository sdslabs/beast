package hint

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// UnlockHintHandler handles POST /api/v1/challenges/:challenge_slug/hints/:hint_id/unlock
func UnlockHintHandler(c *gin.Context) {
	// TODO: implement hint unlock and points deduction
	httputil.RespondNotImplemented(c, "UnlockHintHandler")
}
