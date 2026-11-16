package middleware

import (
	"github.com/gin-gonic/gin"
)

// IdempotencyMiddleware handles deduplication of mutating requests via Idempotency-Key.
func IdempotencyMiddleware(c *gin.Context) {
	key := c.GetHeader("Idempotency-Key")
	if key != "" {
		// TODO: check Redis / cache for existing execution result
		c.Set("idempotency_key", key)
	}
	c.Next()
}
