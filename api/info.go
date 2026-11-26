package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	fileUtils "github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

var (
	leaderboardCacheMu    sync.RWMutex
	leaderboardCache      = make([]UserResp, 0)
	leaderboardStale      = true
	adminLeaderboardCache = make([]UserResp, 0)
	adminLeaderboardStale = true
	graphCache            = make([]database.UserLeaderboardResp, 0)
	graphCacheStale       = true
)

func invalidateLeaderboardCaches() {
	leaderboardCacheMu.Lock()
	leaderboardStale = true
	graphCacheStale = true
	adminLeaderboardStale = true
	leaderboardCacheMu.Unlock()
}

func invalidatePublicLeaderboardCaches() {
	leaderboardCacheMu.Lock()
	leaderboardStale = true
	graphCacheStale = true
	leaderboardCacheMu.Unlock()
}

func invalidateLeaderboardCachesIfSmall() {
	leaderboardCacheMu.Lock()
	if len(adminLeaderboardCache) < core.LEADERBOARD_SIZE {
		leaderboardStale = true
		graphCacheStale = true
		adminLeaderboardStale = true
	}
	leaderboardCacheMu.Unlock()
}

func invalidateLeaderboardCachesForScoreChange(oldScore uint) {
	leaderboardCacheMu.Lock()
	if len(adminLeaderboardCache) < core.LEADERBOARD_SIZE ||
		(len(adminLeaderboardCache) > 0 && oldScore >= adminLeaderboardCache[len(adminLeaderboardCache)-1].Score) {
		leaderboardStale = true
		graphCacheStale = true
		adminLeaderboardStale = true
	}
	leaderboardCacheMu.Unlock()
}

func extractTagNames(tags []*database.Tag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag != nil && strings.TrimSpace(tag.TagName) != "" {
			result = append(result, strings.TrimSpace(tag.TagName))
		}
	}
	return result
}

// @Summary Inspect or unlock a hint
// @Tags info
// @Description GET shows hint metadata or an unlocked hint. POST unlocks a hint and deducts the configured points when applicable.
// @Produce json
// @Security ApiKeyAuth
// @Param hintID path int true "Hint ID"
// @Success 200 {object} api.HintResponse
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 401 {object} api.HTTPErrorResp
// @Failure 404 {object} api.HTTPErrorResp
// @Router /api/info/hint/{hintID} [get]
// @Router /api/info/hint/{hintID} [post]
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

	username, err := resolveRequestUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
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

	invalidateLeaderboardCachesForScoreChange(user.Score)

	// Return the hint description after successfully taking it
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: hint.Description,
	})
}

