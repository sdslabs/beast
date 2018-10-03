package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/deploy"
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
	err := deploy.DeployChallengePipeline(challDir)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.String(http.StatusOK, WIP_TEXT)
}
