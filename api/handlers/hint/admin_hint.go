package hint

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListManagedHintsHandler handles GET /api/v1/admin/challenges/:challenge_slug/hints
func ListManagedHintsHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement administrative hints list
	var resp dto.ManageListHintsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// CreateChallengeHintHandler handles POST /api/v1/admin/challenges/:challenge_slug/hints
func CreateChallengeHintHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	var req dto.ManageCreateHintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement hint creation
	var resp dto.ManageHintResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}

// UpdateChallengeHintHandler handles PATCH /api/v1/admin/challenges/:challenge_slug/hints/:hint_id
func UpdateChallengeHintHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	hintID := c.Param("hint_id")
	_ = slug
	_ = hintID

	var req dto.ManageUpdateHintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement hint update
	var resp dto.ManageHintResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// DeleteChallengeHintHandler handles DELETE /api/v1/admin/challenges/:challenge_slug/hints/:hint_id
func DeleteChallengeHintHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	hintID := c.Param("hint_id")
	_ = slug
	_ = hintID

	// TODO: implement hint deletion
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
