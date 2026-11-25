package competition

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetAdminCompetitionHandler handles GET /api/v1/admin/competition
func GetAdminCompetitionHandler(c *gin.Context) {
	// TODO: implement administrative competition settings retrieval
	var resp dto.AdminCompetitionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// UpdateAdminCompetitionHandler handles PATCH /api/v1/admin/competition
func UpdateAdminCompetitionHandler(c *gin.Context) {
	var req dto.AdminUpdateCompetitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative competition settings update
	var resp dto.AdminCompetitionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// UploadCompetitionLogoHandler handles PUT /api/v1/admin/competition/logo
func UploadCompetitionLogoHandler(c *gin.Context) {
	// Multipart file upload
	file, err := c.FormFile("logo")
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_FILE", err.Error())
		return
	}
	_ = file

	// TODO: implement branding logo upload & validation
	var resp dto.AdminCompetitionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// DeleteCompetitionLogoHandler handles DELETE /api/v1/admin/competition/logo
func DeleteCompetitionLogoHandler(c *gin.Context) {
	// TODO: implement branding logo deletion
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
