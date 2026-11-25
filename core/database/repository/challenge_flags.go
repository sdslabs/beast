package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateChallengeFlag(flag *models.ChallengeFlag) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func GetFlagsByHash(flagHash string) ([]models.ChallengeFlag, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var flags []models.ChallengeFlag
	if err := db.Where("flag_hash = ?", flagHash).Find(&flags).Error; err != nil {
		return nil, MapError(err)
	}
	return flags, nil
}

func DeleteChallengeFlag(id uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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
