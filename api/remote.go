package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/git"

	log "github.com/sirupsen/logrus"
)

// This syncs beasts local challenges database with the remote git repository(hack)
// @Summary Syncs beast's local copy of remote git repository for challenges.
// @Description Syncs beasts local challenges database with the remote git repository(hack) the local copy of the challenge database is located at $HOME/.beast/remote/$REMOTE_NAME.
// @Accept  json
// @Produce application/json
// @Success 200 {JSON} Success
// @Failure 500 {JSON} InternalServerError
// @Router /api/remote/sync/ [post]
func syncBeastGitRemote(c *gin.Context) {
	err := git.SyncBeastRemote()
	if err != nil {
		log.Errorf("Error while syncing beast remote....")
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error while syncing beast remote",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "REMOTE SYNC DONE",
	})
}

// This resets beast git remote directory located in ~/.beast/remote/$REMOTE_NAME
// @Summary Resets beast local copy of remote git repository.
// @Description Resets local copy of remote git directory, it first deletes the existing directory and then clone from the remote again.
// @Accept  json
// @Produce application/json
// @Success 200 {JSON} Success
// @Failure 500 {JSON} InternalServerError
// @Router /api/remote/reset/ [post]
func resetBeastGitRemote(c *gin.Context) {
	err := git.ResetBeastRemote()
	if err != nil {
		log.Errorf("Error while resetting beast remote....")
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error while resetting beast remote",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "REMOTE RESET DONE",
	})
}
