package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm/clause"
)

func CreateNotification(notification *models.Notification) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if notification.Type == "" {
		notification.Type = models.NotificationTypeAnnouncement
	}
	if err := db.Create(notification).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetNotificationByID(id uuid.UUID) (*models.Notification, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var notification models.Notification
	if err := db.Where("id = ?", id).First(&notification).Error; err != nil {
		return nil, MapError(err)
	}
	return &notification, nil
}

// ListNotificationsForUser It assumes that this user is in this team. without content
func ListNotificationsForUser(userID uuid.UUID, teamID uuid.UUID) ([]models.Notification, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var notifications []models.Notification
	q := db.Model(&models.Notification{}).Omit("content")
	q = q.Where("(user_id IS NULL AND team_id IS NULL) OR user_id = ? OR team_id = ?", userID, teamID)

	if err := q.Order("is_pinned DESC, created_at DESC").Find(&notifications).Error; err != nil {
		return nil, MapError(err)
	}
	return notifications, nil
}

func ListAllNotifications(offset, limit int) ([]models.Notification, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}
	var notifications []models.Notification
	q := db.Model(&models.Notification{}).Order("created_at DESC").Limit(limit).Offset(offset)

	if err := q.Find(&notifications).Error; err != nil {
		return nil, MapError(err)
	}
	return notifications, nil
}

func UpdateNotification(notification models.Notification) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.Notification{}).
		Where("id = ?", notification.ID).
		Select("title", "content", "type", "challenge_id", "team_id", "user_id", "is_pinned").
		Updates(notification)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func DeleteNotification(id uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Where("id = ?", id).Delete(&models.Notification{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func MarkNotificationRead(userID uuid.UUID, notificationID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	read := models.UserNotificationRead{
		UserID:         userID,
		NotificationID: notificationID,
		ReadAt:         time.Now(),
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&read).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func MarkAllNotificationsRead(userID uuid.UUID, notificationIDs []uuid.UUID) error {
	if len(notificationIDs) == 0 {
		return nil
	}
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	now := time.Now()
	reads := make([]models.UserNotificationRead, 0, len(notificationIDs))
	for _, nID := range notificationIDs {
		reads = append(reads, models.UserNotificationRead{
			UserID:         userID,
			NotificationID: nID,
			ReadAt:         now,
		})
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&reads).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func ListReadNotificationIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var ids []uuid.UUID
	if err := db.Model(&models.UserNotificationRead{}).
		Where("user_id = ?", userID).
		Pluck("notification_id", &ids).Error; err != nil {
		return nil, MapError(err)
	}
	return ids, nil
}
