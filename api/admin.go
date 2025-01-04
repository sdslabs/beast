package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
)

// Ban/Unban a user based on his id and the action provided.
// @Summary Ban/Unban a user based on his id and the action provided.
// @Description Ban/unban a user based on his user id. This operation can only be done by admins
// @Tags admin
// @Accept  json
// @Produce json
// @Param action query string true "Action to perform ban/unban"
// @Param id query string true "Id of user"
// @Success 200 {object} api.ChallengeStatusResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/admin/users/:action/:id [post]
func banUserHandler(c *gin.Context) {
	action := c.Param("action")
	userId := c.Param("id")

	if (action != core.USER_STATUS["ban"]) && (action != core.USER_STATUS["unban"]) {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Action not provided or invalid action format",
		})
		return
	}

	var userState uint

	if action == core.USER_STATUS["ban"] {
		userState = 1
	} else if action == core.USER_STATUS["unban"] {
		userState = 0
	}

	parsedUserId, err := strconv.Atoi(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "User Id format invalid",
		})
		return
	}

	user, err := database.QueryUserById(uint(parsedUserId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	err = database.UpdateUser(&user, map[string]interface{}{"Status": userState})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Successfully %sned the user with id %s", action, userId),
	})
	return
}

func banTeamHandler(c *gin.Context) {
	action := c.Param("action")
	teamId := c.Param("id")

	// Validate action
	if (action != core.TEAM_STATUS["ban"]) && (action != core.TEAM_STATUS["unban"]) {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Action not provided or invalid action format",
		})
		return
	}

	var teamState uint
	if action == core.TEAM_STATUS["ban"] {
		teamState = 1
	} else if action == core.TEAM_STATUS["unban"] {
		teamState = 0
	}

	// Convert teamId to integer
	parsedTeamId, err := strconv.Atoi(teamId)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Team Id format invalid",
		})
		return
	}

	// Fetch the team from the database
	team, err := database.QueryTeamById(uint(parsedTeamId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	// Update the team's status
	err = database.UpdateTeam(&team, map[string]interface{}{"Status": teamState})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Successfully %sned the team with id %s", action, teamId),
	})

}
