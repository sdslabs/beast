package challenge

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListChallengesHandler handles GET /api/v1/challenges
func ListChallengesHandler(c *gin.Context) {
	var query dto.ListChallengesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement challenges catalog listing
	var resp dto.ListChallengesResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetChallengeHandler handles GET /api/v1/challenges/:challenge_slug
func GetChallengeHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement detailed challenge inspection with hints catalog
	var resp dto.ChallengeDetailResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
