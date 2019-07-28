package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
)

// Adds notifications
// @Summary Adds notifications
// @Description Adds notifications
// @Tags notification
// @Accept  json
// @Produce json
// @Param title query string true "Title of notification to be added"
// @Param desc query string true "Description for the notification to be added"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/add [post]
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

// Removes notifications
// @Summary Removes notifications
// @Description Removes notifications
// @Tags notification
// @Accept  json
// @Produce json
// @Param title query string true "Title of notification to be added"
// @Param desc query string true "Description for the notification to be added"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/delete [post]
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

// Updates notifications
// @Summary Updates notifications
// @Description Updates any changes in the notification notifications
// @Tags notification
// @Accept  json
// @Produce json
// @Param title query string true "Title of notification to be added"
// @Param desc query string true "Description for the notification to be added"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/update [post]
func updateNotifications(c *gin.Context) {
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
