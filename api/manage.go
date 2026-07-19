package api

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

const (
	maxChallengeUploadBytes        int64 = 256 << 20
	maxChallengeUploadRequestBytes       = maxChallengeUploadBytes + 1<<20
)

var challengeUploadMu sync.Mutex

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
func manageMultipleChallengeHandlerTagBased(c *gin.Context) {
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

	if action == core.MANAGE_ACTION_PURGE {
		markLeaderboardCachesStale()
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, identifier)
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: respStr,
	})
}

// Handles route related to managing multiple challenges.
// @Summary Handles multiple challenge management actions.
// @Description Handles challenge management routes with actions which includes - DEPLOY, UNDEPLOY, PURGE of multiple challenges.
// @NameBased manage
// @Accept  json
// @Produce json
// @Param name query string true "Name of the challenge to be managed, here name is the unique identifier for challenges seperated by a comma"
// @Param action query string true "Action for the challenge"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/challenge/multiple/ [post]
func manageMultipleChallengeHandlerNameBased(c *gin.Context) {
	identifier := c.PostForm("names")
	action := c.PostForm("action")
	authorization := c.GetHeader("Authorization")

	challAction, ok := manager.ChallengeActionHandlers[action]
	if !ok {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	names := strings.Split(identifier, ",")
	messages := make(map[string]string)
	var respStr string
	doesExist := make(map[string]bool)

	for _, name := range names {
		if !doesExist[name] {
			log.Infof("Trying %s for challenge with identifier : %s", action, name)
			if msgs := manager.LogTransaction(name, action, authorization); msgs != nil {
				log.Info("error while getting details")
			}

			log.Infof("Trying %s for challenge with identifier : %s", action, name)

			err := challAction(name)

			if err != nil {
				respStr = err.Error()
				messages[name] = respStr
			} else {
				respStr = fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, name)
				messages[name] = respStr
			}
			doesExist[name] = true
		}
	}

	if action == core.MANAGE_ACTION_PURGE {
		markLeaderboardCachesStale()
	}

	c.JSON(http.StatusOK, HTTPPlainMapResp{
		Messages: messages,
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
		if err := manager.DeployStaticContentContainer(); err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Static container deployed",
		})
		return

	case core.MANAGE_ACTION_UNDEPLOY:
		if err := manager.UndeployStaticContentContainer(); err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{Message: err.Error()})
			return
		}
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

// Prepare challenge info from .zip file.
// @Summary Unzip and fetch info from beast.toml file in challenge
// @Description Handles the challenge management from a challenge in zip file. Currently prepare the zip file
// by running `zip -r chall_dir.zip *` inside the chall_dir.
// @Tags manage
// @Accept  json
// @Produce json
// @Param file formData file true ".zip file to be uploaded to fetch challenge info"
// @Success 200 {object} api.ChallengePreviewResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/manage/challenge/upload [post]
func manageUploadHandler(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChallengeUploadRequestBytes)
	file, err := c.FormFile("file")

	if err != nil {
		status := http.StatusBadRequest
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			status = http.StatusRequestEntityTooLarge
		}
		c.AbortWithStatusJSON(status, HTTPPlainResp{
			Message: "a ZIP challenge archive is required",
		})
		return
	}
	archiveName, err := challengeArchiveFilename(file.Filename)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, HTTPErrorResp{Error: err.Error()})
		return
	}

	tempRoot, err := os.MkdirTemp("", "beast-upload-")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: fmt.Sprintf("Unable to create upload staging directory: %s", err),
		})
		return
	}
	defer os.RemoveAll(tempRoot)

	zipContextPath := filepath.Join(tempRoot, archiveName)
	if err := saveChallengeArchive(file, zipContextPath); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errChallengeUploadTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		c.AbortWithStatusJSON(status, HTTPErrorResp{Error: err.Error()})
		return
	}

	tempStageDir, err := manager.UnzipChallengeFolder(zipContextPath, tempRoot)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: fmt.Sprintf("The unzip process failed or the ZIP was unacceptable: %s", err),
		})
		return
	}

	if err := manager.ValidateChallengeConfig(tempStageDir); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	configFile := filepath.Join(tempStageDir, core.CHALLENGE_CONFIG_FILE_NAME)
	config, err := cfg.LoadChallengeConfig(configFile)
	if err != nil {
		log.Errorf("Error while loading uploaded challenge config: %s", err)
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: fmt.Sprintf("CONFIG ERROR: %s", err),
		})
		return
	}
	if err := persistUploadedChallenge(tempStageDir, config.Challenge.Metadata.Name); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, os.ErrExist) {
			status = http.StatusConflict
		}
		c.AbortWithStatusJSON(status, HTTPErrorResp{
			Error: fmt.Sprintf("Unable to store challenge: %s", err),
		})
		return
	}

	c.JSON(http.StatusOK, ChallengePreviewResp{
		Name:            config.Challenge.Metadata.Name,
		Category:        config.Challenge.Metadata.Type,
		Tags:            config.Challenge.Metadata.Tags,
		Assets:          config.Challenge.Metadata.Assets,
		AdditionalLinks: config.Challenge.Metadata.AdditionalLinks,
		Ports:           config.Challenge.Env.Ports,
		PreReqs:         config.Challenge.Metadata.PreReqs,
		Desc:            config.Challenge.Metadata.Description,
		Points:          config.Challenge.Metadata.Points,
	})
}

