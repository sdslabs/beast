package repository

import (
	"context"
	"time"

	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateAuditEvent(ctx context.Context, db *gorm.DB, event *models.AuditEvent) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if err := db.Create(event).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func ListAuditEvents(ctx context.Context, db *gorm.DB, offset, limit int) ([]models.AuditEvent, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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
