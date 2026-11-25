package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// LoginHandler handles POST /api/v1/auth/login
func LoginHandler(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement user login
	var resp dto.AuthSessionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// RefreshTokenHandler handles POST /api/v1/auth/refresh
func RefreshTokenHandler(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement token refresh
	var resp dto.AuthSessionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// LogoutHandler handles POST /api/v1/auth/logout
func LogoutHandler(c *gin.Context) {
	// TODO: implement user logout / session revocation
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
