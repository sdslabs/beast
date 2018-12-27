package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/gin-swagger/swaggerFiles"

	_ "github.com/sdslabs/beastv4/api/docs"
	"github.com/sdslabs/beastv4/git"
)

const (
	DEFAULT_BEAST_PORT = ":5005"
)

func runBeastApiBootsteps() error {
	git.RunBeastBootsetps()

	return nil
}

type HTTPPlainResp struct {
	Message string `json:"message" example:"Messsage in response to your request"`
}

type HTTPAuthorizeResp struct {
	Token   string `json:"token" example:"YOUR_AUTHENTICATION_TOKEN"`
	Message string `json:"message" example:"Response message"`
}

type AuthorizationChallengeResp struct {
	Challenge string `json:"challenge" example:"Challenge String"`
	Message   string `json:"message" example:"Response message"`
}

// @title Beast API
// @version 1.0
// @description Beast the automatic deployment tool for backdoor

// @contact.name SDSLabs
// @contact.url https://chat.sdslabs.co
// @contact.email contact.sdslabs.co.in

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host beast.sdslabs.co
// @BasePath /
func RunBeastApiServer(port string) {
	log.Info("Bootstrapping Beast API server")
	if port != "" {
		port = ":" + port
	} else {
		port = DEFAULT_BEAST_PORT
	}

	runBeastApiBootsteps()

	// Initialize Gin router.
	router := initGinRouter()

	// Setup gin middlewares
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": WELCOME_TEXT,
		})
	})

	router.GET("/help", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": HELP_TEXT,
		})
	})

	router.Run(port)
}
