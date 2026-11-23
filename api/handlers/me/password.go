package me

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ChangePasswordHandler handles POST /api/v1/me/password
func ChangePasswordHandler(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement authenticated password change
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
