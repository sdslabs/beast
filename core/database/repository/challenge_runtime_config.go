package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpsertRuntimeConfig(ctx context.Context, db *gorm.DB, cfg *models.ChallengeRuntimeConfig) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "challenge_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"runtime_type",
			"image_ref",
			"compose_path",
		}),
	}).Create(cfg).Error
	if err != nil {
		return MapError(err)
	}
	return nil
}

func GetRuntimeConfig(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) (*models.ChallengeRuntimeConfig, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var cfg models.ChallengeRuntimeConfig
	if err := db.Where("challenge_id = ?", challengeID).First(&cfg).Error; err != nil {
		return nil, MapError(err)
	}
	return &cfg, nil
}

func DeleteRuntimeConfig(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("challenge_id = ?", challengeID).Delete(&models.ChallengeRuntimeConfig{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
