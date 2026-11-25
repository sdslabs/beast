package repository

import (
	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateHint(hint *models.ChallengeHint) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if hint.ID == uuid.Nil {
		hint.ID = uuid.New()
	}
	if err := db.Create(hint).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetHintByID(id uuid.UUID) (*models.ChallengeHint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var hint models.ChallengeHint
	if err := db.Where("id = ?", id).First(&hint).Error; err != nil {
		return nil, MapError(err)
	}
	return &hint, nil
}

func ListHintsByChallengeID(challengeID uuid.UUID) ([]models.ChallengeHint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var hints []models.ChallengeHint
	if err := db.Where("challenge_id = ?", challengeID).
		Order("cost_points ASC").
		Find(&hints).Error; err != nil {
		return nil, MapError(err)
	}
	return hints, nil
}

func ListHintsByChallengeIDOmitContent(challengeID uuid.UUID) ([]models.ChallengeHint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func UpdateHint(updatedHint models.ChallengeHint) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func DeleteHint(id uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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
