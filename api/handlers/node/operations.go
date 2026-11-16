package node

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// CheckNodeHandler handles POST /api/v1/admin/nodes/:node_id/checks
func CheckNodeHandler(c *gin.Context) {
	// TODO: implement on-demand compute node health probe
	httputil.RespondNotImplemented(c, "CheckNodeHandler")
}

// ReconcileNodeHandler handles POST /api/v1/admin/nodes/:node_id/reconciliations
func ReconcileNodeHandler(c *gin.Context) {
	// TODO: implement node container reconciliation
	httputil.RespondNotImplemented(c, "ReconcileNodeHandler")
}
