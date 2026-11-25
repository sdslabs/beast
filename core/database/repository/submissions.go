package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateSubmission(sub *models.Submission) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func GetSubmissionByID(id uuid.UUID) (*models.Submission, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var sub models.Submission
	if err := db.Where("id = ?", id).First(&sub).Error; err != nil {
		return nil, MapError(err)
	}
	return &sub, nil
}

func ListSubmissions(offset, limit int) ([]models.Submission, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func ListSubmissionsByTeamID(teamID uuid.UUID, offset, limit int) ([]models.Submission, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func ListSubmissionsByChallengeID(challengeID uuid.UUID, offset, limit int) ([]models.Submission, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func CountSubmissions() (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.Submission{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSubmissionsByTeamID(teamID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.Submission{}).Where("team_id = ?", teamID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSubmissionsByChallenge(challengeID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.Submission{}).Where("challenge_id = ?", challengeID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSubmissionsByTeamAndChallenge(teamID, challengeID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.Submission{}).
		Where("team_id = ? AND challenge_id = ?", teamID, challengeID).
		Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}
