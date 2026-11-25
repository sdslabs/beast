package deployment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// LaunchSharedDeploymentHandler handles POST /api/v1/admin/challenges/:challenge_slug/deployments
func LaunchSharedDeploymentHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	var req dto.ManageLaunchSharedDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Optional body
	}

	// TODO: implement shared challenge deployment launch
	var resp dto.ManageDeploymentResponse
	httputil.RespondJSON(c, http.StatusAccepted, resp)
}

// StopChallengeDeploymentHandler handles POST /api/v1/admin/challenges/:challenge_slug/deployments/:deployment_id/stop
func StopChallengeDeploymentHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	deploymentID := c.Param("deployment_id")
	_ = slug
	_ = deploymentID

	// TODO: implement challenge deployment stop & teardown
	var resp dto.ManageDeploymentResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
