package repository

import (
	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm/clause"
)

func UpsertInstanceConfig(cfg *models.ChallengeInstanceConfig) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func GetInstanceConfig(challengeID uuid.UUID) (*models.ChallengeInstanceConfig, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var cfg models.ChallengeInstanceConfig
	if err := db.Where("challenge_id = ?", challengeID).First(&cfg).Error; err != nil {
		return nil, MapError(err)
	}
	return &cfg, nil
}

func DeleteInstanceConfig(challengeID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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
