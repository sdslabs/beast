package repository

import (
	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm/clause"
)

func AddChallengePrerequisite(challengeID uuid.UUID, requiresID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	prereq := models.ChallengePrerequisite{
		ChallengeID: challengeID,
		RequiresID:  requiresID,
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&prereq).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func RemoveChallengePrerequisite(challengeID uuid.UUID, requiresID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Where("challenge_id = ? AND requires_id = ?", challengeID, requiresID).
		Delete(&models.ChallengePrerequisite{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func ListPrerequisitesByChallengeID(challengeID uuid.UUID) ([]uuid.UUID, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var requiredIDs []uuid.UUID
	err := db.Model(&models.ChallengePrerequisite{}).
		Where("challenge_id = ?", challengeID).
		Pluck("requires_id", &requiredIDs).Error
	if err != nil {
		return nil, MapError(err)
	}
	return requiredIDs, nil
}

func ListPrerequisitesByChallengeIDs(challengeIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	res := make(map[uuid.UUID][]uuid.UUID)
	if len(challengeIDs) == 0 {
		return res, nil
	}
	for _, cid := range challengeIDs {
		res[cid] = []uuid.UUID{}
	}

	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var prereqs []models.ChallengePrerequisite
	if err := db.Where("challenge_id IN ?", challengeIDs).Find(&prereqs).Error; err != nil {
		return nil, MapError(err)
	}

	for _, p := range prereqs {
		res[p.ChallengeID] = append(res[p.ChallengeID], p.RequiresID)
	}
	return res, nil
}
