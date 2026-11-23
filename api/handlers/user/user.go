package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListUsersHandler handles GET /api/v1/users
func ListUsersHandler(c *gin.Context) {
	var query dto.ListUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement public users list
	var resp dto.ListUsersResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetUserHandler handles GET /api/v1/users/:user_id
func GetUserHandler(c *gin.Context) {
	// Path param :user_id
	userID := c.Param("user_id")
	_ = userID

	// TODO: implement user public profile inspection
	var resp dto.UserProfileResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
