package challenge

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetChallengeRuntimeHandler handles GET /api/v1/admin/challenges/:challenge_slug/runtime
func GetChallengeRuntimeHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement container runtime configuration inspection
	var resp dto.ManageRuntimeConfigResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
