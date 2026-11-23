package node

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// CheckNodeHandler handles POST /api/v1/admin/nodes/:node_id/checks
func CheckNodeHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	_ = nodeID

	// TODO: implement on-demand compute node health probe
	var resp dto.AdminNodeCheckResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ReconcileNodeHandler handles POST /api/v1/admin/nodes/:node_id/reconciliations
func ReconcileNodeHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	_ = nodeID

	// TODO: implement node container reconciliation
	var resp dto.AdminNodeReconciliationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
