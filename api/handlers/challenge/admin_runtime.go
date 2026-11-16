package challenge

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetChallengeRuntimeHandler handles GET /api/v1/admin/challenges/:challenge_slug/runtime
func GetChallengeRuntimeHandler(c *gin.Context) {
	// TODO: implement container runtime configuration inspection
	httputil.RespondNotImplemented(c, "GetChallengeRuntimeHandler")
}
