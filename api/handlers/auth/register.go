package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// RegisterHandler handles POST /api/v1/auth/register
func RegisterHandler(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement user registration
	var resp dto.AuthSessionResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}
