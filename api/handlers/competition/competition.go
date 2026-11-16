package competition

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetCompetitionHandler handles GET /api/v1/competition
func GetCompetitionHandler(c *gin.Context) {
	// TODO: implement public competition configuration retrieval
	httputil.RespondNotImplemented(c, "GetCompetitionHandler")
}

// GetCompetitionLogoHandler handles GET /api/v1/competition/logo
func GetCompetitionLogoHandler(c *gin.Context) {
	// TODO: implement public competition logo download
	httputil.RespondNotImplemented(c, "GetCompetitionLogoHandler")
}
