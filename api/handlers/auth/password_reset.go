package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// RequestPasswordResetHandler handles POST /api/v1/auth/password-reset/request
func RequestPasswordResetHandler(c *gin.Context) {
	var req dto.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement password reset OTP dispatch
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// ConfirmPasswordResetHandler handles POST /api/v1/auth/password-reset/confirm
func ConfirmPasswordResetHandler(c *gin.Context) {
	var req dto.PasswordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement password reset OTP verification & password change
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
