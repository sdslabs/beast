package repository

import (
	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm/clause"
)

func UpsertRuntimeConfig(cfg *models.ChallengeRuntimeConfig) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func GetRuntimeConfig(challengeID uuid.UUID) (*models.ChallengeRuntimeConfig, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var cfg models.ChallengeRuntimeConfig
	if err := db.Where("challenge_id = ?", challengeID).First(&cfg).Error; err != nil {
		return nil, MapError(err)
	}
	return &cfg, nil
}

func DeleteRuntimeConfig(challengeID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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