var errChallengeUploadTooLarge = errors.New("challenge archive exceeds the 256 MiB limit")

func challengeArchiveFilename(name string) (string, error) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, `\`) {
		return "", fmt.Errorf("invalid challenge archive filename")
	}
	if !strings.EqualFold(filepath.Ext(name), ".zip") || strings.TrimSuffix(name, filepath.Ext(name)) == "" {
		return "", fmt.Errorf("challenge archive must have a non-empty .zip filename")
	}
	return name, nil
}

func saveChallengeArchive(header *multipart.FileHeader, destination string) error {
	if header.Size < 0 || header.Size > maxChallengeUploadBytes {
		return errChallengeUploadTooLarge
	}
	source, err := header.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(target, io.LimitReader(source, maxChallengeUploadBytes+1))
	closeErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxChallengeUploadBytes {
		return errChallengeUploadTooLarge
	}
	return nil
}

func persistUploadedChallenge(source, challengeName string) error {
	uploadsRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR)
	if err := os.MkdirAll(uploadsRoot, 0700); err != nil {
		return err
	}
	rootInfo, err := os.Lstat(uploadsRoot)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("uploads root is not a regular directory")
	}
	if err := os.Chmod(uploadsRoot, 0700); err != nil {
		return err
	}

	stageRoot, err := os.MkdirTemp(uploadsRoot, ".upload-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageRoot)
	stagedChallenge := filepath.Join(stageRoot, challengeName)
	if err := manager.CopyDir(source, stagedChallenge); err != nil {
		return err
	}

	challengeUploadMu.Lock()
	defer challengeUploadMu.Unlock()
	destination := filepath.Join(uploadsRoot, challengeName)
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("challenge %q is already uploaded: %w", challengeName, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(stagedChallenge, destination)
}

func validateFlagHandler(c *gin.Context) {
	flag := c.PostForm("flag")
	challenge_name := c.PostForm("challenge_name")
	authorization := c.GetHeader("Authorization")

	challenges, err := database.QueryChallengeEntries("name", challenge_name)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	if len(challenges) == 0 {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Challenge not found",
		})
		return
	}

	if msgs := manager.LogTransaction(challenge_name, "VALIDATE_FLAG: "+flag, authorization); msgs != nil {
		log.Warn("Error while saving transaction")
	}

	err = manager.ValidateFlag(flag, challenge_name)

	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Flag validated!",
	})

}
