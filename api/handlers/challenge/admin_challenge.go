package challenge

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetManagedChallengeHandler handles GET /api/v1/admin/challenges/:challenge_slug
func GetManagedChallengeHandler(c *gin.Context) {
	// TODO: implement administrative challenge metadata inspection
	httputil.RespondNotImplemented(c, "GetManagedChallengeHandler")
}

// UpdateManagedChallengeHandler handles PATCH /api/v1/admin/challenges/:challenge_slug
func UpdateManagedChallengeHandler(c *gin.Context) {
	// TODO: implement administrative challenge update
	httputil.RespondNotImplemented(c, "UpdateManagedChallengeHandler")
}

// DeleteManagedChallengeHandler handles DELETE /api/v1/admin/challenges/:challenge_slug
func DeleteManagedChallengeHandler(c *gin.Context) {
	// TODO: implement challenge deletion
	httputil.RespondNotImplemented(c, "DeleteManagedChallengeHandler")
}
