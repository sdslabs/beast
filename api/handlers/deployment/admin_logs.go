package deployment

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetManagedDeploymentLogsHandler handles GET /api/v1/admin/deployments/:deployment_id/logs
func GetManagedDeploymentLogsHandler(c *gin.Context) {
	deploymentID := c.Param("deployment_id")
	_ = deploymentID

	var query dto.ManageDeploymentLogsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement container runtime stdout/stderr logs or SSE stream
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte{})
}
