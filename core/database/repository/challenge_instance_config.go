package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpsertInstanceConfig(ctx context.Context, db *gorm.DB, cfg *models.ChallengeInstanceConfig) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if cfg.TTLMinutes <= 0 {
		cfg.TTLMinutes = 30
	}
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "challenge_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"ttl_minutes",
			"max_renewals",
		}),
	}).Create(cfg).Error
	if err != nil {
		return MapError(err)
	}
	return nil
}

func GetInstanceConfig(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) (*models.ChallengeInstanceConfig, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var cfg models.ChallengeInstanceConfig
	if err := db.Where("challenge_id = ?", challengeID).First(&cfg).Error; err != nil {
		return nil, MapError(err)
	}
	return &cfg, nil
}

func DeleteInstanceConfig(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("challenge_id = ?", challengeID).Delete(&models.ChallengeInstanceConfig{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
