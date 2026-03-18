package api

import (
	"fmt"
	"github.com/sdslabs/beastv4/core"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
)

type InstanceResponse struct {
	InstanceID    string    `json:"instance_id"`
	ChallengeName string    `json:"challenge_name"`
	HostedAddress string    `json:"hosted_address"`
	Port          uint32    `json:"port"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	TTLSeconds    int64     `json:"ttl_seconds"`
}

type AdminInstanceResponse struct {
	InstanceResponse
	UserID         string `json:"user_id"`
	Username       string `json:"username"`
	ContainerID    string `json:"container_id"`
	DeploymentType string `json:"deployment_type"`
}

func instanceToResponse(instance *cache.Instance) InstanceResponse {
	ttl := time.Until(instance.ExpiresAt).Seconds()
	if ttl < 0 {
		ttl = 0
	}

	return InstanceResponse{
		InstanceID:    instance.InstanceID,
		ChallengeName: instance.ChallengeName,
		HostedAddress: instance.ServerDeployed,
		Port:          instance.Port,
		CreatedAt:     instance.CreatedAt,
		ExpiresAt:     instance.ExpiresAt,
		TTLSeconds:    int64(ttl),
	}
}

func instanceToAdminResponse(instance *cache.Instance) AdminInstanceResponse {
	return AdminInstanceResponse{
		InstanceResponse: instanceToResponse(instance),
		UserID:           instance.UserID,
		Username:         instance.Username,
		ContainerID:      instance.ContainerID,
		DeploymentType:   instance.DeploymentType,
	}
}

func spawnInstanceHandler(ctx *gin.Context) {
	challengeName := ctx.Param("challenge_name")
	if challengeName == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "challenge_name is required",
		})
		return
	}

	username, err := coreUtils.GetUser(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized",
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "User not found",
		})
		return
	}

	userID := fmt.Sprintf("%d", user.ID)

	instance, err := manager.SpawnInstance(challengeName, userID, username, user.SshKey)
	if err != nil {
		if instance != nil {
			ctx.JSON(http.StatusConflict, HTTPErrorResp{
				Error: err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, instanceToResponse(instance))
}

func getUserInstanceHandler(ctx *gin.Context) {
	challengeName := ctx.Param("challenge_name")
	if challengeName == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "challenge_name is required",
		})
		return
	}

	username, err := coreUtils.GetUser(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized",
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "User not found",
		})
		return
	}

	userID := fmt.Sprintf("%d", user.ID)

	instance, err := manager.GetUserInstance(userID, challengeName)
	if err != nil {
		ctx.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No active instance found for this challenge",
		})
		return
	}

	ctx.JSON(http.StatusOK, instanceToResponse(instance))
}

func getUserInstancesHandler(ctx *gin.Context) {
	username, err := coreUtils.GetUser(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized",
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "User not found",
		})
		return
	}

	userID := fmt.Sprintf("%d", user.ID)

	instances, err := manager.GetUserInstances(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	var response []InstanceResponse
	for _, instance := range instances {
		response = append(response, instanceToResponse(instance))
	}

	if response == nil {
		response = []InstanceResponse{}
	}

	ctx.JSON(http.StatusOK, response)
}

func extendInstanceHandler(ctx *gin.Context) {
	challengeName := ctx.Param("challenge_name")
	if challengeName == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "challenge_name is required",
		})
		return
	}

	username, err := coreUtils.GetUser(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized",
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "User not found",
		})
		return
	}

	userID := fmt.Sprintf("%d", user.ID)

	instance, err := manager.GetUserInstance(userID, challengeName)
	if err != nil {
		ctx.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No active instance found for this challenge",
		})
		return
	}

	additionalSeconds := core.DEFAULT_MINIMUM_EXTEND_TIME
	if seconds := ctx.PostForm("seconds"); seconds != "" {
		var parsedSeconds int64
		_, err := fmt.Sscanf(seconds, "%d", &parsedSeconds)
		if err == nil && parsedSeconds > 0 {
			additionalSeconds = parsedSeconds
		}
	}

	maxExtension := config.Cfg.InstanceConfig.MaxExtension
	if additionalSeconds > maxExtension {
		additionalSeconds = maxExtension
	}

	err = manager.ExtendInstance(instance.InstanceID, additionalSeconds)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	instance, err = manager.GetUserInstance(userID, challengeName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to get updated instance",
		})
		return
	}

	ctx.JSON(http.StatusOK, instanceToResponse(instance))
}

func killUserInstanceHandler(ctx *gin.Context) {
	challengeName := ctx.Param("challenge_name")
	if challengeName == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "challenge_name is required",
		})
		return
	}

	username, err := coreUtils.GetUser(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized",
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "User not found",
		})
		return
	}

	userID := fmt.Sprintf("%d", user.ID)

	err = manager.KillUserInstance(userID, challengeName)
	if err != nil {
		ctx.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Instance killed successfully",
	})
}

func adminGetAllInstancesHandler(ctx *gin.Context) {
	instances, err := manager.GetAllInstances()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	var response []AdminInstanceResponse
	for _, instance := range instances {
		response = append(response, instanceToAdminResponse(instance))
	}

	if response == nil {
		response = []AdminInstanceResponse{}
	}

	ctx.JSON(http.StatusOK, response)
}

func adminGetInstanceHandler(ctx *gin.Context) {
	instanceID := ctx.Param("instance_id")
	if instanceID == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "instance_id is required",
		})
		return
	}

	instance, err := manager.GetInstance(instanceID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "Instance not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, instanceToAdminResponse(instance))
}

func adminKillInstanceHandler(ctx *gin.Context) {
	instanceID := ctx.Param("instance_id")
	if instanceID == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "instance_id is required",
		})
		return
	}

	err := manager.KillInstance(instanceID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Instance killed successfully",
	})
}

func adminKillUserInstancesHandler(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "user_id is required",
		})
		return
	}

	instances, err := manager.GetUserInstances(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	killedCount := 0
	for _, instance := range instances {
		err := manager.KillInstance(instance.InstanceID)
		if err == nil {
			killedCount++
		}
	}

	ctx.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("%d instances killed", killedCount),
	})
}

func adminKillChallengeInstancesHandler(ctx *gin.Context) {
	challengeName := ctx.Param("challenge_name")
	if challengeName == "" {
		ctx.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "challenge_name is required",
		})
		return
	}

	instances, err := manager.GetChallengeInstances(challengeName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	// can be delegated to a coroutine if bottlenecks performance
	killedCount := 0
	for _, instance := range instances {
		err = manager.KillInstance(instance.InstanceID)
		if err == nil {
			killedCount++
		}
	}

	ctx.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("%d instances killed", killedCount),
	})
}