// Returns information about a challenge
// @Summary Returns all information about the challenges.
// @Description Returns challenge metadata and allocated host ports. For instanced challenges, the ports belong to the caller's active instance.
// @Tags info
// @Produce json
// @Param name path string true "Name of challenge"
// @Success 200 {object} api.Challenge
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/info/challenge/{name} [get]
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

	username, err := resolveRequestUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "No Token Provided",
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

	if len(challenges) > 0 && (challenges[0].Status == core.DEPLOY_STATUS["deployed"] || user.Role != core.USER_ROLES["contestant"]) {
		challenge := challenges[0]
		totalSolves, solveStatus, err := database.GetChallengeSolveInfo(challenge.ID, user.ID)
		if err != nil {
			log.Error(err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		challengeTags := extractTagNames(challenge.Tags)

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
			Name:               challenge.Name,
			ChallId:            challenge.ID,
			Tags:               challengeTags,
			CreatedAt:          challenge.CreatedAt,
			Points:             challenge.Points,
			SolvesNumber:       totalSolves,
			SolveStatus:        solveStatus,
			Difficulty:         challenge.Difficulty,
			PreRequisite:       fileUtils.SplitDelimited(challenge.PreReqs, core.DELIMITER),
			DeployedStatus:     challenge.Status,
			Instanced:          challenge.Instanced,
			InstanceExpiration: challenge.InstanceExpiration,
		}

		serverDeployed := challenge.ServerDeployed
		if serverDeployed == "" && challenge.Type != core.STATIC_CHALLENGE_TYPE_NAME {
			serverDeployed = core.LOCALHOST
		}

		var ports []uint32
		if challenge.Instanced {
			if instance, err := manager.GetUserInstance(fmt.Sprintf("%d", user.ID), challenge.Name); err == nil && instance != nil {
				ports = instance.Port
			}
		} else {
			if len(challenge.Ports) > 0 {
				ports = make([]uint32, len(challenge.Ports))
				for i, p := range challenge.Ports {
					ports[i] = p.PortNo
				}
			} else if allocated, err := database.GetAllocatedPorts(challenge); err == nil && len(allocated) > 0 {
				ports = make([]uint32, len(allocated))
				for i, p := range allocated {
					ports[i] = p.PortNo
				}
			} else if cache.Cache != nil && serverDeployed != "" {
				portOwner := challenge.ContainerId
				if challenge.DeploymentType == core.DEPLOYMENT_TYPES["docker_compose"] {
					portOwner = fileUtils.ProjectNameNotInstanced(challenge.Name)
				}
				if portOwner != "" {
					if redisPorts, err := cache.GetContainerPortsOnHost(serverDeployed, portOwner); err == nil && len(redisPorts) > 0 {
						ports = redisPorts
					}
				}
			}
		}
		if ports == nil {
			ports = []uint32{}
		}

		challengeInfo := Challenge{
			ChallengeMetadata: challMetadata,
			Description:       challenge.Description,
			Hints:             hintInfos,
			Category:          challenge.Type,
			Assets:            fileUtils.SplitDelimited(challenge.Assets, core.DELIMITER),
			AdditionalLinks:   fileUtils.SplitDelimited(challenge.AdditionalLinks, core.DELIMITER),
			PreviousTries:     previousTries,
			MaxAttemptLimit:   challenge.MaxAttemptLimit,
			DeployedLink:      serverDeployed,
			Ports:             ports,
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
// @Produce json
// @Param filter query string false "Filter parameter by which challenges are filtered"
// @Param value query string false "Value of filtered parameter"
// @Success 200 {array} api.ChallengeMetadata
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/info/challenges [get]
func challengesMetadataHandler(c *gin.Context) {
	filter := strings.TrimSpace(c.Query("filter"))
	value := strings.TrimSpace(c.Query("value"))

	var challenges []database.Challenge
	var err error

	err, state := utils.CheckTime()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	var autherr error
	if !config.SkipAuthorization {
		authHeader := c.GetHeader("Authorization")
		values := strings.Split(authHeader, " ")
		if len(values) < 2 || values[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, HTTPPlainResp{
				Message: "No Token Provided",
			})
			c.Abort()
			return
		}
		autherr = auth.Authorize(values[1], core.ADMIN)
	}

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
		if filter == "name" && value != "" {
			challenges, err = database.QueryChallengeEntriesMetadata("name", value)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
		} else if filter == "score" && value != "" {
			if _, parseErr := strconv.ParseUint(value, 10, 64); parseErr != nil {
				c.JSON(http.StatusBadRequest, HTTPErrorResp{
					Error: "Invalid score filter value",
				})
				return
			}
			challenges, err = database.QueryChallengeEntriesMetadata("points", value)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
		} else if filter == "author" && value != "" {
			authorUser, lookupErr := database.QueryFirstUserEntry("username", value)
			if lookupErr != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
			if authorUser.ID != 0 {
				challenges, err = database.QueryChallengeEntriesMetadata("author_id", strconv.FormatUint(uint64(authorUser.ID), 10))
			} else if _, parseErr := strconv.ParseUint(value, 10, 64); parseErr == nil {
				challenges, err = database.QueryChallengeEntriesMetadata("author_id", value)
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
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
				return
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

		availableChallenges := make([]ChallengeMetadata, 0, len(challenges))

		username, err := resolveRequestUsername(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "No Token Provided",
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

		for _, challenge := range challenges {
			if challenge.Status != core.DEPLOY_STATUS["deployed"] && user.Role == core.USER_ROLES["contestant"] {
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
			challengeTags := extractTagNames(challenge.Tags)

			availableChallenges = append(availableChallenges, ChallengeMetadata{
				Name:               challenge.Name,
				ChallId:            challenge.ID,
				Tags:               challengeTags,
				CreatedAt:          challenge.CreatedAt,
				Points:             challenge.Points,
				SolvesNumber:       totalSolves,
				SolveStatus:        solveStatus,
				Difficulty:         challenge.Difficulty,
				PreRequisite:       fileUtils.SplitDelimited(challenge.PreReqs, core.DELIMITER),
				DeployedStatus:     challenge.Status,
				Instanced:          challenge.Instanced,
				InstanceExpiration: challenge.InstanceExpiration,
			})
		}

		c.JSON(http.StatusOK, availableChallenges)
		return
	}
}

// Returns available base images.
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
// @Produce json
// @Param challenge query string true "The name of the challenge to get the logs for."
// @Success 200 {object} api.LogsInfoResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Security ApiKeyAuth
// @Router /api/manage/logs [get]
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

func canViewUserEmails(c *gin.Context, targetUsername string) bool {
	if config.SkipAuthorization {
		return true
	}
	callerUsername, err := resolveRequestUsername(c)
	if err != nil {
		return false
	}
	if targetUsername != "" && callerUsername == targetUsername {
		return true
	}
	caller, err := database.QueryFirstUserEntry("username", callerUsername)
	if err != nil || caller.ID == 0 {
		return false
	}
	return caller.Role != core.USER_ROLES["contestant"]
}

// Returns user info
// @Summary Returns user info
// @Description Returns user info based on userId
// @Tags info
// @Produce json
// @Success 200 {object} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Param username path string true "Username"
// @Accept x-www-form-urlencoded,mpfd
// @Security ApiKeyAuth
// @Router /api/info/user/{username} [get]
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
	if userId != "" {
		id, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPErrorResp{
				Error: fmt.Sprintf("Could not parse User Id or invalid User Id"),
			})
			return
		}

		user, err = database.QueryUserById(uint(id))
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
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "User not found",
		})
		return
	}

	solvedChallenges, err := database.GetUserSolvedChallenges(user.ID)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}

	userChallenges := make([]ChallengeSolveResp, len(solvedChallenges))
	for index, sc := range solvedChallenges {

		challengeTags := extractTagNames(sc.Tags)

		challResp := ChallengeSolveResp{
			Id:       sc.ChallengeID,
			Name:     sc.Name,
			Tags:     challengeTags,
			Category: sc.Type,
			SolvedAt: sc.SolvedAt,
			Points:   sc.Points,
		}
		userChallenges[index] = challResp
	}

	var rank int64
	if user.Status == 0 {
		rank, err = database.GetUserRank(user.ID, user.Score, user.UpdatedAt)
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

	email := ""
	if canViewUserEmails(c, user.Username) {
		email = user.Email
	}

	resp := UserResp{
		Username:   user.Username,
		Id:         user.ID,
		Role:       user.Role,
		Status:     user.Status,
		Score:      user.Score,
		Rank:       rank,
		Email:      email,
		Challenges: userChallenges,
	}
	c.JSON(http.StatusOK, resp)
}

