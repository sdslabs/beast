package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// HealthzHandler handles GET /healthz
func HealthzHandler(c *gin.Context) {
	// TODO: implement process liveness probe
	var resp dto.HealthzResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ReadyzHandler handles GET /readyz
func ReadyzHandler(c *gin.Context) {
	// TODO: implement dependency readiness probe (PostgreSQL, Redis, disk)
	var resp dto.ReadyzResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
