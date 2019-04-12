package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/manager"

	log "github.com/sirupsen/logrus"
)

// This syncs beasts local challenges database with the remote git repository(hack)
// @Summary Syncs beast's local copy of remote git repository for challenges.
// @Description Syncs beasts local challenges database with the remote git repository(hack) the local copy of the challenge database is located at $HOME/.beast/remote/$REMOTE_NAME.
// @Accept  json
// @Produce application/json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/remote/sync/ [post]
func syncBeastGitRemote(c *gin.Context) {
	err := manager.SyncBeastRemote()
	if err != nil {
		log.Errorf("Error while syncing beast remote....")
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "Error while syncing beast remote",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "REMOTE SYNC DONE",
	})
}

// This resets beast git remote directory located in ~/.beast/remote/$REMOTE_NAME
// @Summary Resets beast local copy of remote git repository.
// @Description Resets local copy of remote git directory, it first deletes the existing directory and then clone from the remote again.
// @Accept  json
// @Produce application/json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPPlainResp
// @Router /api/remote/reset/ [post]
func resetBeastGitRemote(c *gin.Context) {
	err := manager.ResetBeastRemote()
	if err != nil {
		log.Errorf("Error while resetting beast remote....")
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{
			Message: "Error while resetting beast remote",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "REMOTE RESET DONE",
	})
}
