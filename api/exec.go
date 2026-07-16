package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
)

const (
	maxExecRequestBytes  = 32 << 10
	maxExecArguments     = 64
	maxExecArgumentBytes = 16 << 10
	defaultExecTimeout   = 30
	maxExecTimeout       = 300
)

type execChallengeRequest struct {
	Command        []string `json:"command"`
	InstanceID     string   `json:"instance_id,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

type execChallengeResponse struct {
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
	Truncated bool   `json:"truncated"`
}

func (request *execChallengeRequest) validate() error {
	if len(request.Command) == 0 || len(request.Command) > maxExecArguments {
		return errors.New("command must contain between 1 and 64 arguments")
	}
	totalBytes := 0
	for index, argument := range request.Command {
		if index == 0 && argument == "" {
			return errors.New("command executable cannot be empty")
		}
		if strings.IndexByte(argument, 0) >= 0 {
			return errors.New("command arguments cannot contain NUL bytes")
		}
		totalBytes += len(argument)
	}
	if totalBytes > maxExecArgumentBytes {
		return errors.New("command arguments exceed 16 KiB")
	}
	if request.TimeoutSeconds == 0 {
		request.TimeoutSeconds = defaultExecTimeout
	}
	if request.TimeoutSeconds < 1 || request.TimeoutSeconds > maxExecTimeout {
		return errors.New("timeout_seconds must be between 1 and 300")
	}
	return nil
}

func userCanExecChallenge(user database.User, challenge database.Challenge, maintainer bool) bool {
	if user.Status != 0 {
		return false
	}
	if user.Role == core.USER_ROLES["admin"] {
		return true
	}
	if user.Role != core.USER_ROLES["author"] && user.Role != core.USER_ROLES["maintainer"] {
		return false
	}
	return challenge.AuthorID == user.ID || maintainer
}

func execChallengeHandler(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxExecRequestBytes)
	var request execChallengeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: "invalid exec request"})
		return
	}
	if err := request.validate(); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: err.Error()})
		return
	}

	claimsValue, exists := c.Get("authClaims")
	claims, validClaims := claimsValue.(*auth.CustomClaims)
	if !exists || !validClaims {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{Error: "verified authentication claims are required"})
		return
	}
	user, err := database.QueryFirstUserEntry("username", claims.User)
	if err != nil || user.ID == 0 {
		c.JSON(http.StatusForbidden, HTTPErrorResp{Error: "user is not authorized for container execution"})
		return
	}
	challenge, err := database.QueryFirstChallengeEntry("name", c.Param("name"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "failed to query challenge"})
		return
	}
	if challenge.ID == 0 {
		c.JSON(http.StatusNotFound, HTTPErrorResp{Error: "challenge not found"})
		return
	}
	maintainer, err := database.IsChallengeMaintainer(user.ID, challenge.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "failed to authorize challenge access"})
		return
	}
	if !userCanExecChallenge(user, challenge, maintainer) {
		c.JSON(http.StatusForbidden, HTTPErrorResp{Error: "challenge execution access denied"})
		return
	}
	if challenge.Status != core.DEPLOY_STATUS["deployed"] || challenge.Type == core.STATIC_CHALLENGE_TYPE_NAME {
		c.JSON(http.StatusConflict, HTTPErrorResp{Error: "challenge does not have a running container"})
		return
	}

	containerID := challenge.ContainerId
	serverName := challenge.ServerDeployed
	if challenge.Instanced {
		if request.InstanceID == "" {
			c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: "instance_id is required for an instanced challenge"})
			return
		}
		instance, err := cache.GetInstance(request.InstanceID)
		if err != nil || instance == nil || instance.ChallengeName != challenge.Name {
			c.JSON(http.StatusNotFound, HTTPErrorResp{Error: "challenge instance not found"})
			return
		}
		containerID = instance.ContainerID
		serverName = instance.ServerDeployed
	}
	if containerID == "" {
		c.JSON(http.StatusConflict, HTTPErrorResp{Error: "challenge container is unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(request.TimeoutSeconds)*time.Second)
	defer cancel()
	var result cr.ExecResult
	server, found := config.Cfg.AvailableServers[serverName]
	if !found || !server.Active {
		c.JSON(http.StatusBadGateway, HTTPErrorResp{Error: "challenge worker is unavailable"})
		return
	}
	_ = manager.LogTransaction(challenge.Name, "EXEC", c.GetHeader("Authorization"))
	if config.Cfg.UseLocalDockerDaemon(serverName) {
		result, err = cr.ExecContainer(ctx, containerID, request.Command, cr.DefaultExecOutputLimit)
	} else {
		result, err = remoteManager.ExecContainerRemote(ctx, server, containerID, request.Command, cr.DefaultExecOutputLimit)
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, HTTPErrorResp{Error: "container command timed out"})
		} else {
			c.JSON(http.StatusBadGateway, HTTPErrorResp{Error: "container command failed to execute"})
		}
		return
	}
	c.JSON(http.StatusOK, execChallengeResponse{
		Stdout: result.Stdout, Stderr: result.Stderr, ExitCode: result.ExitCode, Truncated: result.Truncated,
	})
}
