package me

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ChangePasswordHandler handles POST /api/v1/me/password
func ChangePasswordHandler(c *gin.Context) {
	// TODO: implement authenticated password change
	httputil.RespondNotImplemented(c, "ChangePasswordHandler")
}
