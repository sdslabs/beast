package submission

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListSolvesHandler handles GET /api/v1/solves
func ListSolvesHandler(c *gin.Context) {
	var query dto.ListSolvesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement verified solves feed listing
	var resp dto.ListSolvesResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
