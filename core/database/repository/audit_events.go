package repository

import (
	"time"

	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateAuditEvent(event *models.AuditEvent) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if err := db.Create(event).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func ListAuditEvents(offset, limit int) ([]models.AuditEvent, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var events []models.AuditEvent
	q := db.Model(&models.AuditEvent{}).Order("created_at DESC").Limit(limit).Offset(offset)

	if err := q.Find(&events).Error; err != nil {
		return nil, MapError(err)
	}
	return events, nil
}
