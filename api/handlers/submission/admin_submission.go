package submission

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminListSubmissionsHandler handles GET /api/v1/admin/submissions
func AdminListSubmissionsHandler(c *gin.Context) {
	var query dto.AdminListSubmissionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative submissions listing
	var resp dto.AdminListSubmissionsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminGetSubmissionHandler handles GET /api/v1/admin/submissions/:submission_id
func AdminGetSubmissionHandler(c *gin.Context) {
	submissionID := c.Param("submission_id")
	_ = submissionID

	// TODO: implement administrative submission details inspection
	var resp dto.AdminSubmissionDetailResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
