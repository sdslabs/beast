package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// RequestEmailVerificationHandler handles POST /api/v1/auth/email-verification/request
func RequestEmailVerificationHandler(c *gin.Context) {
	// TODO: implement email verification OTP request
	httputil.RespondNotImplemented(c, "RequestEmailVerificationHandler")
}

// ConfirmEmailVerificationHandler handles POST /api/v1/auth/email-verification/confirm
func ConfirmEmailVerificationHandler(c *gin.Context) {
	// TODO: implement email verification OTP confirmation
	httputil.RespondNotImplemented(c, "ConfirmEmailVerificationHandler")
}
