package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminListUsersHandler handles GET /api/v1/admin/users
func AdminListUsersHandler(c *gin.Context) {
	var query dto.AdminListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative users listing
	var resp dto.AdminListUsersResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminCreateUserHandler handles POST /api/v1/admin/users
func AdminCreateUserHandler(c *gin.Context) {
	var req dto.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative user account creation
	var resp dto.AdminUserResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}

// AdminGetUserHandler handles GET /api/v1/admin/users/:user_id
func AdminGetUserHandler(c *gin.Context) {
	userID := c.Param("user_id")
	_ = userID

	// TODO: implement administrative user details retrieval
	var resp dto.AdminUserResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminUpdateUserHandler handles PATCH /api/v1/admin/users/:user_id
func AdminUpdateUserHandler(c *gin.Context) {
	userID := c.Param("user_id")
	_ = userID

	var req dto.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative user status, role, ban modification
	var resp dto.AdminUserResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
