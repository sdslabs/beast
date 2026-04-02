package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/utils"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	fileUtils "github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

var (
	leaderboardCache      []UserResp
	leaderboardStale      = true
	adminLeaderboardCache []UserResp
	adminLeaderboardStale = true
	graphCache            []database.UserLeaderboardResp
	graphCacheStale       = true
)

func hintHandler(c *gin.Context) {
	hintIDStr := c.Param("hintID")

	if hintIDStr == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Hint ID cannot be empty",
		})
		return
	}

	hintID, err := strconv.Atoi(hintIDStr)

	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Hint Id format invalid",
		})
		return
	}

	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
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

	// Fetch hint details
	hint, err := database.GetHintByID(uint(hintID))
	if err != nil {
		if err.Error() == "not_found" {
			c.JSON(http.StatusNotFound, HTTPErrorResp{
				Error: "Hint not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request",
			})
		}
		return
	}

	if user.Role == core.USER_ROLES["admin"] {
		c.JSON(http.StatusOK, HintResponse{
			Description: hint.Description,
			Points:      hint.Points,
		})
		return
	}

	// Check if the user has already taken the hint
	hasTakenHint, err := database.UserHasTakenHint(user.ID, uint(hintID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while checking hint usage",
		})
		return
	}

	if c.Request.Method == "GET" {
		if hasTakenHint {
			c.JSON(http.StatusOK, HintResponse{
				Description: hint.Description,
				Points:      hint.Points,
			})
		} else {
			c.JSON(http.StatusOK, HintResponse{
				Description: "Hint is not taken yet",
				Points:      hint.Points,
			})
		}
		return
	}

	if hasTakenHint {
		// If hint already taken, just return the description
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: hint.Description,
		})
		return
	}

	// Save user hint if not already taken
	if err := database.SaveUserHint(user.ID, hint.ChallengeID, hint.HintID); err != nil {
		if err.Error() == "Not enough points to take this hint" {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "You don't have enough points to take this hint",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while saving the hint usage",
		})
		return
	}
	
	oldScore := user.Score
	newScore := oldScore - hint.Points
	if newScore < 0 {
		newScore = 0
	}

	if len(adminLeaderboardCache) < core.LEADERBOARD_SIZE ||
		(len(adminLeaderboardCache) > 0 && oldScore >= adminLeaderboardCache[len(adminLeaderboardCache)-1].Score) {
		leaderboardStale = true
		graphCacheStale = true
		adminLeaderboardStale = true
	}

	// Return the hint description after successfully taking it
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: hint.Description,
	})
}

// Returns information about a challenge
// @Summary Returns all information about the challenges.
// @Description Returns all information about the challenges by the challenge name.
// @Tags info
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param name query string true "Name of challenge"
// @Success 200 {object} api.ChallengeInfoResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/challenge/info [get]
func challengeInfoHandler(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Challenge name cannot be empty",
		})
		return
	}

	challenges, err := database.QueryChallengeEntries("name", name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	authHeader := c.GetHeader("Authorization")
	username, err := coreUtils.GetUser(authHeader)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "No Token Provided",
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

	if len(challenges) > 0 && challenges[0].Status != "Undeployed" {
		challenge := challenges[0]
		totalSolves, solveStatus, err := database.GetChallengeSolveInfo(challenge.ID, user.ID)
		if err != nil {
			log.Error(err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		challengeTags := make([]string, len(challenge.Tags))

		for index, tags := range challenge.Tags {
			challengeTags[index] = tags.TagName
		}

		hints, err := database.QueryHintsByChallengeID(challenge.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while fetching hints.",
			})
			return
		}

		hintInfos := make([]HintInfo, len(hints))
		for i, hint := range hints {
			hintInfos[i] = HintInfo{
				ID:     hint.HintID,
				Points: hint.Points,
			}
		}

		previousTries, err := database.GetUserPreviousTries(user.ID, challenge.ID)
		if err != nil {
			log.Error(err)
			previousTries = 0

		}
		challMetadata := ChallengeMetadata{
			Name:           challenge.Name,
			ChallId:        challenge.ID,
			Tags:           challengeTags,
			CreatedAt:      challenge.CreatedAt,
			Points:         challenge.Points,
			SolvesNumber:   totalSolves,
			SolveStatus:    solveStatus,
			Difficulty:     challenge.Difficulty,
			PreRequisite:   strings.Split(challenge.PreReqs, core.DELIMITER),
			DeployedStatus: challenge.Status,
		}
		challengeInfo := Challenge{
			ChallengeMetadata: challMetadata,
			Description:       challenge.Description,
			Hints:             hintInfos,
			Category:          challenge.Type,
			Assets:            strings.Split(challenge.Assets, core.DELIMITER),
			AdditionalLinks:   strings.Split(challenge.AdditionalLinks, core.DELIMITER),
			PreviousTries:     previousTries,
			MaxAttemptLimit:   challenge.MaxAttemptLimit,
			DeployedLink:      challenge.ServerDeployed,
		}
		if user.Role == core.USER_ROLES["contestant"] {
			c.JSON(http.StatusOK, challengeInfo)
			return
		}

		c.JSON(http.StatusOK, AdminChallenge{
			Challenge:   challengeInfo,
			DynamicFlag: challenge.DynamicFlag,
			Flag:        challenge.Flag,
		})
	} else {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No challenge found with name: " + name,
		})
	}
}

