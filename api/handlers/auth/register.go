package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// RegisterHandler handles POST /api/v1/auth/register
func RegisterHandler(c *gin.Context) {
	// TODO: implement user registration
	httputil.RespondNotImplemented(c, "RegisterHandler")
}
