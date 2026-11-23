package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// AdminListNotificationsHandler handles GET /api/v1/admin/notifications
func AdminListNotificationsHandler(c *gin.Context) {
	var query dto.ListNotificationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_QUERY", err.Error())
		return
	}

	// TODO: implement administrative notifications list
	var resp dto.ListNotificationsResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminCreateNotificationHandler handles POST /api/v1/admin/notifications
func AdminCreateNotificationHandler(c *gin.Context) {
	var req dto.AdminCreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative notification broadcast creation
	var resp dto.NotificationResponse
	httputil.RespondJSON(c, http.StatusCreated, resp)
}

// AdminGetNotificationHandler handles GET /api/v1/admin/notifications/:notification_id
func AdminGetNotificationHandler(c *gin.Context) {
	notificationID := c.Param("notification_id")
	_ = notificationID

	// TODO: implement administrative notification inspection
	var resp dto.NotificationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminUpdateNotificationHandler handles PATCH /api/v1/admin/notifications/:notification_id
func AdminUpdateNotificationHandler(c *gin.Context) {
	notificationID := c.Param("notification_id")
	_ = notificationID

	var req dto.AdminUpdateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// TODO: implement administrative notification update
	var resp dto.NotificationResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// AdminDeleteNotificationHandler handles DELETE /api/v1/admin/notifications/:notification_id
func AdminDeleteNotificationHandler(c *gin.Context) {
	notificationID := c.Param("notification_id")
	_ = notificationID

	// TODO: implement administrative notification deletion
	var resp dto.GenericMessageResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
