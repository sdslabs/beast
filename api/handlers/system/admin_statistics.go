package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminStatisticsHandler handles GET /api/v1/admin/statistics
func AdminStatisticsHandler(c *gin.Context) {
	// TODO: implement aggregate statistics calculation
	var resp dto.AdminStatisticsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
