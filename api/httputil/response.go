package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func RespondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

func RespondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: c.GetString("request_id"),
	})
}

func RespondNotImplemented(c *gin.Context, handlerName string) {
	// TODO: implement handler logic
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Code:      "NOT_IMPLEMENTED",
		Message:   handlerName + " is not yet implemented",
		RequestID: c.GetString("request_id"),
	})
}
