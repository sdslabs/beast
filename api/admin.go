package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
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

	err = database.UpdateUser(&user, map[string]interface{}{"status": userState})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Successfully %sned the user with id %s", action, userId),
	})
}

var (
	leaderboardFreeze = false
)

func freezeLeaderboardHandler(c *gin.Context) {
	leaderboardFreeze = true
	// get all users data
	us, err := database.QueryUserEntries("role", core.USER_ROLES["contestant"])
	log.Print("Freezing leaderboard")
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	length := len(us)
	users, err := database.QueryTopUsersByScore(length)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	// update the rank of all users
	var frozenUsers []UserResp
	for i, user := range users {
		frozenUsers = append(frozenUsers, UserResp{
			Username: user.Username,
			Id:       user.ID,
			Role:     user.Role,
			Status:   user.Status,
			Score:    user.Score,
			Email:    user.Email,
			Rank:     int64(i + 1),
		})
	}
	chunkSize := core.LEADERBOARD_SIZE
	for i := 0; i < length; i += chunkSize {
		end := i + chunkSize
		if end > length {
			end = length
		}
		chunk := frozenUsers[i:end]
		filePath := filepath.Join(core.BEAST_GLOBAL_DIR, fmt.Sprintf("leadboard-%d.json", i/chunkSize))
		file, err := os.Create(filePath)
		if err != nil {
			log.Errorf("Error while creating file: %s", err)
			continue
		}
		if err := json.NewEncoder(file).Encode(chunk); err != nil {
			log.Errorf("Error while writing to file: %s", err)
			continue
		}
		file.Close()
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Leaderboard frozen and stored",
	})

}

func unfreezeLeaderboardHandler(c *gin.Context) {
	leaderboardFreeze = false
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Leaderboard unfrozen",
	})
}
