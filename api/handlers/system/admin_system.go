package system

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminSystemStatusHandler handles GET /api/v1/admin/system
func AdminSystemStatusHandler(c *gin.Context) {
	// TODO: implement system resource & status inspection
	httputil.RespondNotImplemented(c, "AdminSystemStatusHandler")
}

// ReloadConfigHandler handles POST /api/v1/admin/config/reload
func ReloadConfigHandler(c *gin.Context) {
	// TODO: implement server config reload
	httputil.RespondNotImplemented(c, "ReloadConfigHandler")
}