// Returns metadata about all challenges with and without filters
// @Summary Returns metadata about all challenges with and without filters.
// @Description Returns information about all the challenges present in the database with and without filters.
// @Tags info
// @Accept  json
// @Produce json
// @Param filter query string false "Filter parameter by which challenges are filtered"
// @Param value query string false "Value of filtered parameter"
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.ChallengeInfoResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/challenges [get]
func challengesMetadataHandler(c *gin.Context) {
	filter := c.Query("filter")
	value := c.Query("value")

	var challenges []database.Challenge
	var err error

	err, state := utils.CheckTime()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	authHeader := c.GetHeader("Authorization")

	values := strings.Split(authHeader, " ")

	if len(values) < 2 || values[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "No Token Provided",
		})
		c.Abort()
		return
	}

	autherr := auth.Authorize(values[1], core.ADMIN)

	// If competition is yet to start returns 0
	if state == 0 && autherr != nil {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "0",
		})
		return
	}
	// If competition has ended returns 2
	if state == 2 && autherr != nil {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "2",
		})
		return
	}
	// If comp in ongoing
	if state == 1 || autherr == nil {
		if (filter == "name" || filter == "author" || filter == "score") && value != "" {
			challenges, err = database.QueryChallengeEntriesMetadata(filter, value)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
			}
		} else if filter == "tag" && value != "" {
			tag := database.Tag{
				TagName: value,
			}
			challenges, err = database.QueryRelatedChallengesMetadata(&tag)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
			}
		} else {
			challenges, err = database.QueryAllChallengesMetadata()
			if err != nil {
				c.JSON(http.StatusBadRequest, HTTPErrorResp{
					Error: err.Error(),
				})
				return
			}
			if challenges == nil {
				c.JSON(http.StatusOK, HTTPPlainResp{
					Message: "No challenges currently in the database",
				})
				return
			}
		}

		availableChallenges := make([]ChallengeMetadata, len(challenges))

		authHeader := c.GetHeader("Authorization")
		username, err := coreUtils.GetUser(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "No Token Provided",
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

		for index, challenge := range challenges {
			if challenge.Status == "Undeployed" && user.Role == core.USER_ROLES["contestant"] {
				continue
			}
			totalSolves, solveStatus, err := database.GetChallengeSolveInfo(challenge.ID, user.ID)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
			challengeTags := make([]string, len(challenge.Tags))

			for index, tags := range challenge.Tags {
				challengeTags[index] = tags.TagName
			}

			availableChallenges[index] = ChallengeMetadata{
				Name:               challenge.Name,
				ChallId:            challenge.ID,
				Tags:               challengeTags,
				CreatedAt:          challenge.CreatedAt,
				Points:             challenge.Points,
				SolvesNumber:       totalSolves,
				SolveStatus:        solveStatus,
				Difficulty:         challenge.Difficulty,
				PreRequisite:       strings.Split(challenge.PreReqs, core.DELIMITER),
				DeployedStatus:     challenge.Status,
				Instanced:          challenge.Instanced,
				InstanceExpiration: challenge.InstanceExpiration,
			}
		}

		c.JSON(http.StatusOK, availableChallenges)
		return
	}
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
			Message: fmt.Sprint("challenge name cannot be empty"),
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

