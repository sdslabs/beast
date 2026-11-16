package deployment

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListManagedDeploymentsHandler handles GET /api/v1/admin/deployments
func ListManagedDeploymentsHandler(c *gin.Context) {
	// TODO: implement listing all managed deployments
	httputil.RespondNotImplemented(c, "ListManagedDeploymentsHandler")
}

// GetManagedDeploymentHandler handles GET /api/v1/admin/deployments/:deployment_id
func GetManagedDeploymentHandler(c *gin.Context) {
	// TODO: implement inspection of managed deployment details
	httputil.RespondNotImplemented(c, "GetManagedDeploymentHandler")
}

// StopManagedDeploymentHandler handles POST /api/v1/admin/deployments/:deployment_id/stop
func StopManagedDeploymentHandler(c *gin.Context) {
	// TODO: implement global deployment termination & cleanup
	httputil.RespondNotImplemented(c, "StopManagedDeploymentHandler")
}

// ReplaceManagedDeploymentHandler handles POST /api/v1/admin/deployments/:deployment_id/replacements
func ReplaceManagedDeploymentHandler(c *gin.Context) {
	// TODO: implement rolling redeployment of active deployment
	httputil.RespondNotImplemented(c, "ReplaceManagedDeploymentHandler")
}
