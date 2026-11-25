package challenge

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetChallengeInstancePolicyHandler handles GET /api/v1/admin/challenges/:challenge_slug/instance-policy
func GetChallengeInstancePolicyHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement instance lifetime policy inspection
	var resp dto.ManageInstancePolicyResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ReplaceChallengeInstancePolicyHandler handles PUT /api/v1/admin/challenges/:challenge_slug/instance-policy
func ReplaceChallengeInstancePolicyHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	var req dto.ManageReplaceInstancePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement instance lifetime policy replacement
	var resp dto.ManageInstancePolicyResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