// Returns all user's info
// @Summary Returns all user's info
// @Description Returns all available user's info
// @Tags info
// @Produce json,text/csv
// @Param sort query string false "Sort field for administrators"
// @Param order query string false "Sort order for administrators"
// @Param filter query string false "Status filter for administrators"
// @Param format query string false "Use csv to export as an administrator"
// @Success 200 {array} api.UsersResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/info/users [get]
func getAllUsersInfoHandler(c *gin.Context) {
	users, err := database.QueryUserEntries("role", "contestant")
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "DATABASE ERROR while processing the request.",
		})
		return
	}
	showEmails := canViewUserEmails(c, "")
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

			email := ""
			if showEmails {
				email = user.Email
			}

			availableUsers[index] = UsersResp{
				Username: user.Username,
				Id:       user.ID,
				Role:     user.Role,
				Status:   user.Status,
				Score:    user.Score,
				Email:    email,
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

		filteredUsers := make([]UsersResp, 0)

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
}

// Handles submissions made by the user
// @Summary Handles submissions made by the user
// @Description Handles submissions made by the user
// @Tags info
// @Produce json,text/csv
// @Param page query int false "Page number"
// @Param format query string false "Use csv to export"
// @Success 200 {array} api.SubmissionResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
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
}

// Returns statistics of users in competition
// @Summary statistics of users in competition
// @Description returns statistics of users in competition (currently limited to ban/unban status of users)
// @Tags info
// @Produce json
// @Success 200 {object} api.UsersStatisticsResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
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
		return
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
}

