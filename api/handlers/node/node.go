package node

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListNodesHandler handles GET /api/v1/admin/nodes
func ListNodesHandler(c *gin.Context) {
	// TODO: implement compute nodes listing
	var resp dto.AdminListNodesResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// RegisterNodeHandler handles POST /api/v1/admin/nodes
func RegisterNodeHandler(c *gin.Context) {
	var req dto.AdminRegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement compute node registration
	var resp dto.AdminNodeResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}

// GetNodeHandler handles GET /api/v1/admin/nodes/:node_id
func GetNodeHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	_ = nodeID

	// TODO: implement node details and status inspection
	var resp dto.AdminNodeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// UpdateNodeHandler handles PATCH /api/v1/admin/nodes/:node_id
func UpdateNodeHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	_ = nodeID

	var req dto.AdminUpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement node metadata, capacity, or drain status update
	var resp dto.AdminNodeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
