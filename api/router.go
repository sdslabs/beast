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
	router.Use(gin.Logger())
	router.Use(middleware.RequestResponseLogger)
	router.Use(cors.New(middleware.NewCORSConfig()))

	v1 := router.Group("/api/v1")

	routes.RegisterPublicRoutes(router, v1)

	accountGroup := v1.Group("", middleware.AccountAuthorize)
	routes.RegisterAccountRoutes(accountGroup)

	userGroup := accountGroup.Group("", middleware.UserAuthorize)
	routes.RegisterUserRoutes(userGroup)
	routes.RegisterTeamRoutes(userGroup)

	adminGroup := accountGroup.Group("/admin", middleware.AdminAuthorize)
	routes.RegisterAdminRoutes(adminGroup)

	router.NoRoute(notFoundHandler)

	return router
}
