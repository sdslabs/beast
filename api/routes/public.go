package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/handlers/auth"
	"github.com/sdslabs/beastv4/api/handlers/competition"
	"github.com/sdslabs/beastv4/api/handlers/system"
	"github.com/sdslabs/beastv4/api/middleware"
)

// RegisterPublicRoutes registers root operational endpoints, public competition endpoints, and public auth routes.
func RegisterPublicRoutes(router *gin.Engine, v1 *gin.RouterGroup) {
	// Operational root endpoints
	router.GET("/healthz", system.HealthzHandler)
	router.GET("/readyz", middleware.OperatorAuthorize, system.ReadyzHandler)
	router.GET("/openapi.json", middleware.DocsAuthorize, system.OpenAPISpecHandler)
	router.GET("/docs", middleware.DocsAuthorize, system.DocsUIHandler)

	// Public competition metadata
	v1.GET("/competition", competition.GetCompetitionHandler)
	v1.GET("/competition/logo", competition.GetCompetitionLogoHandler)

	// Public auth routes
	authPublic := v1.Group("/auth")
	{
		authPublic.POST("/register", auth.RegisterHandler)
		authPublic.POST("/login", auth.LoginHandler)
		authPublic.POST("/refresh", auth.RefreshTokenHandler)
		authPublic.POST("/password-reset/request", auth.RequestPasswordResetHandler)
		authPublic.POST("/password-reset/confirm", auth.ConfirmPasswordResetHandler)
	}
}
