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
		c.String(http.StatusInternalServerError, "Error while syncing beast remote")
		return
	}

	c.String(http.StatusOK, "REMOTE SYNC DONE")
}

func resetBeastGitRemote(c *gin.Context) {
	err := git.ResetBeastRemote()
	if err != nil {
		log.Errorf("Error while resetting beast remote....")
		c.String(http.StatusInternalServerError, "Error while resetting beast remote")
		return
	}

	c.String(http.StatusOK, "REMOTE RESET DONE")
}
