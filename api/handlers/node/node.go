package node

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListNodesHandler handles GET /api/v1/admin/nodes
func ListNodesHandler(c *gin.Context) {
	// TODO: implement compute nodes listing
	httputil.RespondNotImplemented(c, "ListNodesHandler")
}

// RegisterNodeHandler handles POST /api/v1/admin/nodes
func RegisterNodeHandler(c *gin.Context) {
	// TODO: implement compute node registration
	httputil.RespondNotImplemented(c, "RegisterNodeHandler")
}

// GetNodeHandler handles GET /api/v1/admin/nodes/:node_id
func GetNodeHandler(c *gin.Context) {
	// TODO: implement node details and status inspection
	httputil.RespondNotImplemented(c, "GetNodeHandler")
}

// UpdateNodeHandler handles PATCH /api/v1/admin/nodes/:node_id
func UpdateNodeHandler(c *gin.Context) {
	// TODO: implement node metadata, capacity, or drain status update
	httputil.RespondNotImplemented(c, "UpdateNodeHandler")
}