// Returns user info
// @Summary Returns user info
// @Description Returns user info based on userId
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param value formData string false "User's id"
// @Param value query string false "username"
// @Success 200 {object} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/user [get]
func userInfoHandler(c *gin.Context) {
	userId := c.PostForm("user_id")
	username := c.Param("username")
	if userId == "" && username == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: fmt.Sprintf("Both User Id and Username cannot be empty"),
		})
		return
	}
	var user database.User
	var err error
	var parsedUserId uint
	if userId != "" {
		id, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPErrorResp{
				Error: fmt.Sprintf("Could not parse User Id or invalid User Id"),
			})
			return
		}
		parsedUserId = uint(id)

		user, err = database.QueryUserById(parsedUserId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
	} else {
		user, err = database.QueryFirstUserEntry("username", username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
	}

	challenges, err := database.GetRelatedChallenges(&user)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	var resp UserResp

	var challNameString []string
	for _, challenge := range challenges {
		challNameString = append(challNameString, challenge.Name)
	}

	userChallenges := make([]ChallengeSolveResp, len(challenges))
	for index, challenge := range challenges {

		challengeTags := make([]string, len(challenge.Tags))

		for index, tags := range challenge.Tags {
			challengeTags[index] = tags.TagName
		}

		challResp := ChallengeSolveResp{
			Id:       challenge.ID,
			Name:     challenge.Name,
			Tags:     challengeTags,
			Category: challenge.Type,
			SolvedAt: challenge.CreatedAt,
			Points:   challenge.Points,
		}
		userChallenges[index] = challResp
	}

	var rank int64
	if user.Status == 0 {
		rank, err = database.GetUserRank(parsedUserId, user.Score, user.UpdatedAt)
	} else {
		rank = 1e9
	}

	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	resp = UserResp{
		Username:   user.Username,
		Id:         user.ID,
		Role:       user.Role,
		Status:     user.Status,
		Score:      user.Score,
		Rank:       rank,
		Email:      user.Email,
		Challenges: userChallenges,
	}
	c.JSON(http.StatusOK, resp)
	return
}

// Returns all user's info
// @Summary Returns all user's info
// @Description Returns all available user's info
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param sort, order, filter
// @Success 200 {object} api.UserResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/users [get]
func getAllUsersInfoHandler(c *gin.Context) {
	users, err := database.QueryUserEntries("role", "contestant")
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	availableUsers := make([]UsersResp, len(users))
	if len(users) > 0 {
		for index, user := range users {

			parsedUserId := uint(user.ID)

			var rank int64
			if user.Status == 0 {
				rank, err = database.GetUserRank(parsedUserId, user.Score, user.UpdatedAt)
			} else {
				rank = 1e9
			}

			if err != nil {
				log.Error(err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}

			availableUsers[index] = UsersResp{
				Username: user.Username,
				Id:       user.ID,
				Role:     user.Role,
				Status:   user.Status,
				Score:    user.Score,
				Email:    user.Email,
				Rank:     rank,
			}
		}

		// sort the availableUsers according to the given params
		sortParam := c.Query("sort")
		orderParam := c.Query("order")

		if sortParam == "username" {
			sort.Slice(availableUsers, func(i, j int) bool {
				return availableUsers[i].Username < availableUsers[j].Username
			})
		} else if sortParam == "score" {
			sort.Slice(availableUsers, func(i, j int) bool {
				if orderParam == "asc" {
					return availableUsers[i].Score < availableUsers[j].Score
				}
				return availableUsers[i].Score > availableUsers[j].Score
			})
		}

		// filter the availableUsers according to the given params
		filterParam := c.Query("filter")

		var filteredUsers []UsersResp

		if filterParam == "banned" {
			for _, user := range availableUsers {
				if user.Status == 1 {
					filteredUsers = append(filteredUsers, user)
				}
			}
		} else if filterParam == "active" {
			for _, user := range availableUsers {
				if user.Status == 0 {
					filteredUsers = append(filteredUsers, user)
				}
			}
		} else if filterParam == "hidden" {
			for _, user := range availableUsers {
				if user.Status == 2 {
					filteredUsers = append(filteredUsers, user)
				}
			}
		} else {
			filteredUsers = make([]UsersResp, len(availableUsers))
			copy(filteredUsers, availableUsers)
		}

		format := c.Query("format")

		if format == "csv" {
			buff, err := utils.StructToCSV(c, filteredUsers, "users.csv")

			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "CSV ERROR while processing the request.",
				})
				return
			}

			c.Data(http.StatusOK, "text/csv", buff.Bytes())
			return
		}

		c.JSON(http.StatusOK, filteredUsers)
	} else {
		c.JSON(http.StatusOK, availableUsers)
	}

	return
}

