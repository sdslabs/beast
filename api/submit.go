package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/notify"
	log "github.com/sirupsen/logrus"
)

var (
	dynamicScoreNotify = make(chan struct{}, 1)
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
	now := time.Now()

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
	if state != 1 {
		return
	}

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
	if err != nil || user.ID == 0 {
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
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid challenge id.",
		})
		return
	}

	chall, err := database.QueryChallengeEntries("id", strconv.Itoa(parsedChallId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	if len(chall) == 0 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Challenge not found.",
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

	if challenge.PreReqs != "" {
		preReqsStatus, err := database.CheckPreReqsStatus(challenge, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}

		if !preReqsStatus {
			c.JSON(http.StatusOK, FlagSubmitResp{
				Message: "You have not solved the prerequisites of this challenge.",
				Success: false,
			})
			return
		}
	}

	attempt, err := database.ReserveSubmissionAttempt(user.ID, challenge.ID, challenge.MaxAttemptLimit, flag, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	switch attempt.Status {
	case database.SubmissionAttemptAlreadySolved:
		c.JSON(http.StatusOK, FlagSubmitResp{
			Message: "Challenge has already been solved.",
			Success: false,
		})
		return
	case database.SubmissionAttemptMaxAttempts:
		c.JSON(http.StatusOK, FlagSubmitResp{
			Message: "You have reached the maximum number of tries for this challenge.",
			Success: false,
		})
		return
	}

	isCheating := false
	if challenge.DynamicFlag {
		validFlags, err := database.QueryDynamicFlagEntries(map[string]interface{}{
			"Name": challenge.Name,
			"Flag": flag,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		if len(validFlags) == 0 {
			c.JSON(http.StatusOK, FlagSubmitResp{
				Message: "Your flag is incorrect",
				Success: false,
			})
			return
		}

		claim, err := database.ClaimDynamicFlag(challenge.ID, user.ID, flag, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		if claim.Status == database.DynamicFlagClaimedByOtherUser {
			claimedBy := fmt.Sprintf("ID %d", claim.ClaimedByID)
			if subuser, lookupErr := database.QueryUserById(claim.ClaimedByID); lookupErr != nil {
				log.Warnf("failed to resolve dynamic flag claimant %d: %v", claim.ClaimedByID, lookupErr)
			} else {
				claimedBy = subuser.Username
			}
			msg := fmt.Sprintf("User %s submitted a duplicate dynamic flag for challenge %s already claimed by user %s", user.Username, challenge.Name, claimedBy)
			go notify.SendNotification(notify.Warning, msg)
			if err := database.MarkSubmissionCheating(user.ID, challenge.ID, flag); err != nil {
				log.Warnf("failed to mark duplicate dynamic flag submission as cheating: %v", err)
			}
			c.JSON(http.StatusOK, FlagSubmitResp{
				Message: "This dynamic flag has already been claimed.",
				Success: false,
			})
			return
		}
	} else if challenge.Flag != flag {
		c.JSON(http.StatusOK, FlagSubmitResp{
			Message: "Your flag is incorrect",
			Success: false,
		})
		return
	}

	wonSolveRace, err := database.MarkSubmissionSolved(user.ID, challenge.ID, flag, isCheating, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	if !wonSolveRace {
		c.JSON(http.StatusOK, FlagSubmitResp{
			Message: "Challenge has already been solved.",
			Success: false,
		})
		return
	}

	challengePoints := challenge.Points
	if err := database.AwardUserScore(user.ID, int64(challengePoints)); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	log.Debugf("Dynamic scoring is set to %t", config.Cfg.CompetitionInfo.DynamicScore)
	if config.Cfg.CompetitionInfo.DynamicScore {
		if err := database.MarkDynamicScoreDirty(challenge.ID, user.ID, now); err != nil {
			log.Errorf("failed to mark dynamic score dirty for challenge %s: %v", challenge.Name, err)
		} else {
			notifyDynamicScoreWorker()
		}
	}

	markLeaderboardCachesStale()

	c.JSON(http.StatusOK, FlagSubmitResp{
		Message: "Your flag is correct",
		Success: true,
	})
}

// dynamicScore returns dynamic score of the challenge based on number of solves
func dynamicScore(maxPoints, minPoints, solvers uint) uint {
	if solvers == 0 || solvers == 1 {
		return maxPoints
	}
	divisor := (1 + math.Pow((float64(solvers)-1)/11.92201, 1.206069))
	return uint(math.Round(float64(minPoints) + (float64(maxPoints)-float64(minPoints))/divisor))
}

func scoreAfterPointChange(currentScore, newPoints, oldPoints uint) uint {
	score := int64(currentScore) + int64(newPoints) - int64(oldPoints)
	if score < 0 {
		return 0
	}
	return uint(score)
}

func startDynamicScoreWorker(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-dynamicScoreNotify:
				processDirtyDynamicScores()
			case <-ticker.C:
				processDirtyDynamicScores()
			}
		}
	}()
	return done
}

func notifyDynamicScoreWorker() {
	select {
	case dynamicScoreNotify <- struct{}{}:
	default:
	}
}

func processDirtyDynamicScores() {
	dirtyScores, err := database.QueryDirtyDynamicScores(100)
	if err != nil {
		log.Errorf("failed to query dirty dynamic scores: %v", err)
		return
	}

	for _, dirty := range dirtyScores {
		if err := recomputeDynamicScore(dirty); err != nil {
			log.Errorf("failed to recompute dynamic score for challenge %d: %v", dirty.ChallengeID, err)
			continue
		}
		if err := database.ClearDynamicScoreDirty(dirty.ChallengeID, dirty.UpdatedAt); err != nil {
			log.Errorf("failed to clear dynamic score dirty marker for challenge %d: %v", dirty.ChallengeID, err)
		}
	}
}

func recomputeDynamicScore(dirty database.DynamicScoreDirty) error {
	challs, err := database.QueryChallengeEntries("id", strconv.Itoa(int(dirty.ChallengeID)))
	if err != nil {
		return err
	}
	if len(challs) == 0 {
		return nil
	}

	challenge := challs[0]
	if !challenge.DynamicFlag {
		return nil
	}

	solvers, err := database.CountSolvedSubmissionsForChallenge(challenge.ID)
	if err != nil {
		return err
	}

	newPoints := dynamicScore(challenge.MaxPoints, challenge.MinPoints, solvers)
	delta := int64(newPoints) - int64(challenge.Points)
	if err := database.ApplyDynamicScoreDelta(challenge.ID, newPoints, delta); err != nil {
		return err
	}

	if delta != 0 {
		log.Debugf("By dynamic scoring the points of challenge %s are changed to %d from %d", challenge.Name, newPoints, challenge.Points)
		markLeaderboardCachesStale()
	}

	return nil
}

// updatePointsOfSolvers updates the points of solvers, whenever points of challenge changes
func updatePointsOfSolvers(submissions []database.UserChallenges, newChallengePointsAfterSolve, oldChallengePointsBeforeSolve uint) error {
	scoreChanged := false
	for _, submission := range submissions {
		user, err := database.QueryUserById(submission.UserID)
		if err != nil {
			return err
		}
		if user.Role == "contestant" {
			newScore := scoreAfterPointChange(user.Score, newChallengePointsAfterSolve, oldChallengePointsBeforeSolve)
			err = database.UpdateUser(&user, map[string]interface{}{"Score": newScore})
			if err != nil {
				return err
			}
			scoreChanged = true
		}
	}
	// Mark cache stale if any user's score change could affect top 25
	if scoreChanged {
		markLeaderboardCachesStale()
	}
	return nil
}
