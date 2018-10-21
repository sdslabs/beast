package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
)

func manageAllHandler(c *gin.Context) {
	c.String(http.StatusOK, WIP_TEXT)
}

func manageChallengeHandler(c *gin.Context) {
	id := c.Param("id")
	action := c.PostForm("action")

	switch action {
	case MANAGE_ACTION_UNDEPLOY:
		log.Infof("Trying %s for challenge with ID : %s", action, id)
		if err := manager.UndeployChallenge(id); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		respStr := fmt.Sprintf("Your action %s on challenge %s was successful", action, id)
		c.String(http.StatusOK, respStr)
		return

	case MANAGE_ACTION_DEPLOY:
		log.Infof("Trying to %s challenge with ID %s", action, id)

		if err := manager.DeployChallenge(id); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		c.String(http.StatusOK, "Deploy for challenge %s has been triggered, check stats", id)
		return

	default:
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid Action : %s", action))
		return
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, id)
	c.String(http.StatusOK, respStr)
}

// Deploy local challenge
// @Summary Deploy a local challenge using the path provided in the post parameter
// @Description Handles deployment of a challenge using the absolute directory path
// @Accept  json
// @Produce text/plain
// @Param challenge_dir query string true "Challenge Directory"
// @Success 200 {string} Success
// @Failure 400 {string} Error
// @Router /api/manage/deploy/local [post]
func deployLocalChallengeHandler(c *gin.Context) {
	challDir := c.PostForm("challenge_dir")
	if challDir == "" {
		c.String(http.StatusNotAcceptable, "No challenge directory specified")
		return
	}

	log.Info("In local deploy challenge Handler")
	err := manager.DeployChallengePipeline(challDir)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	challengeName := filepath.Base(challDir)
	respStr := fmt.Sprintf("Deploy for challenge %s has been triggered.\n", challengeName)

	c.String(http.StatusOK, respStr)
}