// Handles submissions made by the user
// @Summary Handles submissions made by the user
// @Description Handles submissions made by the user
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.SubmissionResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/admin/submissions [get]
func submissionsHandler(c *gin.Context) {
	pageStr := c.Query("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid page number",
		})
		return
	}

	offset := (page - 1) * core.SUBMISSIONS_PAGE_SIZE

	submissions, err := database.QuerySubmissionsWithPagination(core.SUBMISSIONS_PAGE_SIZE, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	submissionsResp := make([]SubmissionResp, 0)

	for _, submission := range submissions {
		user, err := database.QueryUserById(submission.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while fetching user details.",
			})
			return
		}

		challenge, err := database.QueryChallengeEntries("id", strconv.Itoa(int(submission.ChallengeID)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while fetching challenge details.",
			})
			return
		}
		if len(challenge) == 0 {
			continue
		}

		challengeTags := make([]string, len(challenge[0].Tags))

		for index, tags := range challenge[0].Tags {
			challengeTags[index] = tags.TagName
		}

		singleSubmissionResp := SubmissionResp{
			UserId:    user.ID,
			Username:  user.Username,
			ChallId:   challenge[0].ID,
			ChallName: challenge[0].Name,
			Flag:      submission.Flag,
			SolvedAt:  submission.CreatedAt,
			Success:   submission.Solved,
			Cheating:  submission.Cheating,
		}

		submissionsResp = append(submissionsResp, singleSubmissionResp)
	}

	format := c.Query("format")
	if format == "csv" {
		buff, err := utils.StructToCSV(c, submissionsResp, "submissions.csv")

		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "CSV ERROR while processing the request.",
			})
			return
		}

		c.Data(http.StatusOK, "text/csv", buff.Bytes())
		return
	}

	c.JSON(http.StatusOK, submissionsResp)
	return
}

// Returns statistics of users in competition
// @Summary statistics of users in competition
// @Description returns statistics of users in competition (currently limited to ban/unban status of users)
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.UsersStatisticsResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/admin/statistics [get]
func getUsersStatisticsHandler(c *gin.Context) {
	users, err := database.QueryAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	var totalRegisteredUsers uint
	var bannedUsers uint

	if len(users) <= 0 {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No users found in the database",
		})
	}

	for _, user := range users {
		if user.Role == core.USER_ROLES["contestant"] {
			if user.Status == 1 {
				bannedUsers++
			}
			totalRegisteredUsers++
		}
	}

	c.JSON(http.StatusOK, UsersStatisticsResp{
		TotalRegisteredUsers: totalRegisteredUsers,
		BannedUsers:          bannedUsers,
		UnbannedUsers:        totalRegisteredUsers - bannedUsers,
	})

	return
}

// Returns competition information
// @Summary returns competition info
// @Description returns various information about the competition which are used to control competition
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.CompetitionInfoResp
// @Failure 400 {object} api.HTTPErrorResp
// @Router /api/admin/statistics [get]
func competitionInfoHandler(c *gin.Context) {
	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	logoPath := strings.ReplaceAll(competitionInfo.LogoURL, filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_LOGO_DIR), "")
	c.JSON(http.StatusOK, CompetitionInfoResp{
		Name:         competitionInfo.Name,
		About:        competitionInfo.About,
		Prizes:       competitionInfo.Prizes,
		StartingTime: competitionInfo.StartingTime,
		EndingTime:   competitionInfo.EndingTime,
		TimeZone:     competitionInfo.TimeZone,
		LogoURL:      strings.Trim(logoPath, "/"),
	})
	return
}

// Returns allTags
// @Summary returns all tags
// @Description returns all unique tags
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.TagInfoResp
// @Failure 400 {object} api.HTTPErrorResp
// @Router /api/admin/statistics [get]
func tagHandler(c *gin.Context) {
	// Optimized: Query unique tags directly from the database
	tags, err := database.QueryAllUniqueTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while fetching tags.",
		})
		return
	}

	c.JSON(http.StatusOK, TagInfoResp{
		Tags: tags,
	})
}

