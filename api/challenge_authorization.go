package api

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	challengeConfig "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
)

func authenticatedManager(c *gin.Context) (database.User, bool) {
	claimsValue, exists := c.Get("authClaims")
	claims, valid := claimsValue.(*auth.CustomClaims)
	if !exists || !valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, HTTPErrorResp{Error: "verified authentication claims are required"})
		return database.User{}, false
	}
	user, err := database.QueryFirstUserEntry("username", claims.User)
	if err != nil || user.ID == 0 || user.Status != 0 {
		c.AbortWithStatusJSON(http.StatusForbidden, HTTPErrorResp{Error: "challenge management access denied"})
		return database.User{}, false
	}
	return user, true
}

func userOwnsChallengeConfig(user database.User, configuration challengeConfig.BeastChallengeConfig) bool {
	if user.Role == core.USER_ROLES["admin"] {
		return true
	}
	if strings.EqualFold(user.Email, configuration.Author.Email) {
		return true
	}
	for _, maintainer := range configuration.Maintainers {
		if strings.EqualFold(user.Email, maintainer.Email) {
			return true
		}
	}
	return false
}

func authorizeChallengeManagement(c *gin.Context, challengeName string, allowRemoteConfig bool) bool {
	user, ok := authenticatedManager(c)
	if !ok {
		return false
	}
	challenge, err := database.QueryFirstChallengeEntry("name", challengeName)
	if err == nil {
		maintainer, relationErr := database.IsChallengeMaintainer(user.ID, challenge.ID)
		if relationErr != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, HTTPErrorResp{Error: "failed to authorize challenge access"})
			return false
		}
		if !userCanExecChallenge(user, challenge, maintainer) {
			c.AbortWithStatusJSON(http.StatusForbidden, HTTPErrorResp{Error: "challenge management access denied"})
			return false
		}
		return true
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, HTTPErrorResp{Error: "failed to query challenge"})
		return false
	}
	if !allowRemoteConfig {
		c.AbortWithStatusJSON(http.StatusNotFound, HTTPErrorResp{Error: "challenge not found"})
		return false
	}

	challengeDir := coreUtils.GetChallengeDir(challengeName)
	if challengeDir == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, HTTPErrorResp{Error: "challenge not found"})
		return false
	}
	configuration, loadErr := challengeConfig.LoadChallengeConfig(filepath.Join(challengeDir, core.CHALLENGE_CONFIG_FILE_NAME))
	if loadErr != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, HTTPErrorResp{Error: "challenge configuration is invalid"})
		return false
	}
	if !userOwnsChallengeConfig(user, configuration) {
		c.AbortWithStatusJSON(http.StatusForbidden, HTTPErrorResp{Error: "challenge management access denied"})
		return false
	}
	return true
}
