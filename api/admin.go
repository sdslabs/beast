package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
)

// Ban/Unban/Hide/Unhide a user based on his id and the action provided.
// @Summary Ban/Unban/Hide/Unhide a user based on his id and the action provided.
// @Description Ban/Unban/Hide/Unhide a user based on his user id. This operation can only be done by admins
// @Tags admin
// @Accept  json
// @Produce json
// @Param action query string true "Action to perform Ban/Unban/Hide/Unhide"
// @Param id query string true "Id of user"
// @Success 200 {object} api.ChallengeStatusResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/admin/users/:action/:id [post]
func userActionHandler(c *gin.Context) {
	action := c.Param("action")
	userId := c.Param("id")

	if (action != core.USER_STATUS["ban"]) && (action != core.USER_STATUS["unban"]) && (action != core.USER_STATUS["hide"]) && (action != core.USER_STATUS["unhide"]) {
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
	} else if action == core.USER_STATUS["hide"] {
		userState = 2
	} else if action == core.USER_STATUS["unhide"] {
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

	err = database.UpdateUser(&user, map[string]interface{}{"status": userState})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	if val, _ := database.IsFrozenScoreSet(); !val {
		leaderboardStale = true
		graphCacheStale = true
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Successfully %sned the user with id %s", action, userId),
	})
}
