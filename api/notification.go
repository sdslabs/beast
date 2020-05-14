package api

import (
	"net/http"

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

	notify := database.Notification{
		Title:       title,
		Description: desc,
	}

	if msgs := database.AddNotification(&notify); msgs != nil {
		log.Info("Error while adding notification")
	}
}

// Removes notifications
// @Summary Removes notifications
// @Description Removes notifications
// @Tags notification
// @Accept  json
// @Produce json
// @Param title query string true "Title of notification"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/delete [post]
func removeNotification(c *gin.Context) {
	title := c.PostForm("title")

	notify := database.Notification{
		Title: title,
	}

	if msgs := database.DeleteNotification(&notify); msgs != nil {
		log.Info("Error while deleting notification")
	}
}

// Updates notifications
// @Summary Updates notifications
// @Description Updates any changes in the notification notifications
// @Tags notification
// @Accept  json
// @Produce json
// @Param title query string true "Title of notification"
// @Param description query string true "Description for the notification to be changed"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/update [post]
func updateNotifications(c *gin.Context) {
	title := c.PostForm("title")
	changedDescription := c.PostForm("description")

	if title != "" && changedDescription != "" {
		notify := database.Notification{
			Title: title,
		}

		if changedDescription != "" {
			if msgs := database.UpdateNotification(&notify, map[string]interface{}{
				"Description": changedDescription,
			}); msgs != nil {
				log.Info("Error while updating notification")
			}
		}
	}
}

func availableNotificationHandler(c *gin.Context) {
	notifications, err := database.QueryAllNotification()
	if err != nil {
		log.Errorf("Error while retriving notifications")
	} else if len(notifications) == 0 {
		log.Info("No notifications present in database")
	} else {
		c.JSON(http.StatusOK, NotificationResp{
			Message:       "All notifications",
			Notifications: notifications,
		})
		return
	}
}
