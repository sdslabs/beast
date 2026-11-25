package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateSession(session *models.AuthSession) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	if err := db.Create(session).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetSessionByID(id int64) (*models.AuthSession, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var session models.AuthSession
	if err := db.Where("id = ?", id).First(&session).Error; err != nil {
		return nil, MapError(err)
	}
	return &session, nil
}

func GetSessionByRefreshHash(hash string) (*models.AuthSession, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var session models.AuthSession
	if err := db.Where("refresh_token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, MapError(err)
	}
	return &session, nil
}

func ListSessionsByUserID(userID uuid.UUID) ([]models.AuthSession, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var sessions []models.AuthSession
	if err := db.Where("user_id = ? AND is_revoked = FALSE", userID).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, MapError(err)
	}
	return sessions, nil
}

func RotateSessionRefreshHash(id int64, newHash string, newExpiresAt time.Time, lastUsedAt time.Time) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.AuthSession{}).Where("id = ?", id).Updates(map[string]any{
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

func RevokeSession(id int64) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func RevokeAllSessionsForUser(userID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.AuthSession{}).
		Where("user_id = ? AND is_revoked = FALSE", userID).
		Update("is_revoked", true)
	if res.Error != nil {
		return MapError(res.Error)
	}
	return nil
}

func DeleteExpiredSessions(before time.Time) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	res := db.Where("expires_at < ?", before).Delete(&models.AuthSession{})
	if res.Error != nil {
		return 0, MapError(res.Error)
	}
	return res.RowsAffected, nil
}
