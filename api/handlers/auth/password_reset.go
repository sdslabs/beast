package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// RequestPasswordResetHandler handles POST /api/v1/auth/password-reset/request
func RequestPasswordResetHandler(c *gin.Context) {
	// TODO: implement password reset OTP dispatch
	httputil.RespondNotImplemented(c, "RequestPasswordResetHandler")
}

// ConfirmPasswordResetHandler handles POST /api/v1/auth/password-reset/confirm
func ConfirmPasswordResetHandler(c *gin.Context) {
	// TODO: implement password reset OTP verification & password change
	httputil.RespondNotImplemented(c, "ConfirmPasswordResetHandler")
}
