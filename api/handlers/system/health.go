package system

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// HealthzHandler handles GET /healthz
func HealthzHandler(c *gin.Context) {
	// TODO: implement process liveness probe
	httputil.RespondNotImplemented(c, "HealthzHandler")
}

// ReadyzHandler handles GET /readyz
func ReadyzHandler(c *gin.Context) {
	// TODO: implement dependency readiness probe (PostgreSQL, Redis, disk)
	httputil.RespondNotImplemented(c, "ReadyzHandler")
}
