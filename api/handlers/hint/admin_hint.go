package hint

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListManagedHintsHandler handles GET /api/v1/admin/challenges/:challenge_slug/hints
func ListManagedHintsHandler(c *gin.Context) {
	// TODO: implement administrative hints list
	httputil.RespondNotImplemented(c, "ListManagedHintsHandler")
}

// CreateChallengeHintHandler handles POST /api/v1/admin/challenges/:challenge_slug/hints
func CreateChallengeHintHandler(c *gin.Context) {
	// TODO: implement hint creation
	httputil.RespondNotImplemented(c, "CreateChallengeHintHandler")
}

// UpdateChallengeHintHandler handles PATCH /api/v1/admin/challenges/:challenge_slug/hints/:hint_id
func UpdateChallengeHintHandler(c *gin.Context) {
	// TODO: implement hint update
	httputil.RespondNotImplemented(c, "UpdateChallengeHintHandler")
}

// DeleteChallengeHintHandler handles DELETE /api/v1/admin/challenges/:challenge_slug/hints/:hint_id
func DeleteChallengeHintHandler(c *gin.Context) {
	// TODO: implement hint deletion
	httputil.RespondNotImplemented(c, "DeleteChallengeHintHandler")
}