// Returns competition information
// @Summary returns competition info
// @Description returns various information about the competition which are used to control competition
// @Tags info
// @Produce json
// @Success 200 {object} api.CompetitionInfoResp
// @Failure 400 {object} api.HTTPErrorResp
// @Router /api/info/competition-info [get]
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
}

// Returns allTags
// @Summary returns all tags
// @Description returns all unique tags
// @Tags info
// @Produce json
// @Success 200 {object} api.TagInfoResp
// @Failure 400 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/info/tags [get]
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
		Tags: fileUtils.CleanStringSlice(tags),
	})
}

func isSensitiveAssetPath(cleanAsset string) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(cleanAsset))
	base := filepath.Base(lowerPath)
	return strings.HasSuffix(base, ".toml") ||
		strings.HasSuffix(base, ".tar.gz") ||
		base == ".env" ||
		base == "check.sh" ||
		base == "dockerfile" ||
		base == "docker-compose.yml" ||
		base == "docker-compose.yaml" ||
		base == "xinetd.conf" ||
		strings.HasPrefix(base, "dockerfile.") ||
		strings.HasPrefix(base, ".env.")
}

// @Summary Download a published challenge asset
// @Tags info
// @Produce application/octet-stream
// @Param challenge query string true "Challenge name"
// @Param asset query string true "Relative asset path"
// @Success 200 {file} file
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 403 {object} api.HTTPErrorResp
// @Router /api/info/download [get]
func serveAssets(c *gin.Context) {
	challenge := filepath.Base(strings.TrimSpace(c.Query("challenge")))
	rawAsset := strings.TrimSpace(c.Query("asset"))
	if challenge == "" || challenge == "." || challenge == ".." || rawAsset == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "challenge and asset parameters are required"})
		return
	}

	cleanAsset := filepath.Clean(filepath.FromSlash(rawAsset))
	if filepath.IsAbs(cleanAsset) || cleanAsset == "." || cleanAsset == ".." || strings.HasPrefix(cleanAsset, ".."+string(filepath.Separator)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset path"})
		return
	}

	if isSensitiveAssetPath(cleanAsset) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	assetRoots := []string{
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challenge, core.BEAST_STATIC_FOLDER),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challenge, core.PUBLIC),
	}

	var targetPath string
	for _, rootDir := range assetRoots {
		candidate, err := fileUtils.ResolvePathWithin(rootDir, cleanAsset)
		if err != nil {
			continue
		}
		if isSensitiveAssetPath(candidate) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		if fileUtils.ValidateFileExists(candidate) == nil {
			targetPath = candidate
			break
		}
	}
	if targetPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect file requested"})
		return
	}
	c.FileAttachment(targetPath, filepath.Base(cleanAsset))
}

