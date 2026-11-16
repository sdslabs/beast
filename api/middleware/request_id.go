package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDMiddleware assigns a unique X-Request-ID header and context value.
func RequestIDMiddleware(c *gin.Context) {
	reqID := c.GetHeader("X-Request-ID")
	if reqID == "" {
		reqID = uuid.New().String()
	}
	c.Set("request_id", reqID)
	c.Writer.Header().Set("X-Request-ID", reqID)
	c.Next()
}
