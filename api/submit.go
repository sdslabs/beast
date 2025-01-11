package api

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	log "github.com/sirupsen/logrus"
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

	err, state := coreUtils.CheckTime()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}
	if state == 0 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Competition is yet to start",
		})
		return
	}
	if state == 2 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Competition has ended",
		})
		return
	}
	if state == 1 {
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

		user, err := database.QueryFirstUserEntry("username", username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Unauthorized user",
			})
			return
		}

		if user.Status == 1 {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Banned user",
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
		if challenge.Status != core.DEPLOY_STATUS["deployed"] {
			c.JSON(http.StatusOK, FlagSubmitResp{
				Message: "Challenge is unavailable",
				Success: false,
			})
			return
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
				Message: "Challenge has already been solved.",
				Success: false,
			})
			return
		}

		tryStatus, err := database.GetUserPreviousTriesStatus(user.ID, challenge.ID, challenge.FailSolveLimit)

		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request."})
			return
		}

		if !tryStatus {
			c.JSON(http.StatusOK, FlagSubmitResp{
				Message: "You have reached the maximum number of tries for this challenge.",
				Success: false,
			})
			return
		}

		// Increase user tries by 1
		err = database.UpdateUserChallengeTries(user.ID, challenge.ID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}

		if challenge.Flag != flag {
			c.JSON(http.StatusOK, FlagSubmitResp{
				Message: "Your flag is incorrect",
				Success: false,
			})
			return
		}

		challengePoints := challenge.Points
		log.Debugf("Dynamic scoring is set to %t", config.Cfg.CompetitionInfo.DynamicScore)
		if config.Cfg.CompetitionInfo.DynamicScore {
			submissions, err := database.QuerySubmissions(map[string]interface{}{
				"challenge_id": parsedChallId,
			})
			if err != nil {
				log.Error(err)
			}
			solvers := len(submissions)
			newPoints := dynamicScore(challenge.MaxPoints, challenge.MinPoints, uint(solvers))
			if newPoints != challengePoints {
				database.UpdateChallenge(&challenge, map[string]interface{}{
					"Points": newPoints,
				})
				log.Debugf("By dynamic scoring the points of challenge %s are changed to %d from %d", challenge.Name, newPoints, challengePoints)
				err = updatePointsOfSolvers(submissions, newPoints, challengePoints)
				if err != nil {
					log.Error(err)
				}
				challengePoints = newPoints
			}
		}

		err = database.UpdateUser(&user, map[string]interface{}{"Score": user.Score + challengePoints})
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
			Solved:      true,
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
}

// dynamicScore returns dynamic score of the challenge based on number of solves
func dynamicScore(maxPoints, minPoints, solvers uint) uint {
	if solvers == 0 || solvers == 1 {
		return maxPoints
	}
	divisor := (1 + math.Pow((float64(solvers)-1)/11.92201, 1.206069))
	return uint(math.Round(float64(minPoints) + (float64(maxPoints)-float64(minPoints))/divisor))
}

// updatePointsOfSolvers updates the points of solvers, whenever points of challenge changes
func updatePointsOfSolvers(submissions []database.UserChallenges, newChallengePointsAfterSolve, oldChallengePointsBeforeSolve uint) error {
	for _, submission := range submissions {
		user, err := database.QueryUserById(submission.UserID)
		if err != nil {
			return err
		}
		if user.Role == "contestant" {
			err = database.UpdateUser(&user, map[string]interface{}{"Score": user.Score + (newChallengePointsAfterSolve - oldChallengePointsBeforeSolve)})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
