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
	"github.com/sdslabs/beastv4/pkg/auth"
	fileUtils "github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

// Returns port in use by beast.
// @Summary Returns ports in use by beast by looking in the hack git repository, also returns min and max value of port allowed while specifying in beast challenge config.
// @Description Returns the ports in use by beast, which cannot be used in creating a new challenge..
// @Tags info
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.PortsInUseResp
// @Router /api/info/ports/used [get]
func usedPortsInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, PortsInUseResp{
		MinPortValue: core.ALLOWED_MIN_PORT_VALUE,
		MaxPortValue: core.ALLOWED_MAX_PORT_VALUE,
		PortsInUse:   cfg.USED_PORTS_LIST,
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
			Error: fmt.Sprintf("Challenge name cannot be empty"),
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

	if len(challenges) > 0 {
		challenge := challenges[0]

		users, err := database.GetRelatedUsers(&challenge)
		if err != nil {
			log.Error(err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "DATABASE ERROR while processing the request.",
			})
			return
		}

		challengePorts := make([]uint32, len(challenge.Ports))
		for index, port := range challenge.Ports {
			challengePorts[index] = port.PortNo
		}

		var challSolves int
		challengeUser := make([]UserSolveResp, 0)
		for _, user := range users {
			if user.Role == core.USER_ROLES["contestant"] {
				userResp := UserSolveResp{
					UserID:   user.ID,
					Username: user.Username,
					SolvedAt: user.CreatedAt,
				}
				challengeUser = append(challengeUser, userResp)
				challSolves++
			}
		}

		challengeTags := make([]string, len(challenge.Tags))

		for index, tags := range challenge.Tags {
			challengeTags[index] = tags.TagName
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

		if autherr != nil {
			c.JSON(http.StatusOK, ChallengeInfoResp{
				Name:            name,
				ChallId:         challenge.ID,
				Category:        challenge.Type,
				CreatedAt:       challenge.CreatedAt,
				Tags:            challengeTags,
				Status:          challenge.Status,
				Ports:           challengePorts,
				Hints:           challenge.Hints,
				Desc:            challenge.Description,
				Assets:          strings.Split(challenge.Assets, core.DELIMITER),
				AdditionalLinks: strings.Split(challenge.AdditionalLinks, core.DELIMITER),
				Points:          challenge.Points,
				SolvesNumber:    challSolves,
				Solves:          challengeUser,
			})
			return
		}

		c.JSON(http.StatusOK, ChallengeInfoResp{
			Name:            name,
			ChallId:         challenge.ID,
			Category:        challenge.Type,
			Flag:            challenge.Flag,
			CreatedAt:       challenge.CreatedAt,
			Tags:            challengeTags,
			Status:          challenge.Status,
			Ports:           challengePorts,
			Hints:           challenge.Hints,
			Desc:            challenge.Description,
			Assets:          strings.Split(challenge.Assets, core.DELIMITER),
			AdditionalLinks: strings.Split(challenge.AdditionalLinks, core.DELIMITER),
			Points:          challenge.Points,
			SolvesNumber:    challSolves,
			Solves:          challengeUser,
		})
	} else {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No challenge found with name: " + name,
		})
	}

	return
}

// Returns information about all challenges with and without filters
// @Summary Returns information about all challenges with and without filters.
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
func challengesInfoHandler(c *gin.Context) {
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
		if value == "" || filter == "" {
			challenges, err = database.QueryAllChallenges()
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

		if filter == "name" || filter == "author" || filter == "score" {
			challenges, err = database.QueryChallengeEntries(filter, value)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
			}
		}

		if filter == "tag" {
			tag := database.Tag{
				TagName: value,
			}
			challenges, err = database.QueryRelatedChallenges(&tag)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
			}
		}

		availableChallenges := make([]ChallengeInfoResp, len(challenges))

		for index, challenge := range challenges {
			users, err := database.GetRelatedUsers(&challenge)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while processing the request.",
				})
				return
			}

			challengePorts := make([]uint32, len(challenge.Ports))
			for index, port := range challenge.Ports {
				challengePorts[index] = port.PortNo
			}

			var challSolves int
			challengeUser := make([]UserSolveResp, 0)

			for _, user := range users {
				if user.Role == core.USER_ROLES["contestant"] {
					userResp := UserSolveResp{
						UserID:   user.ID,
						Username: user.Username,
						SolvedAt: user.CreatedAt,
					}
					challengeUser = append(challengeUser, userResp)
					challSolves++
				}
			}

			challengeTags := make([]string, len(challenge.Tags))

			for index, tags := range challenge.Tags {
				challengeTags[index] = tags.TagName
			}

			availableChallenges[index] = ChallengeInfoResp{
				Name:            challenge.Name,
				ChallId:         challenge.ID,
				Category:        challenge.Type,
				Tags:            challengeTags,
				CreatedAt:       challenge.CreatedAt,
				Status:          challenge.Status,
				Ports:           challengePorts,
				Hints:           challenge.Hints,
				Desc:            challenge.Description,
				Points:          challenge.Points,
				Assets:          strings.Split(challenge.Assets, core.DELIMITER),
				AdditionalLinks: strings.Split(challenge.AdditionalLinks, core.DELIMITER),
				SolvesNumber:    challSolves,
				Solves:          challengeUser,
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
			Message: fmt.Sprintf("Challenge name cannot be empty"),
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
// @Router /api/info/submissions [get]
func submissionsHandler(c *gin.Context) {

	submissions, err := database.QueryAllSubmissions()
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

		if user.Role == core.USER_ROLES["contestant"] {
			challenge, err := database.QueryChallengeEntries("id", strconv.Itoa(int(submission.ChallengeID)))
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "DATABASE ERROR while fetching user details.",
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
				Category:  challenge[0].Type,
				Tags:      challengeTags,
				Points:    challenge[0].Points,
				SolvedAt:  submission.CreatedAt,
			}
			submissionsResp = append(submissionsResp, singleSubmissionResp)
		}
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
	challenges, _ := database.QueryAllChallenges()
	var tags []string
	for _, challenge := range challenges {
		for _, tag := range challenge.Tags {
			tags = append(tags, tag.TagName)
		}
	}
	keys := make(map[string]bool)
	uniqueTags := []string{}
	for _, entry := range tags {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			uniqueTags = append(uniqueTags, entry)
		}
	}
	c.JSON(http.StatusOK, TagInfoResp{
		Tags: uniqueTags,
	})
	return
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
	log.Print("Recieved request")
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

	return
}
