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
// @Tags status
// @Accept  json
// @Produce json
// @Param action param "Action to perform (ban/unban)"
// @Param id param "Id of user"
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
