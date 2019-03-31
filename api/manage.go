package api

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/auth"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/database"
	log "github.com/sirupsen/logrus"
)

// Handles route related to manage all the challenges for current beast remote.
// @Summary Handles challenge management actions for multiple(all) challenges.
// @Description Handles challenge management routes for all the challenges with actions which includes - DEPLOY, UNDEPLOY.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param action query string true "Action for the challenge"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/manage/all/:action [post]
func decryptToken(authHeader string) string {
	values := strings.Split(authHeader, " ")
	userInfoEncr := strings.Split(values[1], ".")
	sDec, err1 := b64.StdEncoding.DecodeString(userInfoEncr[1])
	if err1 != nil {
		fmt.Printf("Error in decrypting JWT token")
	}
	in := []byte(sDec)
	var raw auth.CustomClaims
	json.Unmarshal(in, &raw)
	return raw.User
}

func manageMultipleChallengeHandler(c *gin.Context) {
	action := c.Param("action")

	switch action {
	case core.MANAGE_ACTION_DEPLOY:
		log.Infof("Starting deploy for all challenges")
		msgs := manager.DeployAll(true, decryptToken(c.GetHeader("Authorization")))

		var msg string
		if len(msgs) != 0 {
			msg = strings.Join(msgs, " ::: ")
		} else {
			msg = "Deploy for all challeges started"
		}

		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": msg,
		})
		break

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Invalid Action : %s", action),
		})
	}
}

// Handles route related to managing a challenge
// @Summary Handles challenge management actions.
// @Description Handles challenge management routes with actions which includes - DEPLOY, UNDEPLOY, PURGE.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param name query string true "Name of the challenge to be managed, here name is the unique identifier for challenge"
// @Param action query string true "Action for the challenge"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 402 {object} api.HTTPPlainResp
// @Router /api/manage/challenge/ [post]
func manageChallengeHandler(c *gin.Context) {
	identifier := c.PostForm("name")
	action := c.PostForm("action")

	challengeId, error := database.QueryFirstChallengeEntry("name", identifier)
	if error != nil {
		log.Infof("Error while getting challenge ID")
	}

	TransactionEntry := database.Transaction{
		Action:      action,
		UserId:      decryptToken(c.GetHeader("Authorization")),
		ChallengeID: challengeId.ID,
	}

	log.Infof("Trying %s for challenge with identifier : %s", action, identifier)

	switch action {
	case core.MANAGE_ACTION_UNDEPLOY:
		if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
			fmt.Printf("Error")
		}
		if err := manager.UndeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

	case core.MANAGE_ACTION_PURGE:
		if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
			fmt.Printf("Error")
		}
		if err := manager.PurgeChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

	case core.MANAGE_ACTION_REDEPLOY:
		// Redeploying a challenge means to first purge the challenge and then try to deploy it.
		if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
			fmt.Printf("Error")
		}
		if err := manager.RedeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

	case core.MANAGE_ACTION_DEPLOY:
		// For deploy, identifier is name
		if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
			fmt.Printf("Error")
		}
		if err := manager.DeployChallenge(identifier); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": err.Error(),
			})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Invalid Action : %s", action),
		})
		return
	}

	respStr := fmt.Sprintf("Your action %s on challenge %s has been triggered, check stats.", action, identifier)
	c.JSON(http.StatusOK, gin.H{
		"message": respStr,
	})
}

// Deploy local challenge
// @Summary Deploy a local challenge using the path provided in the post parameter
// @Description Handles deployment of a challenge using the absolute directory path
// @Tags manage
// @Accept  json
// @Produce application/json
// @Param challenge_dir query string true "Challenge Directory"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/deploy/local [post]
func deployLocalChallengeHandler(c *gin.Context) {
	identifier := c.PostForm("name")
	action := c.PostForm("action")
	challDir := c.PostForm("challenge_dir")
	if challDir == "" {
		c.JSON(http.StatusNotAcceptable, gin.H{
			"message": "No challenge directory specified",
		})
		return
	}
	challengeId, error := database.QueryFirstChallengeEntry("name", identifier)
	if error != nil {
		log.Infof("Error while getting challenge ID")
	}

	TransactionEntry := database.Transaction{
		Action:      action,
		UserId:      decryptToken(c.GetHeader("Authorization")),
		ChallengeID: challengeId.ID,
	}
	log.Info("In local deploy challenge Handler")
	err := manager.DeployChallengePipeline(challDir)
	if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
		fmt.Printf("Error")
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}

	challengeName := filepath.Base(challDir)
	respStr := fmt.Sprintf("Deploy for challenge %s has been triggered.\n", challengeName)

	c.JSON(http.StatusOK, gin.H{
		"message": respStr,
	})
}

// Handles route related to beast static content serving container
// @Summary Handles route related to beast static content serving container, takes action as route parameter and perform that action
// @Description Handles beast static content serving container routes.
// @Tags manage
// @Accept  json
// @Produce application/json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/manage/static/:action [post]
func beastStaticContentHandler(c *gin.Context) {
	action := c.Param("action")
	identifier := c.Param("name")
	challengeId, error := database.QueryFirstChallengeEntry("name", identifier)
	if error != nil {
		log.Infof("Error while getting challenge ID")
	}

	TransactionEntry := database.Transaction{
		Action:      action,
		UserId:      decryptToken(c.GetHeader("Authorization")),
		ChallengeID: challengeId.ID,
	}
	switch action {
	case core.MANAGE_ACTION_DEPLOY:
		go manager.DeployStaticContentContainer()
		if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
			fmt.Printf("Error")
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Static container deploy started",
		})
		return

	case core.MANAGE_ACTION_UNDEPLOY:
		go manager.UndeployStaticContentContainer()
		if trans := database.SaveTransaction(&TransactionEntry); trans != nil {
			fmt.Printf("Error")
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Static content container undeploy started",
		})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Invalid Action : %s", action),
		})
	}
}
