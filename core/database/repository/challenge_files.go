package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateChallengeFile(ctx context.Context, db *gorm.DB, file *models.ChallengeFile) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if file.ID == uuid.Nil {
		file.ID = uuid.New()
	}
	if file.CreatedAt.IsZero() {
		file.CreatedAt = time.Now()
	}
	if err := db.Create(file).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetChallengeFileByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.ChallengeFile, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var file models.ChallengeFile
	if err := db.Where("id = ?", id).First(&file).Error; err != nil {
		return nil, MapError(err)
	}
	return &file, nil
}

func ListChallengeFiles(ctx context.Context, db *gorm.DB, challengeID uuid.UUID, teamID *uuid.UUID) ([]models.ChallengeFile, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var files []models.ChallengeFile
	q := db.Model(&models.ChallengeFile{}).Omit("storage_path").Where("challenge_id = ?", challengeID)
	if teamID != nil {
		q = q.Where("team_id IS NULL OR team_id = ?", *teamID)
	} else {
		q = q.Where("team_id IS NULL")
	}
	if err := q.Order("created_at ASC").Find(&files).Error; err != nil {
		return nil, MapError(err)
	}
	return files, nil
}

func DeleteChallengeFile(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Where("id = ?", id).Delete(&models.ChallengeFile{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
