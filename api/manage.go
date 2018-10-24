package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
)

func manageChallengeHandler(c *gin.Context) {
	identifier := c.PostForm("name")
	action := c.PostForm("action")

	switch action {
	case MANAGE_ACTION_UNDEPLOY:
		log.Infof("Trying %s for challenge with identifier : %s", action, identifier)
		if err := manager.UndeployChallenge(identifier); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		respStr := fmt.Sprintf("Your action %s on challenge %s was successful", action, identifier)
		c.String(http.StatusOK, respStr)
		return

	case MANAGE_ACTION_DEPLOY:
		// For deploy, identifier is name
		log.Infof("Trying to %s challenge with name %s", action, identifier)

		if err := manager.DeployChallenge(identifier); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		c.String(http.StatusOK, "Deploy for challenge %s has been triggered, check stats", identifier)
		return

	default:
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid Action : %s", action))
		return
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, identifier)
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

func beastStaticContentHandler(c *gin.Context) {
	action := c.Param("action")

	switch action {
	case MANAGE_ACTION_DEPLOY:
		go manager.DeployStaticContentContainer()
		c.String(http.StatusOK, "Static container deploy started")
		return

	default:
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid Action : %s", action))
	}
}
