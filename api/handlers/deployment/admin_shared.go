package deployment

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// LaunchSharedDeploymentHandler handles POST /api/v1/admin/challenges/:challenge_slug/deployments
func LaunchSharedDeploymentHandler(c *gin.Context) {
	// TODO: implement shared challenge deployment launch
	httputil.RespondNotImplemented(c, "LaunchSharedDeploymentHandler")
}

// StopChallengeDeploymentHandler handles POST /api/v1/admin/challenges/:challenge_slug/deployments/:deployment_id/stop
func StopChallengeDeploymentHandler(c *gin.Context) {
	// TODO: implement challenge deployment stop & teardown
	httputil.RespondNotImplemented(c, "StopChallengeDeploymentHandler")
}
