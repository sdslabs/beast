package hint

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// UnlockHintHandler handles POST /api/v1/challenges/:challenge_slug/hints/:hint_id/unlock
func UnlockHintHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	hintID := c.Param("hint_id")
	_ = slug
	_ = hintID

	// TODO: implement hint unlock and points deduction
	var resp dto.UnlockedHintResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
