package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/deploy"
	log "github.com/sirupsen/logrus"
)

func deployAllHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func deployChallengeHandler(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")

	switch action {
	case DEPLOY_ACTION_UNDEPLOY:
		log.Infof("Trying %s for challenge with ID : %s", action, id)
		if err := deploy.UndeployChallenge(id); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		respStr := fmt.Sprintf("Your action %s on challenge %s was successful", action, id)
		c.String(http.StatusOK, respStr)
		return

	default:
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid Action : %s", action))
		return
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, id)
	c.String(http.StatusOK, respStr)
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

	challengeName := filepath.Base(challDir)
	respStr := fmt.Sprintf("Deploy for challenge %s has been triggered.\n", challengeName)

	c.String(http.StatusOK, respStr)
}
