package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/auth"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
)

// Handles route related to manage all the challenges or the challenges related to a particular tag for current beast remote.
// @Summary Handles challenge management actions for multiple challenges.
// @Description Handles challenge management routes for multiple the challenges with actions which includes - DEPLOY, UNDEPLOY.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param action query string true "Action for the challenge"
// @Param tag query string false "Tag for a group of challenges"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/multiple/:action [post]

func manageMultipleChallengeHandler(c *gin.Context) {
	// If no tags are provided we by default we apply the action to all
	// the challenges.
	action := c.Param("action")
	tag := c.PostForm("tag")

	// We are trying to get the username for the request from JWT claims here
	// Since upto this point the request is already authorized, we use a default
	// username if any error occurs while getting the username.
	authorName, err := auth.GetUser(c.GetHeader("Authorization"))
	if err == nil {
		log.Warnf("Error while getting user from authorization header, using default user(since already authorized)")
		authorName = core.DEFAULT_USER_NAME
	}

	_, ok := manager.MapOfFunctions[action]
	if !ok {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	var msg string

	if tag != "" {
		log.Infof("Starting %s for all challenges related to tags", action)
		msgs := manager.HandleTagRelatedChallenges(action, tag, authorName)
		if len(msgs) != 0 {
			msg = fmt.Sprintf("Error while performing %s : %s", action, strings.Join(msgs, " || "))
		} else {
			msg = fmt.Sprintf("%s for all challeges started", action)
		}
	} else {
		log.Infof("Starting %s for all challenges", action)
		msgs := manager.HandleAll(action, authorName)
		if len(msgs) != 0 {
			msg = fmt.Sprintf("Error while performing %s : %s", action, strings.Join(msgs, " || "))
		} else {
			msg = "Deploy for all challeges started"
		}
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: msg,
	})
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
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/challenge/ [post]
func manageChallengeHandler(c *gin.Context) {
	identifier := c.PostForm("name")
	action := c.PostForm("action")
	authorization := c.GetHeader("Authorization")

	log.Infof("Trying %s for challenge with identifier : %s", action, identifier)
	if msgs := manager.LogTransaction(identifier, action, authorization); msgs != nil {
		log.Info("error while getting details")
	}

	f, ok := manager.MapOfFunctions[action]
	if !ok {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	log.Infof("Trying %s for challenge with identifier : %s", action, identifier)

	err := f(identifier)

	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, identifier)
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: respStr,
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
// @Failure 406 {object} api.HTTPPlainResp
// @Router /api/manage/deploy/local [post]
func deployLocalChallengeHandler(c *gin.Context) {
	action := core.MANAGE_ACTION_DEPLOY
	challDir := c.PostForm("challenge_dir")
	authorization := c.GetHeader("Authorization")

	if challDir == "" {
		c.JSON(http.StatusNotAcceptable, HTTPPlainResp{
			Message: "No challenge directory specified",
		})
		return
	}

	log.Info("In local deploy challenge Handler")
	err := manager.DeployChallengePipeline(challDir)
	if msgs := manager.LogTransaction(strings.Split(challDir, "/")[len(strings.Split(challDir, "/"))-1], action, authorization); msgs != nil {
		log.Info("error while saving transaction")
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	challengeName := filepath.Base(challDir)
	respStr := fmt.Sprintf("Deploy for challenge %s has been triggered.\n", challengeName)

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: respStr,
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
	identifier := core.BEAST_STATIC_CONTAINER_NAME
	authorization := c.GetHeader("Authorization")

	if msgs := manager.LogTransaction(identifier, action, authorization); msgs != nil {
		log.Error("error while getting details")
	}

	switch action {
	case core.MANAGE_ACTION_DEPLOY:
		go manager.DeployStaticContentContainer()
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Static container deploy started",
		})
		return

	case core.MANAGE_ACTION_UNDEPLOY:
		go manager.UndeployStaticContentContainer()
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Static content container undeploy started",
		})
		return

	default:
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
	}
}

// Handles route related to logs handling
// @Summary Handles route related to logs handling of container
// @Description Container logs
// @Tags manage
// @Accept  json
// @Produce application/json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/logs/ [get]
func challengeLogsHandler(c *gin.Context) {
	chall := c.Param("name")
	if chall == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Name cannot be empty"),
		})
	}
	logs, err := utils.GetLogs(chall, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"stdout": logs.Stdout,
			"stderr": logs.Stderr,
		})
	}
}

// Commit a challenge container
// @Summary Commits the challenge container so that later the challenge image can be used deployment
// @Description Commits the challenge container for later use
// @Tags manage
// @Accept  json
// @Produce application/json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/manage/commit/ [post]
func commitChallenge(c *gin.Context) {
	challenge := c.PostForm("chall")

	err := manager.CommitChallengeContainer(challenge)

	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: fmt.Sprintf("Error : %s", err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Commit Done",
	})
}

func verifyHandler(c *gin.Context) {
	challengeName := c.PostForm("challenge")
	challengeStagingDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, config.Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR, challengeName)
	err := manager.ValidateChallengeConfig(challengeStagingDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"Error": err.Error(),
		})
		return
	} else {
		c.JSON(http.StatusOK, gin.H{
			"Message": "This challenge can be deployed",
		})
		return
	}
}
