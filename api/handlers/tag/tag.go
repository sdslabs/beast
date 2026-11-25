package tag

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListTagsHandler handles GET /api/v1/tags
func ListTagsHandler(c *gin.Context) {
	// TODO: implement public tags catalog listing
	var resp dto.ListTagsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
