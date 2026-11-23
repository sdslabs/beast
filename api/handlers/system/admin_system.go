package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminSystemStatusHandler handles GET /api/v1/admin/system
func AdminSystemStatusHandler(c *gin.Context) {
	// TODO: implement system resource & status inspection
	var resp dto.AdminSystemStatusResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ReloadConfigHandler handles POST /api/v1/admin/config/reload
func ReloadConfigHandler(c *gin.Context) {
	// TODO: implement server config reload
	var resp dto.AdminConfigReloadResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
