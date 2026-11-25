package competition

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetCompetitionHandler handles GET /api/v1/competition
func GetCompetitionHandler(c *gin.Context) {
	// TODO: implement public competition configuration retrieval
	var resp dto.CompetitionPublicResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetCompetitionLogoHandler handles GET /api/v1/competition/logo
func GetCompetitionLogoHandler(c *gin.Context) {
	// TODO: implement public competition logo download
	c.Data(http.StatusOK, "image/png", []byte{})
}
