package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/utils"
)

// Gets a challenge deployment status on the basis of name.
// @Summary Returns challenge deployment status from the beast database.
// @Description Returns challenge deployment status from the beast database, for those challenges which are not present a status value NA is returned.
// @Tags status
// @Accept  json
// @Produce application/json
// @Param name query string true "Name of the challenge"
// @Success 200 {object} api.ChallengeStatusResp
// @Failure 500 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/status/challenge/:name [get]
func challengeStatusHandler(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Name of the challenge is a required parameter to process request.",
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

	var status string
	var updatedAt time.Time
	if len(challenge) > 0 {
		status = challenge[0].Status
		updatedAt = challenge[0].UpdatedAt
	} else {
		status = "Not Available"
	}

	c.JSON(http.StatusOK, ChallengeStatusResp{
		Name:      name,
		Status:    status,
		UpdatedAt: updatedAt,
	})
}

// Gets the list of the challenges with status, according to the filter provided.
// @Summary Returns challenge deployment status from the beast database for the challenges which matches the stauts according to filter.
// @Description This returns the challenges in the status provided, along with their name and last updated time.
// @Tags status
// @Accept  json
// @Produce application/json
// @Param filter query string true "Status type to filter with, if none specified then all"
// @Failure 500 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Success 200 {array} api.ChallengeStatusResp
// @Router /api/status/all/:filter [get]
func statusHandler(c *gin.Context) {
	filter := c.Param("filter")

	var availableStatus []string
	for key := range core.DEPLOY_STATUS {
		availableStatus = append(availableStatus, key)
	}

	if !utils.StringInSlice(filter, availableStatus) && filter != "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("%s is not a valid status to get the challenges by.", filter),
		})
		return
	}

	var challenges []database.Challenge
	var err error
	if filter == "" {
		challenges, err = database.QueryAllChallenges()
	} else {
		challenges, err = database.QueryChallengeEntries("status", core.DEPLOY_STATUS[filter])
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
	} else {
		var resp []ChallengeStatusResp
		for _, challenge := range challenges {
			r := ChallengeStatusResp{
				Name:      challenge.Name,
				Status:    challenge.Status,
				UpdatedAt: challenge.UpdatedAt,
			}
			resp = append(resp, r)
		}

		c.JSON(http.StatusOK, resp)
	}
}

func challengeDescriptionHandler(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Name of the challenge is a required parameter to process request.",
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
	if len(challenge) > 0 {
		challDescription = challenge[0].Description
	} else {
		challDescription = "Not Available"
	}
	c.JSON(http.StatusOK, challengeDescriptionResp{
		Name:        name,
		Description: challDescription,
	})
}
