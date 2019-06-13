package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/manager"
	"net/http"
)

// Returns port in use by beast.
// @Summary Returns ports in use by beast by looking in the hack git repository, also returns min and max value of port allowed while specifying in beast challenge config.
// @Description Returns the ports in use by beast, which cannot be used in creating a new challenge..
// @Tags info
// @Accept  json
// @Produce json
// @Success 200 {object} api.PortsInUseResp
// @Router /api/info/ports/used [get]
func usedPortsInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, PortsInUseResp{
		MinPortValue: core.ALLOWED_MIN_PORT_VALUE,
		MaxPortValue: core.ALLOWED_MAX_PORT_VALUE,
		PortsInUse:   cfg.USED_PORTS_LIST,
	})
}

func challengeInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: WIP_TEXT,
	})
}

func availableChallengeInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: WIP_TEXT,
	})
}

// Returns available base images.
// @Summary Gives all the base images that can be used while creating a beast challenge, this is a constant specified in beast global config
// @Description Returns all the available base images  which can be used for challenge creation as the base OS for challenge.
// @Tags info
// @Accept  json
// @Produce json
// @Success 200 {object} api.AvailableImagesResp
// @Router /api/info/images/available [get]
func availableImagesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, AvailableImagesResp{
		Message: "Available base images are",
		Images:  cfg.Cfg.AllowedBaseImages,
	})
}

// Returns available challenges.
// @Summary Gives all challenges available in the in the database
// @Description Returns all challenges available in the in the database
// @Tags info
// @Accept json
// @Produce json
// @Success 200 {object} api.ChallengesResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/info/challenges/available [get]
func challengesHandler(c *gin.Context) {
	challenges, err := manager.GetAvailableChallenges()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	} else if challenges == nil {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "No challenges currently in the database",
		})
		return
	} else {
		c.JSON(http.StatusOK, ChallengesResp{
			Message:    "All Challenges",
			Challenges: challenges,
		})
	}
}
