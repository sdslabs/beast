package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/middleware"
	"github.com/sdslabs/beastv4/api/routes"
)

func notFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"code":       "PAGE_NOT_FOUND",
		"message":    "Page not found",
		"request_id": c.GetString("request_id"),
	})
}

func InitGinRouter() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.RequestIDMiddleware)
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{Skip: middleware.LoggerSkipFunc}))
	router.Use(middleware.RequestResponseLogger)
	router.Use(cors.New(middleware.NewCORSConfig()))

	v1 := router.Group("/api/v1")

	// 1. Public operational root routes, competition routes, and auth
	routes.RegisterPublicRoutes(router, v1)

	// 2. Account-scoped routes (logged in; email verification may be pending)
	accountGroup := v1.Group("", middleware.AccountAuthorize)
	routes.RegisterAccountRoutes(accountGroup)

	// 3. User-scoped routes (logged in, email-verified, permitted to participate)
	userGroup := accountGroup.Group("", middleware.UserAuthorize)
	routes.RegisterUserRoutes(userGroup)
	routes.RegisterTeamRoutes(userGroup)

	// 4. Admin-scoped routes
	adminGroup := accountGroup.Group("/admin", middleware.AdminAuthorize)
	routes.RegisterAdminRoutes(adminGroup)

	router.NoRoute(notFoundHandler)

	return router
}

func initGinRouter() *gin.Engine {
	return InitGinRouter()
}
