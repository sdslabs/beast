package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

// Handles route related to manage all the challenges or the challenges related to a particular tag for current beast remote.
// @Summary Handles challenge management actions for multiple challenges.
// @Description Handles challenge management routes for multiple the challenges with actions which includes - DEPLOY, UNDEPLOY.
// @Tags manage
// @Accept  json
// @Produce json
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
	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err == nil {
		log.Warnf("Error while getting user from authorization header, using default user(since already authorized)")
		username = core.DEFAULT_USER_NAME
	}

	_, ok := manager.ChallengeActionHandlers[action]
	if !ok {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	var msg string

	if tag != "" {
		log.Infof("Starting %s for all challenges related to tags", action)
		msgs := manager.HandleTagRelatedChallenges(action, tag, username)

		if len(msgs) != 0 {
			msg = fmt.Sprintf("Error while performing %s : %s", action, strings.Join(msgs, " || "))
		} else {
			msg = fmt.Sprintf("%s for all challeges started", action)
		}
	} else {
		log.Infof("Starting %s for all challenges", action)
		msgs := manager.HandleAll(action, username)

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
// @Produce json
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

	challAction, ok := manager.ChallengeActionHandlers[action]
	if !ok {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	log.Infof("Trying %s for challenge with identifier : %s", action, identifier)

	err := challAction(identifier)

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
// @Produce json
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
		log.Warn("Error while saving transaction")
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
// @Produce json
// @Param action query string true "Action to apply on the beast static content provider"
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

	// Static content provider for beast only supports two actions
	// Deploy and Undeploy
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

// Commit a challenge container
// @Summary Commits the challenge container so that later the challenge image can be used deployment
// @Description Commits the challenge container for later use
// @Tags manage
// @Accept  json
// @Produce json
// @Param challenge query string true "Name of the challenge to commit"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/manage/commit/ [post]
func commitChallenge(c *gin.Context) {
	challenge := c.PostForm("challenge")

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

// Verify a challenge for configuration.
// @Summary Validates the configuration of the challenge and tells if challenge can be deployed or not.
// @Description Validates challenge configuration for deployment.
// @Tags manage
// @Accept  json
// @Produce json
// @Param challenge query string true "Name of the challenge to verify the deployment configuration for."
// @Success 200 {object} api.HTTPPlainResp
// @Success 200 {object} api.HTTPErrorResp
// @Router /api/manage/commit/ [post]
func verifyHandler(c *gin.Context) {
	challengeName := c.PostForm("challenge")
	challengeRemoteDir := coreUtils.GetChallengeDir(challengeName)
	if challengeRemoteDir == "" {
		log.Errorf("Challenge does not exist")
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Challenge does not exist",
		})
		return
	}
	err := manager.ValidateChallengeConfig(challengeRemoteDir)
	if err != nil {
		c.JSON(http.StatusOK, HTTPErrorResp{
			Error: err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Challenge verified",
		})
	}
}

// Execute a scheduled action on a challenge.
// @Summary Schedule an action(deploy, undeploy, purge etc.) on a particular challenge
// @Description Handles scheduleing of challenge action to executed at some later point of time
// @Tags manage
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param action query string true "Action for the underlying challenge in context"
// @Param challenge query string false "The name of the challenge to schedule the action for."
// @Param tags query string false "Tag corresponding to challenges in context, optional if challenge name is provided"
// @Param at query string false "Timestamp at which the challenge should be scheduled should be a unix timestamp string."
// @Param after query string false "Time after which the action on the selector should be executed should be of duration format as in '1m20s' etc."
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/schedule/:action [post]
func manageScheduledAction(c *gin.Context) {
	action := c.Param("action")
	challenge := c.PostForm("challenge")
	tag := c.PostForm("tag")

	authorization := c.GetHeader("Authorization")
	username, err := coreUtils.GetUser(authorization)
	if err == nil {
		log.Warn("Error while getting user from authorization header, using default user(since already authorized)")
		username = core.DEFAULT_USER_NAME
	}

	if action == "" || (challenge == "" && tag == "") {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Action and a challenge selector like name or tag is required but not provided",
		})
		return
	}

	at := c.PostForm("at")
	after := c.PostForm("after")
	if at == "" && after == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "A time correspondence should be associated with a schedule, no parameter at or after",
		})
		return
	}

	var duration time.Duration
	if at != "" {
		duration, err = utils.GetDurationFromTimestamp(at)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Invalid timestamp provided: %s: %s", at, err),
			})
			return
		}
	} else {
		duration, err = time.ParseDuration(after)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Invalid after duration provided: %s: %s", after, err),
			})
			return
		}
	}

	actionHandler, ok := manager.ChallengeActionHandlers[action]
	if !ok {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	// If a tag is provided we deploy using the tag, else we deploy the challenge
	// name we are provided
	if tag != "" {
		manager.LogTransaction(fmt.Sprintf("TAG:%s", tag), "SCHEDULE::"+action, authorization)

		BeastScheduler.ScheduleAfter(duration, manager.HandleTagRelatedChallenges, action, tag, username)
		log.Infof("Scheduled %s for challenges with tag %s", action, tag)
	} else {
		manager.LogTransaction(challenge, "SCHEDULE::"+action, authorization)

		BeastScheduler.ScheduleAfter(duration, actionHandler, challenge)
		log.Infof("Scheduled %s for challenge %s", action, challenge)
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Challenge with the provided selector has been scheduled for %s", action),
	})
}

