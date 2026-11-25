package submission

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// SubmitFlagHandler handles POST /api/v1/challenges/:challenge_slug/submissions
func SubmitFlagHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	var req dto.SubmitFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement flag validation, solve recording, and dynamic score updating
	var resp dto.TeamSubmissionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
