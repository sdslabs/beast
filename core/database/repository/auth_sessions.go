package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateSession(ctx context.Context, db *gorm.DB, session *models.AuthSession) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	if err := db.Create(session).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetSessionByID(ctx context.Context, db *gorm.DB, id int64) (*models.AuthSession, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var session models.AuthSession
	if err := db.Where("id = ?", id).First(&session).Error; err != nil {
		return nil, MapError(err)
	}
	return &session, nil
}

func GetSessionByRefreshHash(ctx context.Context, db *gorm.DB, hash string) (*models.AuthSession, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var session models.AuthSession
	if err := db.Where("refresh_token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, MapError(err)
	}
	return &session, nil
}

func ListSessionsByUserID(ctx context.Context, db *gorm.DB, userID uuid.UUID) ([]models.AuthSession, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var sessions []models.AuthSession
	if err := db.Where("user_id = ? AND is_revoked = FALSE", userID).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, MapError(err)
	}
	return sessions, nil
}

func RotateSessionRefreshHash(ctx context.Context, db *gorm.DB, id int64, oldHash, newHash string, newExpiresAt, lastUsedAt time.Time) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.AuthSession{}).
		Where("id = ? AND refresh_token_hash = ? AND is_revoked = FALSE", id, oldHash).
		Updates(map[string]any{
			"refresh_token_hash": newHash,
			"expires_at":         newExpiresAt,
			"last_used_at":       lastUsedAt,
		})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func RevokeSession(ctx context.Context, db *gorm.DB, id int64) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.AuthSession{}).Where("id = ?", id).Update("is_revoked", true)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func RevokeAllSessionsForUser(ctx context.Context, db *gorm.DB, userID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.AuthSession{}).
		Where("user_id = ? AND is_revoked = FALSE", userID).
		Update("is_revoked", true)
	if res.Error != nil {
		return MapError(res.Error)
	}
	return nil
}

func DeleteExpiredSessions(ctx context.Context, db *gorm.DB, before time.Time) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("expires_at < ?", before).Delete(&models.AuthSession{})
	if res.Error != nil {
		return 0, MapError(res.Error)
	}
	return res.RowsAffected, nil
}
