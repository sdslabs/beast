package me

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetMeHandler handles GET /api/v1/me
func GetMeHandler(c *gin.Context) {
	// TODO: implement current authenticated profile inspection
	httputil.RespondNotImplemented(c, "GetMeHandler")
}

// UpdateMeHandler handles PATCH /api/v1/me
func UpdateMeHandler(c *gin.Context) {
	// TODO: implement profile update (username, public_ssh_key)
	httputil.RespondNotImplemented(c, "UpdateMeHandler")
}
