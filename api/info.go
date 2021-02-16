package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
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
// @Success 200 {object} api.ChallengeInfoResp
// @Router /api/info/challenge/info [post]
func challengeInfoHandler(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Challenge name cannot be empty"),
		})
		return
	}

	challenges, err := database.QueryChallengeEntries("name", name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	if len(challenges) > 0 {
		challenge := challenges[0]

		users, err := database.GetRelatedUsers(&challenge)
		if err != nil {
			log.Error(err)
			c.JSON(http.StatusInternalServerError, HTTPPlainResp{
				Message: "DATABASE ERROR while processing the request.",
			})
			return
		}

		var challSolves int
		var challengeUser []UserSolveResp

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

		c.JSON(http.StatusOK, ChallengeInfoResp{
			Name:         name,
			ChallId:      challenge.ID,
			Category:     challenge.Type,
			CreatedAt:    challenge.CreatedAt,
			Status:       challenge.Status,
			Ports:        challenge.Ports,
			Hints:        challenge.Hints,
			Desc:         challenge.Description,
			Points:       challenge.Points,
			SolvesNumber: challSolves,
			Solves:       challengeUser,
		})
	} else {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No challenge found with name: " + name,
		})
	}

	return
}

// Returns information about all challenges
// @Summary Returns information about all challenges.
// @Description Returns information about all the challenges present in the database.
// @Tags info
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.ChallengeInfoResp
// @Router /api/info/available [post]
func availableChallengeInfoHandler(c *gin.Context) {
	challenges, err := database.QueryAllChallenges()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	availableChallenges := make([]ChallengeInfoResp, len(challenges))
	if len(challenges) > 0 {
		for index, challenge := range challenges {

			users, err := database.GetRelatedUsers(&challenge)
			if err != nil {
				log.Error(err)
				c.JSON(http.StatusInternalServerError, HTTPPlainResp{
					Message: "DATABASE ERROR while processing the request.",
				})
				return
			}

			var challSolves int
			var challengeUser []UserSolveResp

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

			availableChallenges[index] = ChallengeInfoResp{
				Name:         challenge.Name,
				ChallId:      challenge.ID,
				Category:     challenge.Type,
				CreatedAt:    challenge.CreatedAt,
				Status:       challenge.Status,
				Ports:        challenge.Ports,
				Hints:        challenge.Hints,
				Desc:         challenge.Description,
				Points:       challenge.Points,
				SolvesNumber: challSolves,
				Solves:       challengeUser,
			}
		}

	}
	c.JSON(http.StatusOK, availableChallenges)
	return
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

// Returns available challenges from the database by filter
// @Summary Gives all challenges available in the database that has a particular parameter same
// @Description Returns all challenges available in the in the database that has a particular parameter same
// @Tags info
// @Accept json
// @Produce json
// @Success 200 {object} api.ChallengesResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/info/challenges [get]
func challengesInfoHandler(c *gin.Context) {
	filter := c.Query("filter")
	value := c.Query("value")

	if value == "" || filter == "" {
		challenges, err := database.QueryAllChallenges()
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: err.Error(),
			})
			return
		} else if challenges == nil {
			c.JSON(http.StatusOK, HTTPPlainResp{
				Message: "No challenges currently in the database",
			})
			return
		} else {
			var challNameString []string
			for _, challenge := range challenges {
				challNameString = append(challNameString, challenge.Name)
			}
			c.JSON(http.StatusOK, ChallengesResp{
				Message:    "All Challenges",
				Challenges: challNameString,
			})
			return
		}
	}

	var challenges []database.Challenge
	var err error
	if filter == "name" || filter == "author" || filter == "score" {
		challenges, err = database.QueryChallengeEntries(filter, value)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPPlainResp{
				Message: "DATABASE ERROR while processing the request.",
			})
		} else {
			var challNameString []string
			for _, challenge := range challenges {
				challNameString = append(challNameString, challenge.Name)
			}
			c.JSON(http.StatusOK, ChallengesResp{
				Message:    "Challenges with " + filter + " = " + value,
				Challenges: challNameString,
			})
		}
	}

	if filter == "tag" {
		tag := database.Tag{
			TagName: value,
		}
		challenges, err = database.QueryRelatedChallenges(&tag)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPPlainResp{
				Message: "DATABASE ERROR while processing the request.",
			})
		} else {
			var challNameString []string
			for _, challenge := range challenges {
				challNameString = append(challNameString, challenge.Name)
			}
			c.JSON(http.StatusOK, ChallengesResp{
				Message:    "Challenges with " + filter + " = " + value,
				Challenges: challNameString,
			})
		}
	}
}

