package deployment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListManagedDeploymentsHandler handles GET /api/v1/admin/deployments
func ListManagedDeploymentsHandler(c *gin.Context) {
	var query dto.ManageListDeploymentsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement listing all managed deployments
	var resp dto.ManageListDeploymentsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetManagedDeploymentHandler handles GET /api/v1/admin/deployments/:deployment_id
func GetManagedDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deployment_id")
	_ = deploymentID

	// TODO: implement inspection of managed deployment details
	var resp dto.ManageDeploymentResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// StopManagedDeploymentHandler handles POST /api/v1/admin/deployments/:deployment_id/stop
func StopManagedDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deployment_id")
	_ = deploymentID

	// TODO: implement global deployment termination & cleanup
	var resp dto.ManageDeploymentResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ReplaceManagedDeploymentHandler handles POST /api/v1/admin/deployments/:deployment_id/replacements
func ReplaceManagedDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deployment_id")
	_ = deploymentID

	var req dto.ManageReplaceDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement rolling redeployment of active deployment
	var resp dto.ManageDeploymentReplacementResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
