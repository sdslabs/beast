package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
)

// Verifies and creates an entry in the database for successful submission of flag for a challenge.
// @Summary Verifies and creates an entry in the database for successful submission of flag for a challenge.
// @Description Returns success or error response based on the flag submitted. Also, the flag will not be submitted if it was previously submitted
// @Tags Submit
// @Accept  json
// @Produce json
// @Param chall_id formData string true "Name of challenge"
// @Param flag formData string true "Flag for the challenge"
// @Success 200 {object} api.ChallengeStatusResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 401 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/submit/challenge [post]
func submitFlagHandler(c *gin.Context) {
	challId := c.PostForm("chall_id")
	flag := c.PostForm("flag")

	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
		})
		return
	}

	if challId == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Id of the challenge is a required parameter to process request.",
		})
		return
	}

	if flag == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Flag for the challenge is a required parameter to process request.",
		})
		return
	}

	parsedChallId, err := strconv.Atoi(challId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	
	chall, err := database.QueryChallengeEntries("id", strconv.Itoa(int(parsedChallId)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	challenge := chall[0]

	if challenge.Flag != flag {
		c.JSON(http.StatusOK, FlagSubmitResp{
			Message: "Your flag is incorrect",
			Success: false,
		})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
		})
	}

	solved, err := database.CheckPreviousSubmissions(user.ID, challenge.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	if solved {
		c.JSON(http.StatusOK, FlagSubmitResp{
			Message: "Question has already been answered.",
			Success: false,
		})
		return
	}

	err = database.UpdateUser(&user, map[string]interface{}{"Score": user.Score + challenge.Points})
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	UserChallengesEntry := database.UserChallenges{
		CreatedAt:   time.Time{},
		UserID:      user.ID,
		ChallengeID: challenge.ID,
	}

	err = database.SaveFlagSubmission(&UserChallengesEntry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	c.JSON(http.StatusOK, FlagSubmitResp{
		Message: "Your flag is correct",
		Success: true,
	})

	return
}
