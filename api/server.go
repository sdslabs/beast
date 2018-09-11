package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	DEFAULT_BEAST_PORT = ":5005"
)

func RunBeastApiServer(port string) {
	log.Info("Bootstrapping Beast API server")
	if port != "" {
		port = ":" + port
	} else {
		port = DEFAULT_BEAST_PORT
	}

	// Initialize Gin router.
	router := initGinRouter()

	// Setup gin middlewares
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, WELCOME_TEXT)
	})

	router.Run(port)
}
