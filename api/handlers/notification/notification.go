package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListNotificationsHandler handles GET /api/v1/notifications
func ListNotificationsHandler(c *gin.Context) {
	var query dto.ListNotificationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement notifications listing
	var resp dto.ListNotificationsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// GetNotificationHandler handles GET /api/v1/notifications/:notification_id
func GetNotificationHandler(c *gin.Context) {
	notificationID := c.Param("notification_id")
	_ = notificationID

	// TODO: implement notification details retrieval
	var resp dto.NotificationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// MarkNotificationReadHandler handles PUT /api/v1/notifications/:notification_id/read
func MarkNotificationReadHandler(c *gin.Context) {
	notificationID := c.Param("notification_id")
	_ = notificationID

	// TODO: implement marking single notification as read
	var resp dto.NotificationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// MarkAllNotificationsReadHandler handles POST /api/v1/notifications/read-all
func MarkAllNotificationsReadHandler(c *gin.Context) {
	// TODO: implement marking all notifications as read
	var resp dto.ReadAllNotificationsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
