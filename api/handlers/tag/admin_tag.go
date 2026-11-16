package tag

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// CreateTagHandler handles POST /api/v1/admin/tags
func CreateTagHandler(c *gin.Context) {
	// TODO: implement tag creation
	httputil.RespondNotImplemented(c, "CreateTagHandler")
}

// UpdateTagHandler handles PATCH /api/v1/admin/tags/:tag_id
func UpdateTagHandler(c *gin.Context) {
	// TODO: implement tag rename/update
	httputil.RespondNotImplemented(c, "UpdateTagHandler")
}

// DeleteTagHandler handles DELETE /api/v1/admin/tags/:tag_id
func DeleteTagHandler(c *gin.Context) {
	// TODO: implement tag global deletion
	httputil.RespondNotImplemented(c, "DeleteTagHandler")
}
