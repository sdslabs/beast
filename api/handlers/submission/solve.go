package submission

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListSolvesHandler handles GET /api/v1/solves
func ListSolvesHandler(c *gin.Context) {
	// TODO: implement verified solves feed listing
	httputil.RespondNotImplemented(c, "ListSolvesHandler")
}
