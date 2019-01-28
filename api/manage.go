package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/manager"
	log "github.com/sirupsen/logrus"
)

// Handles route related to manage all the challenges for current beast remote.
// @Summary Handles challenge management actions for multiple(all) challenges.
// @Description Handles challenge management routes for all the challenges with actions which includes - DEPLOY, UNDEPLOY.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param action query string true "Action for the challenge"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/manage/all/:action [post]
func manageMultipleChallengeHandler(c *gin.Context) {
	action := c.Param("action")

	switch action {
	case MANAGE_ACTION_DEPLOY:
		log.Infof("Starting deploy for all challenges")
		msgs := manager.DeployAll(true)
		var msg string
		if len(msgs) != 0 {
			msg = strings.Join(msgs, " ::: ")
		} else {
			msg = "Deploy for all challeges started"
		}

		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": msg,
		})
		break

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Invalid Action : %s", action),
		})
	}
}

// Handles route related to managing a challenge
// @Summary Handles challenge management actions.
// @Description Handles challenge management routes with actions which includes - DEPLOY, UNDEPLOY, PURGE.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param name query string true "Name of the challenge to be managed, here name is the unique identifier for challenge"
// @Param action query string true "Action for the challenge"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/manage/challenge/ [post]
func manageChallengeHandler(c *gin.Context) {
	identifier := c.PostForm("name")
	action := c.PostForm("action")

	log.Infof("Trying %s for challenge with identifier : %s", action, identifier)

	switch action {
	case MANAGE_ACTION_UNDEPLOY:
		if err := manager.UndeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		respStr := fmt.Sprintf("Your action %s on challenge %s has started", action, identifier)
		c.JSON(http.StatusOK, gin.H{
			"message": respStr,
		})
		return

	case MANAGE_ACTION_PURGE:
		if err := manager.UndeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		respStr := fmt.Sprintf("Your action %s on challenge %s has started", action, identifier)
		c.JSON(http.StatusOK, gin.H{
			"message": respStr,
		})
		return

	case MANAGE_ACTION_REDEPLOY:
		// Redeploying a challenge means to first purge the challenge and then try to deploy it.
		if err := manager.UndeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := manager.DeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

	case MANAGE_ACTION_DEPLOY:
		// For deploy, identifier is name
		if err := manager.DeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, identifier)
	c.JSON(http.StatusOK, gin.H{
		"message": respStr,
	})
}

// Deploy local challenge
// @Summary Deploy a local challenge using the path provided in the post parameter
// @Description Handles deployment of a challenge using the absolute directory path
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param challenge_dir query string true "Challenge Directory"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/deploy/local [post]
func deployLocalChallengeHandler(c *gin.Context) {
	challDir := c.PostForm("challenge_dir")
	if challDir == "" {
		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": "No challenge directory specified",
		})
		return
	}

	log.Info("In local deploy challenge Handler")
	err := manager.DeployChallengePipeline(challDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	challengeName := filepath.Base(challDir)
	respStr := fmt.Sprintf("Deploy for challenge %s has been triggered.\n", challengeName)

	c.JSON(http.StatusOK, gin.H{
		"message": respStr,
	})
}

// Handles route related to beast static content serving container
// @Summary Handles route related to beast static content serving container, takes action as route parameter and perform that action
// @Description Handles beast static content serving container routes.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/static/:action [post]
func beastStaticContentHandler(c *gin.Context) {
	action := c.Param("action")

	switch action {
	case MANAGE_ACTION_DEPLOY:
		go manager.DeployStaticContentContainer()
		c.JSON(http.StatusOK, gin.H{
			"message": "Static container deploy started",
		})
		return

	case MANAGE_ACTION_UNDEPLOY:
		go manager.UndeployStaticContentContainer()
		c.JSON(http.StatusOK, gin.H{
			"message": "Static content container undeploy started",
		})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Invalid Action : %s", action),
		})
	}
}
