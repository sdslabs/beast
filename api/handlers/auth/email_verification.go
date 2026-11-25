package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// RequestEmailVerificationHandler handles POST /api/v1/auth/email-verification/request
func RequestEmailVerificationHandler(c *gin.Context) {
	// TODO: implement email verification OTP request
	var resp dto.OTPDispatchResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ConfirmEmailVerificationHandler handles POST /api/v1/auth/email-verification/confirm
func ConfirmEmailVerificationHandler(c *gin.Context) {
	var req dto.ConfirmOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement email verification OTP confirmation
	var resp dto.AuthSessionResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
