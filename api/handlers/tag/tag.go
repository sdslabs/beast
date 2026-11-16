package tag

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListTagsHandler handles GET /api/v1/tags
func ListTagsHandler(c *gin.Context) {
	// TODO: implement public tags catalog listing
	httputil.RespondNotImplemented(c, "ListTagsHandler")
}