// Returns available challenges from the remote directory
// @Summary Gives all challenges available in the remote directory
// @Description Returns all challenges available in the in the remote directory
// @Tags info
// @Accept json
// @Produce json
// @Success 200 {object} api.ChallengesResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/info/challenges/available [get]
func availableChallengeHandler(c *gin.Context) {
	challenges, err := manager.GetAvailableChallenges()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	} else if challenges == nil {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "No challenges currently in the database",
		})
		return
	} else {
		c.JSON(http.StatusOK, ChallengesResp{
			Message:    "All Challenges",
			Challenges: challenges,
		})
	}
}

// Returns user info
// @Summary Returns user info
// @Description Returns user info based on userId
// @Tags info
// @Accept json
// @Produce json
// @Success 200 {object} api.UserResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/info/user [post]
func userInfoHandler(c *gin.Context) {
	userId := c.PostForm("user_id")
	username := c.PostForm("username")
	if userId == "" && username == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Both User Id and Username cannot be empty"),
		})
		return
	}
	var user database.User
	var err error
	var parsedUserId uint
	if userId != "" {
		id, err := strconv.ParseUint(userId, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Could not parse User Id or invalid User Id"),
			})
			return
		}
		parsedUserId = uint(id)

		user, err = database.QueryUserById(parsedUserId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPPlainResp{
				Message: "DATABASE ERROR while processing the request.",
			})
			return
		}
	} else {
		user, err = database.QueryFirstUserEntry("username", username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPPlainResp{
				Message: "DATABASE ERROR while processing the request.",
			})
			return
		}
	}

	challenges, err := database.GetRelatedChallenges(&user)
	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	var resp UserResp

	var challNameString []string
	for _, challenge := range challenges {
		challNameString = append(challNameString, challenge.Name)
	}
	var userChallenges []ChallengeSolveResp
	for _, challenge := range challenges {
		challResp := ChallengeSolveResp{
			Id:       challenge.ID,
			Name:     challenge.Name,
			Category: challenge.Type,
			SolvedAt: challenge.CreatedAt,
			Points:   challenge.Points,
		}
		userChallenges = append(userChallenges, challResp)
	}

	rank, err := database.GetUserRank(parsedUserId, user.Score, user.UpdatedAt)

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
// @Success 200 {object} api.UserResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/info/user/available [get]
func getAllUsersInfoHandler(c *gin.Context) {
	users, err := database.QueryAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}

	if len(users) > 0 {
		availableUsers := make([]UsersResp, len(users))
		for index, user := range users {
			availableUsers[index] = UsersResp{
				Username: user.Username,
				Id:       user.ID,
				Role:     user.Role,
				Status:   user.Status,
				Score:    user.Score,
				Email:    user.Email,
			}
		}
		c.JSON(http.StatusOK, availableUsers)
	} else {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "No users found in the database",
		})
	}

	return
}

func submissionsHandler(c *gin.Context) {
	submissions, err := database.QueryAllSubmissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
		})
		return
	}
	var submissionsResp []SubmissionResp

	for _, submission := range submissions {
		user, err := database.QueryUserById(submission.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPPlainResp{
				Message: "DATABASE ERROR while fetching user details.",
			})
			return
		}

		if user.Role == core.USER_ROLES["contestant"] {
			challenge, err := database.QueryChallengeEntries("id", strconv.Itoa(int(submission.ChallengeID)))
			if err != nil {
				c.JSON(http.StatusInternalServerError, HTTPPlainResp{
					Message: "DATABASE ERROR while fetching user details.",
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
				Category:  challenge[0].Type,
				Points:    challenge[0].Points,
				SolvedAt:  submission.CreatedAt,
			}
			submissionsResp = append(submissionsResp, singleSubmissionResp)
		}
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
// @Success 200 {object} api.UserResp
// @Failure 404 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/admin/statistics [post]
func getUsersStatisticsHandler(c *gin.Context) {
	users, err := database.QueryAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "DATABASE ERROR while processing the request.",
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
// @Success 200 {object} api.CompetitionInfoResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/admin/statistics [get]
func competitionInfoHandler(c *gin.Context) {
	competitionInfo, err := config.GetCompetitionInfo()
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, CompetitionInfoResp{
		Name:         competitionInfo.Name,
		About:        competitionInfo.About,
		Prizes:       competitionInfo.Prizes,
		StartingTime: competitionInfo.StartingTime,
		EndingTime:   competitionInfo.EndingTime,
		TimeZone:     competitionInfo.TimeZone,
		LogoURL:      "/api/info/logo",
	})
	return
}