// @Tags info
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param challenge query string false "The name of the challenge to get the logs for."
// @Param asset query string false "The name of the static asset requested."
// @Success 200 {object} api.LogsInfoResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/info/download [get]
func serveAssets(c *gin.Context) {
	challenge := c.Query("challenge")
	assetName := c.Query("asset")
	challenge = filepath.Base(challenge)
	assetName = filepath.Base(assetName)
	if challenge == "" || assetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "challenge and asset parameters are required"})
		return
	}
	filepath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challenge, core.BEAST_STATIC_FOLDER, assetName)
	err := fileUtils.ValidateFileExists(filepath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect file requested"})
		return
	}
	c.FileAttachment(filepath, assetName)

}

// This route returns the number of users in the databse with role=contestant
// @Summary Returns the number of users in the database with role=contestant
// @Description Returns the number of users in the database with role=contestant
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.UserCountResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/usercount [get]
func getUserCountHandler(c *gin.Context) {
	count, err := database.GetUserCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	c.JSON(http.StatusOK, UserCountResp{
		UserCount: count,
	})
}

// Returns leaderboard
// @Summary Returns leaderboard
// @Description Returns leaderboard of all users
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param page query string false "Page number"
// @Success 200 {object} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/leaderboard [get]
func getLeaderboardHandler(c *gin.Context) {
	pageStr := c.Query("page")
	log.Print(pageStr)
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid page number",
		})
		return
	}
	isLeaderboardFrozen, err := database.IsFrozenScoreSet()
	if err != nil {
		log.Errorf("DATABASE ERROR:: Unable to lookup FrozenLeaderboard %s. Continuing with normal leaderboard", err.Error())
	}
	if isLeaderboardFrozen {
		if page == 1 {
			if leaderboardStale {
				users, err := database.QueryTopUsersByFrozenScore(core.LEADERBOARD_SIZE)
				if err != nil {
					c.JSON(http.StatusInternalServerError, HTTPErrorResp{
						Error: "DATABASE ERROR while processing the request.",
					})
					return
				}
				var leaderboard []UserResp
				for index, user := range users {
					if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
						continue
					}
					resp := UserResp{
						Username: user.Username,
						Id:       user.ID,
						Role:     user.Role,
						Status:   user.Status,
						Score:    user.FrozenScore,
						Rank:     int64(index + 1),
					}
					leaderboard = append(leaderboard, resp)
				}
				leaderboardCache = leaderboard
				leaderboardStale = false
			}
			c.JSON(http.StatusOK, leaderboardCache)
			return
		}
		offset := (page - 1) * core.LEADERBOARD_SIZE
		users, err := database.QueryUsersByFrozenScoreOffsetLimit(core.LEADERBOARD_SIZE, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		var leaderboard []UserResp
		rankOffset := offset
		for index, user := range users {
			if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
				continue
			}
			resp := UserResp{
				Username: user.Username,
				Id:       user.ID,
				Role:     user.Role,
				Status:   user.Status,
				Score:    user.FrozenScore,
				Rank:     int64(rankOffset + index + 1),
			}
			leaderboard = append(leaderboard, resp)
		}
		c.JSON(http.StatusOK, leaderboard)
		return
	}

	if page == 1 {
		if leaderboardStale {
			users, err := database.QueryTopUsersByScore(core.LEADERBOARD_SIZE)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
			var leaderboard []UserResp
			for index, user := range users {
				if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
					continue
				}
				resp := UserResp{
					Username: user.Username,
					Id:       user.ID,
					Role:     user.Role,
					Status:   user.Status,
					Score:    user.Score,
					Rank:     int64(index + 1),
				}
				leaderboard = append(leaderboard, resp)
			}
			leaderboardCache = leaderboard
			leaderboardStale = false
		}
		c.JSON(http.StatusOK, leaderboardCache)
		return
	}
	offset := (page - 1) * core.LEADERBOARD_SIZE
	users, err := database.QueryUsersByScoreOffsetLimit(core.LEADERBOARD_SIZE, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	var leaderboard []UserResp
	rankOffset := offset
	for index, user := range users {
		if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
			continue
		}
		resp := UserResp{
			Username: user.Username,
			Id:       user.ID,
			Role:     user.Role,
			Status:   user.Status,
			Score:    user.Score,
			Rank:     int64(rankOffset + index + 1),
		}
		leaderboard = append(leaderboard, resp)
	}
	c.JSON(http.StatusOK, leaderboard)
}

