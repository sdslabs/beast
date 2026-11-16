package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
)

// NewCORSConfig returns standard CORS configuration for the Beast API router.
func NewCORSConfig() cors.Config {
	return cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Cookie",
			"Idempotency-Key",
			"Last-Event-ID",
			"X-Request-ID",
		},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
		AllowAllOrigins:  true,
		MaxAge:           12 * time.Hour,
	}
}
