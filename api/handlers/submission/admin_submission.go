package submission

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminListSubmissionsHandler handles GET /api/v1/admin/submissions
func AdminListSubmissionsHandler(c *gin.Context) {
	// TODO: implement administrative submissions listing
	httputil.RespondNotImplemented(c, "AdminListSubmissionsHandler")
}

// AdminGetSubmissionHandler handles GET /api/v1/admin/submissions/:submission_id
func AdminGetSubmissionHandler(c *gin.Context) {
	// TODO: implement administrative submission details inspection
	httputil.RespondNotImplemented(c, "AdminGetSubmissionHandler")
}