// Returns admin leaderboard
// @Summary Returns admin leaderboard
// @Description Returns admin leaderboard of all users
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Param page query string false "Page number"
// @Success 200 {object} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/admin/leaderboard [get]
func adminLeaderboardHandler(c *gin.Context) {
	pageStr := c.Query("page")
	log.Print(pageStr)
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid page number",
		})
		return
	}
	if page == 1 {
		if adminLeaderboardStale {
			users, err := database.QueryTopUsersByScore(core.LEADERBOARD_SIZE)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
			var leaderboard []UserResp
			for index, user := range users {
				if user.Status != 0 {
					continue
				}
				resp := UserResp{
					Username: user.Username,
					Id:       user.ID,
					Role:     user.Role,
					Status:   user.Status,
					Score:    user.Score,
					Email:    user.Email,
					Rank:     int64(index + 1),
				}
				leaderboard = append(leaderboard, resp)
			}
			adminLeaderboardCache = leaderboard
			adminLeaderboardStale = false
		}
		c.JSON(http.StatusOK, adminLeaderboardCache)
		return
	}
	offset := (page - 1) * core.LEADERBOARD_SIZE
	users, err := database.QueryUsersByScoreOffsetLimit(core.LEADERBOARD_SIZE, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	var leaderboard []UserResp
	rankOffset := offset
	for index, user := range users {
		if user.Status != 0 {
			continue
		}
		resp := UserResp{
			Username: user.Username,
			Id:       user.ID,
			Role:     user.Role,
			Status:   user.Status,
			Score:    user.Score,
			Email:    user.Email,
			Rank:     int64(rankOffset + index + 1),
		}
		leaderboard = append(leaderboard, resp)
	}
	c.JSON(http.StatusOK, leaderboard)
}

// Freeze user leaderboard
// @Summary Freeze user leaderboard
// @Description freezes the user leaderboard on demand.
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/admin/freezeLeaderboard [post]
func freezeLeaderboardHandler(c *gin.Context) {
	err := database.UpdateFrozenScores()
	if err != nil {
		log.Errorf("DATABASE ERROR while freezing the leaderboard. %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
	}
	leaderboardStale = true
	graphCacheStale = true
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User leaderboard frozen successfully",
	})
}

// Unfreeze user leaderboard
// @Summary Unfreeze user leaderboard
// @Description unfreezes the user leaderboard on demand.
// @Tags info
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/admin/unfreezeLeaderboard [post]
func unfreezeLeaderboardHandler(c *gin.Context) {
	err := database.ResetFrozenScores()
	if err != nil {
		log.Errorf("DATABASE ERROR while un-freezing the leaderboard. %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
	}
	leaderboardStale = true
	graphCacheStale = true
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User leaderboard unfrozen successfully",
	})
}

