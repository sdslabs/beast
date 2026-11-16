package challenge

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetChallengeInstancePolicyHandler handles GET /api/v1/admin/challenges/:challenge_slug/instance-policy
func GetChallengeInstancePolicyHandler(c *gin.Context) {
	// TODO: implement instance lifetime policy inspection
	httputil.RespondNotImplemented(c, "GetChallengeInstancePolicyHandler")
}

// ReplaceChallengeInstancePolicyHandler handles PUT /api/v1/admin/challenges/:challenge_slug/instance-policy
func ReplaceChallengeInstancePolicyHandler(c *gin.Context) {
	// TODO: implement instance lifetime policy replacement
	httputil.RespondNotImplemented(c, "ReplaceChallengeInstancePolicyHandler")
}
