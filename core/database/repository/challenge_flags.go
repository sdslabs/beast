package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateChallengeFlag(ctx context.Context, db *gorm.DB, flag *models.ChallengeFlag) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if flag.ID == uuid.Nil {
		flag.ID = uuid.New()
	}
	if flag.CreatedAt.IsZero() {
		flag.CreatedAt = time.Now()
	}
	if err := db.Create(flag).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetFlagsByHash(ctx context.Context, db *gorm.DB, flagHash string) ([]models.ChallengeFlag, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var flags []models.ChallengeFlag
	if err := db.Where("flag_hash = ?", flagHash).Find(&flags).Error; err != nil {
		return nil, MapError(err)
	}
	return flags, nil
}

func DeleteChallengeFlag(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("id = ?", id).Delete(&models.ChallengeFlag{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
