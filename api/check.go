package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

// Verifies and creates an entry in the database for successful submission for a challenge.
// @Summary Verifies and creates an entry in the database for successful submission of a challenge.
// @Description Returns success or error response based on the status of completion.
// @Tags Submit
// @Accept  json
// @Produce json
// @Param chall_id formData string true "Name of challenge"
// @Param instance_id formData string true "Instance ID for challenge"
// @Success 200 {object} api.ChallengeStatusResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 401 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/check/challenge [post]
func checkFlagHandler(c *gin.Context) {
	challId := c.PostForm("chall_id")
	instanceId := c.PostForm("instance_id")

	err, state := coreUtils.CheckTime()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}
	if state == 0 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Competition is yet to start",
		})
		return
	}
	if state == 2 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Competition has ended",
		})
		return
	}
	if state == 1 {
		username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Unauthorized user",
			})
			return
		}

		if challId == "" {
			c.JSON(http.StatusBadRequest, HTTPErrorResp{
				Error: "Id of the challenge is a required parameter to process request.",
			})
			return
		}

		if instanceId == "" {
			c.JSON(http.StatusBadRequest, HTTPErrorResp{
				Error: "Instance of id is required",
			})
			return
		}

		user, err := database.QueryFirstUserEntry("username", username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Unauthorized user",
			})
			return
		}

		if user.Status == 1 {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Banned user",
			})
			return
		}

		chall, err := database.QueryChallengeEntries("id", challId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		if len(chall) == 0 {
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: "Challenge is unavailable",
				Success: false,
			})
			return
		}

		lockManager := GetUserLockManager()
		lockId := fmt.Sprintf("%d_%s", user.ID, challId)
		if !lockManager.TryLock(lockId) {
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: "Request already in process",
				Success: false,
			})
			return
		}
		defer lockManager.Unlock(lockId)

		challenge := chall[0]
		if challenge.Status != core.DEPLOY_STATUS["deployed"] {
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: "Challenge is unavailable",
				Success: false,
			})
			return
		}

		if challenge.PreReqs != "" {
			preReqsStatus, err := database.CheckPreReqsStatus(challenge, user.ID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}

			if !preReqsStatus {
				c.JSON(http.StatusOK, ChallengeSubmitResponse{
					Message: "You have not solved the prerequisites of this challenge.",
					Success: false,
				})
				return
			}
		}

		instance, err := cache.GetInstance(instanceId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "CACHE ERROR while processing the request.",
			})
			return
		}

		if instance.Username != username {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Unauthorized user",
			})
			return
		}
		if instance.ChallengeName != challenge.Name {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Unauthorized challenge",
			})
			return
		}
		if err := manager.ValidateSadServersInstanceChecker(instance, challenge); err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: fmt.Sprintf("VALIDATION ERROR: %s", err.Error()),
			})
			return
		}

		attempt, err := database.ReserveSubmissionAttempt(user.ID, challenge.ID, challenge.MaxAttemptLimit, instanceId, time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		switch attempt.Status {
		case database.SubmissionAttemptAlreadySolved:
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: "Challenge has already been solved.",
				Success: false,
			})
			return
		case database.SubmissionAttemptMaxAttempts:
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: "You have reached the maximum number of tries for this challenge.",
				Success: false,
			})
			return
		}

		localDeploy := config.Cfg.UseLocalDockerDaemon(instance.ServerDeployed)

		var result cr.ExecResult
		if challenge.SadServers {
			result, err = manager.ExecuteCheckerScript(instance)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("CONTAINER RUNTIME ERROR while executing sadservers checker: %s", err.Error()),
				})
				return
			}
		} else if instance.DeploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
			verified, err := manager.ValidateCheckerManifest(instance)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("CONTAINER RUNTIME ERROR while validating checker manifest: %s", err.Error()),
				})
				return
			}
			if !verified {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "VALIDATION ERROR: challenge files do not match the recorded checker manifest.",
				})
				return
			}

			result, err = manager.ExecuteCheckerScript(instance)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("CONTAINER RUNTIME ERROR while executing checker script: %s", err.Error()),
				})
				return
			}
		} else {
			exists, err := checkScriptExistence(localDeploy, instance)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("CONTAINER RUNTIME ERROR while verifying the existence of check script: %s", err.Error()),
				})
				return
			}

			if !exists {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("VALIDATION ERROR: check script not found at %s.", core.SAD_CHECK_SCRIPT_LOCATION),
				})
				return
			}

			verified, err := validateCheckScriptHash(localDeploy, instance)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("CONTAINER RUNTIME ERROR while processing the request: %s", err.Error()),
				})
				return
			}
			if !verified {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "VALIDATION ERROR: hash of check.sh does not match, file tampered with.",
				})
				return
			}

			result, err = executeCheckScript(localDeploy, instance)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("CONTAINER RUNTIME ERROR while executing check script: %s", err.Error()),
				})
				return
			}
		}

		if result.ExitCode != 0 {
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: fmt.Sprintf("Challenge check failed with EXIT CODE: %v", result.ExitCode),
				Success: false,
			})
			return
		}

		log.Debugf("Dynamic scoring is set to %t", config.Cfg.CompetitionInfo.DynamicScore)
		finalized, err := database.FinalizeSubmissionSolve(user.ID, challenge.ID, instanceId, false, time.Now(), config.Cfg.CompetitionInfo.DynamicScore)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		if !finalized.Awarded {
			c.JSON(http.StatusOK, ChallengeSubmitResponse{
				Message: "Challenge has already been solved.",
				Success: false,
			})
			return
		}
		if config.Cfg.CompetitionInfo.DynamicScore && finalized.Points != challenge.Points {
			log.Debugf("By dynamic scoring the points of challenge %s are changed to %d from %d", challenge.Name, finalized.Points, challenge.Points)
		}

		leaderboardStale = true
		graphCacheStale = true
		adminLeaderboardStale = true

		c.JSON(http.StatusOK, ChallengeSubmitResponse{
			Message: "Your challenge submission has been verified",
			Success: true,
		})

		return
	}
}

