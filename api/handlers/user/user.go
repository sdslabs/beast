package user

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListUsersHandler handles GET /api/v1/users
func ListUsersHandler(c *gin.Context) {
	// TODO: implement public users list
	httputil.RespondNotImplemented(c, "ListUsersHandler")
}

// GetUserHandler handles GET /api/v1/users/:user_id
func GetUserHandler(c *gin.Context) {
	// TODO: implement user public profile inspection
	httputil.RespondNotImplemented(c, "GetUserHandler")
}
