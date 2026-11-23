package me

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetMeHandler handles GET /api/v1/me
func GetMeHandler(c *gin.Context) {
	// TODO: implement current authenticated profile inspection
	var resp dto.MeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// UpdateMeHandler handles PATCH /api/v1/me
func UpdateMeHandler(c *gin.Context) {
	var req dto.UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement profile update (username, public_ssh_key)
	var resp dto.MeResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