func checkScriptExistence(localDeploy bool, instance *cache.Instance) (bool, error) {
	var err error
	var result cr.ExecResult

	containerId := instance.ContainerID
	fileCommand := fmt.Sprintf("[ -f '%s' ]", core.SAD_CHECK_SCRIPT_LOCATION)
	if localDeploy {
		result, err = cr.RunCommandInContainer(containerId, []string{
			"sh", "-c", fileCommand,
		})
	} else {
		server := config.Cfg.AvailableServers[instance.ServerDeployed]
		result, err = remoteManager.RunCommandInContainerOnServer(server, containerId, fileCommand)
	}

	if err != nil {
		return false, err
	} else if result.ExitCode != 0 {
		return false, fmt.Errorf("failed to verify location of check.sh")
	}

	return true, nil
}

func validateCheckScriptHash(localDeploy bool, instance *cache.Instance) (bool, error) {
	var err error
	var result cr.ExecResult

	containerId := instance.ContainerID
	hashCommand := fmt.Sprintf("command cat %s | sha256sum", core.SAD_CHECK_SCRIPT_LOCATION)
	if localDeploy {
		result, err = cr.RunCommandInContainer(containerId, []string{
			"sh", "-c", hashCommand,
		})
	} else {
		server := config.Cfg.AvailableServers[instance.ServerDeployed]
		result, err = remoteManager.RunCommandInContainerOnServer(server, containerId, hashCommand)
	}

	if err != nil {
		return false, err
	}
	if result.ExitCode != 0 {
		return false, fmt.Errorf("check script hash failed")
	}

	return strings.TrimSpace(result.Output) == instance.CheckHash, nil
}

func executeCheckScript(localDeploy bool, instance *cache.Instance) (cr.ExecResult, error) {
	if localDeploy {
		return cr.RunCommandInContainer(instance.ContainerID, []string{
			"sh", "-c", core.SAD_CHECK_SCRIPT_LOCATION,
		})
	} else {
		server := config.Cfg.AvailableServers[instance.ServerDeployed]
		return remoteManager.RunCommandInContainerOnServer(server, instance.ContainerID, core.SAD_CHECK_SCRIPT_LOCATION)
	}
}