// This route returns the number of users in the databse with role=contestant
// @Summary Returns the number of users in the database with role=contestant
// @Description Returns the number of users in the database with role=contestant
// @Tags info
// @Produce json
// @Success 200 {object} api.UserCountResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
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
// @Produce json
// @Param page query int false "Page number"
// @Success 200 {array} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
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
			leaderboardCacheMu.RLock()
			if !leaderboardStale {
				cached := make([]UserResp, len(leaderboardCache))
				copy(cached, leaderboardCache)
				leaderboardCacheMu.RUnlock()
				c.JSON(http.StatusOK, cached)
				return
			}
			leaderboardCacheMu.RUnlock()

			users, err := database.QueryTopUsersByFrozenScore(core.LEADERBOARD_SIZE)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}
			leaderboard := make([]UserResp, 0, len(users))
			for _, user := range users {
				if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
					continue
				}
				resp := UserResp{
					Username: user.Username,
					Id:       user.ID,
					Role:     user.Role,
					Status:   user.Status,
					Score:    user.FrozenScore,
					Rank:     int64(len(leaderboard) + 1),
				}
				leaderboard = append(leaderboard, resp)
			}

			leaderboardCacheMu.Lock()
			leaderboardCache = leaderboard
			leaderboardStale = false
			cached := make([]UserResp, len(leaderboard))
			copy(cached, leaderboard)
			leaderboardCacheMu.Unlock()

			c.JSON(http.StatusOK, cached)
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
		leaderboard := make([]UserResp, 0, len(users))
		for _, user := range users {
			if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
				continue
			}
			resp := UserResp{
				Username: user.Username,
				Id:       user.ID,
				Role:     user.Role,
				Status:   user.Status,
				Score:    user.FrozenScore,
				Rank:     int64(offset + len(leaderboard) + 1),
			}
			leaderboard = append(leaderboard, resp)
		}
		c.JSON(http.StatusOK, leaderboard)
		return
	}

	if page == 1 {
		leaderboardCacheMu.RLock()
		if !leaderboardStale {
			cached := make([]UserResp, len(leaderboardCache))
			copy(cached, leaderboardCache)
			leaderboardCacheMu.RUnlock()
			c.JSON(http.StatusOK, cached)
			return
		}
		leaderboardCacheMu.RUnlock()

		users, err := database.QueryTopUsersByScore(core.LEADERBOARD_SIZE)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		leaderboard := make([]UserResp, 0, len(users))
		for _, user := range users {
			if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
				continue
			}
			resp := UserResp{
				Username: user.Username,
				Id:       user.ID,
				Role:     user.Role,
				Status:   user.Status,
				Score:    user.Score,
				Rank:     int64(len(leaderboard) + 1),
			}
			leaderboard = append(leaderboard, resp)
		}

		leaderboardCacheMu.Lock()
		leaderboardCache = leaderboard
		leaderboardStale = false
		cached := make([]UserResp, len(leaderboard))
		copy(cached, leaderboard)
		leaderboardCacheMu.Unlock()

		c.JSON(http.StatusOK, cached)
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
	leaderboard := make([]UserResp, 0, len(users))
	for _, user := range users {
		if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
			continue
		}
		resp := UserResp{
			Username: user.Username,
			Id:       user.ID,
			Role:     user.Role,
			Status:   user.Status,
			Score:    user.Score,
			Rank:     int64(offset + len(leaderboard) + 1),
		}
		leaderboard = append(leaderboard, resp)
	}
	c.JSON(http.StatusOK, leaderboard)
}

// Returns admin leaderboard
// @Summary Returns admin leaderboard
// @Description Returns admin leaderboard of all users
// @Tags info
// @Produce json
// @Param page query int false "Page number"
// @Success 200 {array} api.UserResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
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
		leaderboardCacheMu.RLock()
		if !adminLeaderboardStale {
			cached := make([]UserResp, len(adminLeaderboardCache))
			copy(cached, adminLeaderboardCache)
			leaderboardCacheMu.RUnlock()
			c.JSON(http.StatusOK, cached)
			return
		}
		leaderboardCacheMu.RUnlock()

		users, err := database.QueryTopUsersByScore(core.LEADERBOARD_SIZE)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}
		leaderboard := make([]UserResp, 0, len(users))
		for _, user := range users {
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
				Rank:     int64(len(leaderboard) + 1),
			}
			leaderboard = append(leaderboard, resp)
		}

		leaderboardCacheMu.Lock()
		adminLeaderboardCache = leaderboard
		adminLeaderboardStale = false
		cached := make([]UserResp, len(leaderboard))
		copy(cached, leaderboard)
		leaderboardCacheMu.Unlock()

		c.JSON(http.StatusOK, cached)
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
	leaderboard := make([]UserResp, 0, len(users))
	for _, user := range users {
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
			Rank:     int64(offset + len(leaderboard) + 1),
		}
		leaderboard = append(leaderboard, resp)
	}
	c.JSON(http.StatusOK, leaderboard)
}

