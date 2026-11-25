package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateSubmission(ctx context.Context, db *gorm.DB, sub *models.Submission) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}
	if err := db.Create(sub).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetSubmissionByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.Submission, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var sub models.Submission
	if err := db.Where("id = ?", id).First(&sub).Error; err != nil {
		return nil, MapError(err)
	}
	return &sub, nil
}

func ListSubmissions(ctx context.Context, db *gorm.DB, offset, limit int) ([]models.Submission, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var list []models.Submission
	q := db.Model(&models.Submission{}).Order("created_at DESC").Limit(limit).Offset(offset)

	if err := q.Find(&list).Error; err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListSubmissionsByTeamID(ctx context.Context, db *gorm.DB, teamID uuid.UUID, offset, limit int) ([]models.Submission, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var list []models.Submission
	q := db.Where("team_id = ?", teamID).Order("created_at DESC").Limit(limit).Offset(offset)
	if err := q.Find(&list).Error; err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListSubmissionsByChallengeID(ctx context.Context, db *gorm.DB, challengeID uuid.UUID, offset, limit int) ([]models.Submission, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var list []models.Submission
	q := db.Where("challenge_id = ?", challengeID).Order("created_at DESC").Limit(limit).Offset(offset)
	if err := q.Find(&list).Error; err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func CountSubmissions(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Submission{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSubmissionsByTeamID(ctx context.Context, db *gorm.DB, teamID uuid.UUID) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Submission{}).Where("team_id = ?", teamID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSubmissionsByChallenge(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Submission{}).Where("challenge_id = ?", challengeID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSubmissionsByTeamAndChallenge(ctx context.Context, db *gorm.DB, teamID, challengeID uuid.UUID) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Submission{}).
		Where("team_id = ? AND challenge_id = ?", teamID, challengeID).
		Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}
