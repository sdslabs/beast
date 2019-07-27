package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
)

// Returns port in use by beast.
// @Summary Returns ports in use by beast by looking in the hack git repository, also returns min and max value of port allowed while specifying in beast challenge config.
// @Description Returns the ports in use by beast, which cannot be used in creating a new challenge..
// @Tags info
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
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
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Challenge name cannot be empty"),
		})
		return
	}

	challenge, err := database.QueryChallengeEntries("name", name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	var challDescription string
	var challAuthorID uint
	if len(challenge) > 0 {
		challDescription = challenge[0].Description
		challAuthorID = challenge[0].AuthorID
	} else {
		challDescription = "Not Available"
		challAuthorID = 0
	}
	challAuthor, err := database.QueryUserById(challAuthorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while fetching author info.",
		})
		return
	}
	c.JSON(http.StatusOK, ChallengeDescriptionResp{
		Name:   name,
		Author: challAuthor.Name,
		Desc:   challDescription,
	})
	return
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
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.AvailableImagesResp
// @Router /api/info/images/available [get]
func availableImagesHandler(c *gin.Context) {
	c.JSON(http.StatusOK, AvailableImagesResp{
		Message: "Available base images are",
		Images:  cfg.Cfg.AllowedBaseImages,
	})
}

// Handles route related to logs handling
// @Summary Handles route related to logs handling of container
// @Description Gives container logs for a particular challenge, useful for debugging purposes.
// @Tags info
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param challenge query string false "The name of the challenge to get the logs for."
// @Success 200 {object} api.LogsInfoResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/info/logs [get]
func challengeLogsHandler(c *gin.Context) {
	chall := c.Query("challenge")
	if chall == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Challenge name cannot be empty"),
		})
		return
	}

	logs, err := utils.GetLogs(chall, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, LogsInfoResp{
			Stdout: logs.Stdout,
			Stderr: logs.Stderr,
		})
	}
}

// Returns available challenges by filter
// @Summary Gives all challenges available in the in the database that has a particular parameter same
// @Description Returns all challenges available in the in the database that has a particular parameter same
// @Tags info
// @Accept json
// @Produce json
// @Success 200 {object} api.ChallengesResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/info/challenges [get]
func challengeInfoByFilterHandler(c *gin.Context) {
	filter, ok := c.GetQuery("filter")
	value, ok := c.GetQuery("value")

	if !ok {
		fmt.Println("Url Param 'key' is missing")
		return
	}

	if value == "" || filter == "" {
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

	var challenges []database.Challenge
	var err error
	challenges, err = database.QueryChallengeEntries(filter, value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
	} else {
		var resp []ChallengesByFilterResp
		for _, challenge := range challenges {
			r := ChallengesByFilterResp{
				Message:    "Challenges with " + filter + " = " + value,
				Challenges: challenge.Name,
			}
			resp = append(resp, r)
		}

		c.JSON(http.StatusOK, resp)
	}
}
