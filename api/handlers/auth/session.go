package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// LoginHandler handles POST /api/v1/auth/login
func LoginHandler(c *gin.Context) {
	// TODO: implement user login
	httputil.RespondNotImplemented(c, "LoginHandler")
}

// RefreshTokenHandler handles POST /api/v1/auth/refresh
func RefreshTokenHandler(c *gin.Context) {
	// TODO: implement token refresh
	httputil.RespondNotImplemented(c, "RefreshTokenHandler")
}

// LogoutHandler handles POST /api/v1/auth/logout
func LogoutHandler(c *gin.Context) {
	// TODO: implement user logout / session revocation
	httputil.RespondNotImplemented(c, "LogoutHandler")
}
