package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
)

func addNotification(c *gin.Context) {
	title := c.PostForm("title")
	desc := c.PostForm("desc")

	Notify := database.Notification{
		Title:       title,
		Description: desc,
	}

	if msgs := database.AddNotification(&Notify); msgs != nil {
		log.Info("error while adding notification")
	}
}

func removeNotification(c *gin.Context) {
	title := c.PostForm("title")
	desc := c.PostForm("desc")

	Notify := database.Notification{
		Title:       title,
		Description: desc,
	}

	if msgs := database.DeleteNotification(&Notify); msgs != nil {
		log.Info("error while deleting notification")
	}
}

func updateChallenge(c *gin.Context) {
	title := c.PostForm("title")
	desc := c.PostForm("desc")
	changedTitle := c.PostForm("changedTitle")
	changedDesc := c.PostForm("changedDesc")

	if title != "" && desc != "" {
		Notify := database.Notification{
			Title:       title,
			Description: desc,
		}

		if title == "" && desc != "" {
			if msgs := database.UpdateNotification(&Notify, map[string]interface{}{
				"Description": changedDesc,
			}); msgs != nil {
				log.Info("error while deleting notification")
			}
		} else if title != "" && desc == "" {
			if msgs := database.UpdateNotification(&Notify, map[string]interface{}{
				"Title": changedTitle,
			}); msgs != nil {
				log.Info("error while deleting notification")
			}
		} else {
			if msgs := database.UpdateNotification(&Notify, map[string]interface{}{
				"Title":       changedTitle,
				"Description": changedDesc,
			}); msgs != nil {
				log.Info("error while deleting notification")
			}
		}
	}
}