// Prepare challenge info from .tar file.
// @Summary Untar and fetch info from beast.toml file in challenge
// @Description Handles the challenge management from a challenge in tar file. Currently prepare the tar file
// by running `tar cvf chall_dir.tar .` inside the chall_dir.
// @Tags manage
// @Accept  json
// @Produce json
// @Param file query file true ".tar file to be uploaded to fetch challenge info"
// @Success 200 {object} api.ChallengePreviewResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/manage/challenge/upload [post]
func manageUploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")

	// The file cannot be received.
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("No file received from user"),
		})
		return
	}

	if err = utils.CreateIfNotExistDir(core.BEAST_TEMP_DIR); err != nil {
		if err := os.MkdirAll(core.BEAST_TEMP_DIR, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: fmt.Sprintf("Could not create dir %s: %s", core.BEAST_TEMP_DIR, err),
			})
		}
	}

	tarContextPath := filepath.Join(core.BEAST_TEMP_DIR, file.Filename)

	// The file is received, save it
	if err := c.SaveUploadedFile(file, tarContextPath); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: fmt.Sprintf("Unable to save file: %s", err),
		})
		return
	}

	// Extract and show from tar and return response
	tempStageDir, err := manager.UnTarChallengeFolder(tarContextPath, core.BEAST_TEMP_DIR)

	// The file cannot be successfully un-tar-ed or the resultant was a malformed directory
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: fmt.Sprintf("The un-TAR process failed or the TAR was unacceptable: %s", err),
		})
		return
	}

	err = manager.ValidateChallengeConfig(tempStageDir)
	if err != nil {
		c.JSON(http.StatusOK, HTTPErrorResp{
			Error: err.Error(),
		})
	}

	challengeUploadDirectory := filepath.Join(
		core.BEAST_GLOBAL_DIR,
		core.BEAST_UPLOADS_DIR,
		strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
	)

	if err = manager.CopyDir(tempStageDir, challengeUploadDirectory); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: fmt.Sprintf("Unable to move challenge directory: %s", err),
		})
		return
	}

	challengeName := filepath.Base(challengeUploadDirectory)
	configFile := filepath.Join(challengeUploadDirectory, core.CHALLENGE_CONFIG_FILE_NAME)

	var config cfg.BeastChallengeConfig
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		log.Errorf("Error while loading beast config for challenge %s : %s", challengeName, err)
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: fmt.Sprintf("CONFIG ERROR: %s : %s", challengeName, err),
		})
		return
	}

	c.JSON(http.StatusOK, ChallengePreviewResp{
		Name:     config.Challenge.Metadata.Name,
		Category: config.Challenge.Metadata.Type,
		Ports:    config.Challenge.Env.Ports,
		Hints:    config.Challenge.Metadata.Hints,
		Desc:     config.Challenge.Metadata.Description,
		Points:   config.Challenge.Metadata.Points,
	})
}
