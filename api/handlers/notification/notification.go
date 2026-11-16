package notification

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListNotificationsHandler handles GET /api/v1/notifications
func ListNotificationsHandler(c *gin.Context) {
	// TODO: implement notifications listing
	httputil.RespondNotImplemented(c, "ListNotificationsHandler")
}

// GetNotificationHandler handles GET /api/v1/notifications/:notification_id
func GetNotificationHandler(c *gin.Context) {
	// TODO: implement notification details retrieval
	httputil.RespondNotImplemented(c, "GetNotificationHandler")
}

// MarkNotificationReadHandler handles PUT /api/v1/notifications/:notification_id/read
func MarkNotificationReadHandler(c *gin.Context) {
	// TODO: implement marking single notification as read
	httputil.RespondNotImplemented(c, "MarkNotificationReadHandler")
}

// MarkAllNotificationsReadHandler handles POST /api/v1/notifications/read-all
func MarkAllNotificationsReadHandler(c *gin.Context) {
	// TODO: implement marking all notifications as read
	httputil.RespondNotImplemented(c, "MarkAllNotificationsReadHandler")
}