// Fetch all the attempts on this challenge
// @Summary Get challenge attempts
// @Description Returns all user attempts for a given challenge.
// @Tags info
// @Accept json
// @Produce json
// @Param challenge_id path int true "Challenge ID"
// @Param Authorization header string true "Bearer"
// @Success 200 {array} api.UserSolveResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/challenges/{challenge_id}/attempts [get]
func getChallengeAttempts(c *gin.Context) {
	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
		})
		return
	}

	queryingUser, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
		})
		return
	}

	isContestant := queryingUser.Role == core.USER_ROLES["contestant"]

	challengeIDStr := c.Param("challenge_id")
	challengeID, err := strconv.ParseUint(challengeIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Invalid challenge_id",
		})
		return
	}

	challenge, err := database.QueryChallengeEntries("id", challengeIDStr)
	if err != nil {
		log.Errorf("DATABASE ERROR while fetching challenge details: %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	if len(challenge) == 0 {
		c.JSON(http.StatusNotFound, HTTPPlainResp{
			Message: "Challenge not found",
		})
		return
	}

	challengeTags := make([]string, len(challenge[0].Tags))
	for index, tag := range challenge[0].Tags {
		challengeTags[index] = tag.TagName
	}

	attempts, err := database.QueryChallAttempts(challengeID)
	if err != nil {
		log.Errorf("DATABASE ERROR while fetching challenge attempts: %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	resp := make([]SubmissionResp, 0, len(attempts))
	for _, attempt := range attempts {
		if isContestant && (!attempt.Correct || attempt.Cheating) {
			continue
		}

		submissionResp := SubmissionResp{
			UserId:    attempt.UserId,
			Username:  attempt.Username,
			ChallId:   challenge[0].ID,
			ChallName: challenge[0].Name,
			SolvedAt:  attempt.SolvedAt,
			Success:   attempt.Correct,
		}

		if !isContestant {
			submissionResp.Flag = attempt.Flag
			submissionResp.Cheating = attempt.Cheating
		}

		resp = append(resp, submissionResp)
	}
	c.JSON(http.StatusOK, resp)
}

// Get submissions by user ID
// @Summary Get submissions by user
// @Description Returns all submissions for a specific user
// @Tags info
// @Accept json
// @Produce json
// @Param user_id path int true "User ID"
// @Param Authorization header string true "Bearer"
// @Success 200 {array} api.SubmissionResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/info/submissions/user/{user_id} [get]
func getUserAttempts(c *gin.Context) {
	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
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

	isContestant := user.Role == core.USER_ROLES["contestant"]

	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid user_id",
		})
		return
	}

	submissionUser, err := database.QueryUserById(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "User not found",
		})
		return
	}

	if submissionUser.Role != core.USER_ROLES["contestant"] {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Can only view submissions for contestants",
		})
		return
	}

	attempts, err := database.QueryUserAttempts(uint(userID))
	if err != nil {
		log.Errorf("DATABASE ERROR while fetching user attempts: %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	resp := make([]SubmissionResp, 0, len(attempts))

	for _, attempt := range attempts {
		if isContestant && (!attempt.Correct || attempt.Cheating) {
			continue
		}

		challenge, err := database.QueryChallengeEntries("id", strconv.Itoa(int(attempt.ChallengeID)))
		if err != nil {
			log.Errorf("DATABASE ERROR while fetching challenge details: %s", err.Error())
			continue
		}
		if len(challenge) == 0 {
			continue
		}

		challengeTags := make([]string, len(challenge[0].Tags))
		for index, tag := range challenge[0].Tags {
			challengeTags[index] = tag.TagName
		}

		submissionResp := SubmissionResp{
			UserId:    submissionUser.ID,
			Username:  submissionUser.Username,
			ChallId:   challenge[0].ID,
			ChallName: challenge[0].Name,
			SolvedAt:  attempt.SolvedAt,
			Success:   attempt.Correct,
		}

		if !isContestant {
			submissionResp.Flag = attempt.Flag
			submissionResp.Cheating = attempt.Cheating
		}

		resp = append(resp, submissionResp)
	}

	c.JSON(http.StatusOK, resp)
}

func getLeaderboardGraphHandler(c *gin.Context) {
	var topUsers []uint
	// TODO: Add a check for leaderboard stale to prevent stale graphs
	// Try if graphCache and leaderboardCache can be merged.
	// Right now graph cache gets invalidated whenver leaderboardCache gets invalidated even if no user under top 10 are changed. Fix later.
	if leaderboardStale {
		users, err := database.QueryTopUsersByFrozenScore(core.LEADERBOARD_SIZE)
		if err == nil {
			for i := 0; i < 10 && i < len(users); i++ {
				user := users[i]
				topUsers = append(topUsers, user.ID)
			}
		}
	} else {
		for i := 0; i < 10 && i < len(leaderboardCache); i++ {
			user := leaderboardCache[i]
			topUsers = append(topUsers, user.Id)
		}
	}
	// If leaderboard is frozen then directly send the last graph instance without updating
	isLeaderboardFrozen, _ := database.IsFrozenScoreSet()
	if !graphCacheStale || isLeaderboardFrozen {
		c.JSON(http.StatusOK, graphCache)
	} else {
		// TODO: Add a fallback for frozen leaderboard as graphcache is in memory and not persistent. SO, might get lost if server got down in between.
		graphCache = database.QueryTimeSeriesForTopUsers(topUsers)
		graphCacheStale = false
		c.JSON(http.StatusOK, graphCache)
	}

}
