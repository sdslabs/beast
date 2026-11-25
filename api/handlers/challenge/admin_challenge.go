package challenge

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetManagedChallengeHandler handles GET /api/v1/admin/challenges/:challenge_slug
func GetManagedChallengeHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement administrative challenge metadata inspection
	var resp dto.ManageChallengeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// UpdateManagedChallengeHandler handles PATCH /api/v1/admin/challenges/:challenge_slug
func UpdateManagedChallengeHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	var req dto.ManageUpdateChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative challenge update
	var resp dto.ManageChallengeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// DeleteManagedChallengeHandler handles DELETE /api/v1/admin/challenges/:challenge_slug
func DeleteManagedChallengeHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement challenge deletion
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
