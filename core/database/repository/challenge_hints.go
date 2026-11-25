package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateHint(ctx context.Context, db *gorm.DB, hint *models.ChallengeHint) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if hint.ID == uuid.Nil {
		hint.ID = uuid.New()
	}
	if err := db.Create(hint).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetHintByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.ChallengeHint, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var hint models.ChallengeHint
	if err := db.Where("id = ?", id).First(&hint).Error; err != nil {
		return nil, MapError(err)
	}
	return &hint, nil
}

func ListHintsByChallengeID(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) ([]models.ChallengeHint, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var hints []models.ChallengeHint
	if err := db.Where("challenge_id = ?", challengeID).
		Order("cost_points ASC").
		Find(&hints).Error; err != nil {
		return nil, MapError(err)
	}
	return hints, nil
}

func ListHintsByChallengeIDOmitContent(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) ([]models.ChallengeHint, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var hints []models.ChallengeHint
	if err := db.Model(&models.ChallengeHint{}).
		Where("challenge_id = ?", challengeID).
		Omit("content").
		Order("cost_points ASC").
		Find(&hints).Error; err != nil {
		return nil, MapError(err)
	}
	return hints, nil
}

func UpdateHint(ctx context.Context, db *gorm.DB, updatedHint models.ChallengeHint) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.ChallengeHint{}).
		Where("id = ?", updatedHint.ID).
		Select("content", "cost_points", "release_at").
		Updates(updatedHint)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func DeleteHint(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("id = ?", id).Delete(&models.ChallengeHint{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
