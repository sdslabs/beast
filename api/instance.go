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

// @Summary Spawn a per-user challenge instance
// @Tags instances
// @Produce json
// @Param challenge_name path string true "Challenge name"
// @Security ApiKeyAuth
// @Success 200 {object} api.InstanceResponse
// @Failure 400 {object} api.HTTPErrorResp
// @Router /api/instances/{challenge_name}/spawn [post]
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

	instance, err := manager.SpawnInstance(challengeName, userID, username)
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

// @Summary Get the current user's challenge instance
// @Tags instances
// @Produce json
// @Param challenge_name path string true "Challenge name"
// @Security ApiKeyAuth
// @Success 200 {object} api.InstanceResponse
// @Failure 404 {object} api.HTTPErrorResp
// @Router /api/instances/{challenge_name} [get]
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

// @Summary List the current user's instances
// @Tags instances
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} api.InstanceResponse
// @Router /api/instances [get]
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

// @Summary Extend the current user's challenge instance
// @Tags instances
// @Produce json
// @Param challenge_name path string true "Challenge name"
// @Param seconds formData int false "Requested extension in seconds"
// @Security ApiKeyAuth
// @Success 200 {object} api.InstanceResponse
// @Failure 400 {object} api.HTTPErrorResp
// @Router /api/instances/{challenge_name}/extend [post]
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

// @Summary Delete the current user's challenge instance
// @Tags instances
// @Produce json
// @Param challenge_name path string true "Challenge name"
// @Security ApiKeyAuth
// @Success 200 {object} api.HTTPPlainResp
// @Router /api/instances/{challenge_name} [delete]
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

// @Summary List all active instances
// @Tags admin
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} api.AdminInstanceResponse
// @Router /api/admin/instances [get]
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

// @Summary Get an instance by ID
// @Tags admin
// @Produce json
// @Param instance_id path string true "Instance ID"
// @Security ApiKeyAuth
// @Success 200 {object} api.AdminInstanceResponse
// @Router /api/admin/instances/{instance_id} [get]
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

// @Summary Delete an instance by ID
// @Tags admin
// @Produce json
// @Param instance_id path string true "Instance ID"
// @Security ApiKeyAuth
// @Success 200 {object} api.HTTPPlainResp
// @Router /api/admin/instances/{instance_id} [delete]
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

// @Summary Delete all instances owned by a user
// @Tags admin
// @Produce json
// @Param user_id path string true "User ID"
// @Security ApiKeyAuth
// @Success 200 {object} api.HTTPPlainResp
// @Router /api/admin/instances/user/{user_id} [delete]
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

// @Summary Delete all instances for a challenge
// @Tags admin
// @Produce json
// @Param challenge_name path string true "Challenge name"
// @Security ApiKeyAuth
// @Success 200 {object} api.HTTPPlainResp
// @Router /api/admin/instances/challenge/{challenge_name} [delete]
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
