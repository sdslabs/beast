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

	if title == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Notification title cannot be empty",
		})
		return
	}

	notify := database.Notification{
		Title:       title,
		Description: desc,
	}

	if err := database.AddNotification(&notify); err != nil {
		log.Info("Error while adding notification")
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Error while adding notification",
		})
		return
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Notification successfully added",
	})
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
	id := c.PostForm("id")

	if id == "" {
		log.Errorf("Please provide notification Id")
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Please provide notification Id",
		})
		return
	}

	notify, err := database.QueryFirstNotificationEntry("ID", id)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Error while accessing database",
		})
		return
	}

	if notify.ID == 0 {
		log.Errorf("No notification exist with id : %s", id)
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "No such notification exists",
		})
		return
	}

	if err := database.DeleteNotification(&notify); err != nil {
		log.Errorf("Error while deleting notification from the database: %v", err)
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Error while deleting notification from the database",
		})
		return
	}
	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Notification successfully removed",
	})
}

// Updates notifications
// @Summary Updates notifications
// @Description Updates any changes in the notifications
// @Tags notification
// @Accept  json
// @Produce json
// @Param title query string true "Title of notification"
// @Param description query string true "Description for the notification to be changed"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/update [post]
func updateNotifications(c *gin.Context) {
	id := c.PostForm("id")
	changedtitle := c.PostForm("title")
	changedDescription := c.PostForm("desc")

	if changedtitle != "" && changedDescription != "" {
		notify, err := database.QueryFirstNotificationEntry("ID", id)
		if err != nil {
			log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
			c.JSON(http.StatusOK, HTTPPlainResp{
				Message: "Error while accessing database",
			})
			return
		}

		if notify.ID == 0 {
			log.Errorf("No notification exist with id : %s", id)
			c.JSON(http.StatusOK, HTTPPlainResp{
				Message: "No such notification exists",
			})
			return
		}

		if err := database.UpdateNotification(&notify, map[string]interface{}{
			"Description": changedDescription,
			"Title":       changedtitle,
		}); err != nil {
			log.Info("Error while updating notification")
			c.JSON(http.StatusOK, HTTPPlainResp{
				Message: "Error while updating notification",
			})
			return
		}
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Notification successfully updated",
		})
	}
}

// Fetch available notifications
// @Summary Fetch available notifications
// @Description Fetch all the notifications from database
// @Tags notification
// @Accept  json
// @Produce json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/notification/available [post]
func availableNotificationHandler(c *gin.Context) {
	notifications, err := database.QueryAllNotification()
	if err != nil {
		log.Errorf("Error while retriving notifications")
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Error while retriving notifications",
		})
	} else if len(notifications) == 0 {
		log.Info("No notifications present in database")
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "No notification present in database",
		})
	} else {
		var resp []NotificationResp
		for _, notification := range notifications {
			r := NotificationResp{
				ID:        notification.ID,
				Title:     notification.Title,
				Desc:      notification.Description,
				UpdatedAt: notification.UpdatedAt,
			}
			resp = append(resp, r)
		}

		c.JSON(http.StatusOK, resp)
		return
	}
}
