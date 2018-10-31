package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/git"

	log "github.com/sirupsen/logrus"
)

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
