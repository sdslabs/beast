package submission

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// SubmitFlagHandler handles POST /api/v1/challenges/:challenge_slug/submissions
func SubmitFlagHandler(c *gin.Context) {
	// TODO: implement flag validation, solve recording, and dynamic score updating
	httputil.RespondNotImplemented(c, "SubmitFlagHandler")
}
