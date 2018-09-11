package api

import "github.com/gin-gonic/gin"

func initGinRouter() *gin.Engine {
	router := gin.New()

	return router
}