// Freeze user leaderboard
// @Summary Freeze user leaderboard
// @Description freezes the user leaderboard on demand.
// @Tags info
// @Produce json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/admin/freezeLeaderboard [post]
func freezeLeaderboardHandler(c *gin.Context) {
	err := database.UpdateFrozenScores()
	if err != nil {
		log.Errorf("DATABASE ERROR while freezing the leaderboard. %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	invalidatePublicLeaderboardCaches()
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User leaderboard frozen successfully",
	})
}

// Unfreeze user leaderboard
// @Summary Unfreeze user leaderboard
// @Description unfreezes the user leaderboard on demand.
// @Tags info
// @Produce json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/admin/unfreezeLeaderboard [post]
func unfreezeLeaderboardHandler(c *gin.Context) {
	err := database.ResetFrozenScores()
	if err != nil {
		log.Errorf("DATABASE ERROR while un-freezing the leaderboard. %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	invalidatePublicLeaderboardCaches()
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User leaderboard unfrozen successfully",
	})
}

// Fetch all the attempts on this challenge
// @Summary Get challenge attempts
// @Description Returns all user attempts for a given challenge.
// @Tags info
// @Produce json
// @Param challenge_id path int true "Challenge ID"
// @Success 200 {array} api.SubmissionResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/info/submissions/challenge/{challenge_id} [get]
func getChallengeAttempts(c *gin.Context) {
	username, err := resolveRequestUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
		})
		return
	}

	queryingUser, err := database.QueryFirstUserEntry("username", username)
	if err != nil || queryingUser.ID == 0 {
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
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {array} api.SubmissionResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 404 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Security ApiKeyAuth
// @Router /api/info/submissions/user/{user_id} [get]
func getUserAttempts(c *gin.Context) {
	username, err := resolveRequestUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{
			Error: "Unauthorized user",
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
	if err != nil || submissionUser.ID == 0 {
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

// @Summary Get leaderboard score history
// @Tags info
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} database.UserLeaderboardResp
// @Router /api/info/leaderboard-graph [get]
func getLeaderboardGraphHandler(c *gin.Context) {
	isLeaderboardFrozen, _ := database.IsFrozenScoreSet()

	leaderboardCacheMu.RLock()
	if !graphCacheStale {
		cached := make([]database.UserLeaderboardResp, len(graphCache))
		copy(cached, graphCache)
		leaderboardCacheMu.RUnlock()
		c.JSON(http.StatusOK, cached)
		return
	}
	lbStale := leaderboardStale
	var topUsers []uint
	if !lbStale {
		for i := 0; i < 10 && i < len(leaderboardCache); i++ {
			topUsers = append(topUsers, leaderboardCache[i].Id)
		}
	}
	leaderboardCacheMu.RUnlock()

	if lbStale {
		var (
			users []database.User
			err   error
		)
		if isLeaderboardFrozen {
			users, err = database.QueryTopUsersByFrozenScore(core.LEADERBOARD_SIZE)
		} else {
			users, err = database.QueryTopUsersByScore(core.LEADERBOARD_SIZE)
		}
		if err == nil {
			for _, user := range users {
				if user.Status != 0 || user.Role != core.USER_ROLES["contestant"] {
					continue
				}
				topUsers = append(topUsers, user.ID)
				if len(topUsers) >= 10 {
					break
				}
			}
		}
	}

	graphFresh := database.QueryTimeSeriesForTopUsers(topUsers)
	if graphFresh == nil {
		graphFresh = make([]database.UserLeaderboardResp, 0)
	}

	leaderboardCacheMu.Lock()
	graphCache = graphFresh
	graphCacheStale = false
	cached := make([]database.UserLeaderboardResp, len(graphCache))
	copy(cached, graphCache)
	leaderboardCacheMu.Unlock()

	c.JSON(http.StatusOK, cached)
}
