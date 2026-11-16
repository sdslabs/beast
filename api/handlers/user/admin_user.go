package user

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminListUsersHandler handles GET /api/v1/admin/users
func AdminListUsersHandler(c *gin.Context) {
	// TODO: implement administrative users listing
	httputil.RespondNotImplemented(c, "AdminListUsersHandler")
}

// AdminCreateUserHandler handles POST /api/v1/admin/users
func AdminCreateUserHandler(c *gin.Context) {
	// TODO: implement administrative user account creation
	httputil.RespondNotImplemented(c, "AdminCreateUserHandler")
}

// AdminGetUserHandler handles GET /api/v1/admin/users/:user_id
func AdminGetUserHandler(c *gin.Context) {
	// TODO: implement administrative user details retrieval
	httputil.RespondNotImplemented(c, "AdminGetUserHandler")
}

// AdminUpdateUserHandler handles PATCH /api/v1/admin/users/:user_id
func AdminUpdateUserHandler(c *gin.Context) {
	// TODO: implement administrative user status, role, ban modification
	httputil.RespondNotImplemented(c, "AdminUpdateUserHandler")
}
