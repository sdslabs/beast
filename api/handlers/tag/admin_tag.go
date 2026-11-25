package tag

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// CreateTagHandler handles POST /api/v1/admin/tags
func CreateTagHandler(c *gin.Context) {
	var req dto.ManageCreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement tag creation
	var resp dto.TagResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}

// UpdateTagHandler handles PATCH /api/v1/admin/tags/:tag_id
func UpdateTagHandler(c *gin.Context) {
	tagID := c.Param("tag_id")
	_ = tagID

	var req dto.ManageUpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement tag rename/update
	var resp dto.TagResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// DeleteTagHandler handles DELETE /api/v1/admin/tags/:tag_id
func DeleteTagHandler(c *gin.Context) {
	tagID := c.Param("tag_id")
	_ = tagID

	// TODO: implement tag global deletion
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
