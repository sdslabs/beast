package api

import (
	"net/http"

	"github.com/fristonio/beast/core/deploy"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func deployAllHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func deployChallengeHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func deployLocalChallengeHandler(c *gin.Context) {
	challDir := c.PostForm("challenge_dir")
	if challDir == "" {
		c.String(http.StatusNotAcceptable, "No challenge directory specified")
		return
	}

	log.Info("In local deploy challenge Handler")
	err := deploy.DeployChallenge(challDir)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.String(http.StatusOK, WIP_TEXT)
}
