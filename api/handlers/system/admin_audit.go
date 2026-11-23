package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminAuditEventsHandler handles GET /api/v1/admin/audit-events
func AdminAuditEventsHandler(c *gin.Context) {
	var query dto.AdminListAuditEventsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative audit events listing
	var resp dto.AdminListAuditEventsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
