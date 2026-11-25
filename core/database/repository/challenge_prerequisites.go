package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func AddChallengePrerequisite(ctx context.Context, db *gorm.DB, challengeID uuid.UUID, requiresID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func RemoveChallengePrerequisite(ctx context.Context, db *gorm.DB, challengeID uuid.UUID, requiresID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func ListPrerequisitesByChallengeID(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) ([]uuid.UUID, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func ListPrerequisitesByChallengeIDs(ctx context.Context, db *gorm.DB, challengeIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := make(map[uuid.UUID][]uuid.UUID)
	if len(challengeIDs) == 0 {
		return res, nil
	}
	for _, cid := range challengeIDs {
		res[cid] = []uuid.UUID{}
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
