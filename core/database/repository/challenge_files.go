package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateChallengeFile(file *models.ChallengeFile) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func GetChallengeFileByID(id uuid.UUID) (*models.ChallengeFile, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var file models.ChallengeFile
	if err := db.Where("id = ?", id).First(&file).Error; err != nil {
		return nil, MapError(err)
	}
	return &file, nil
}

// ListChallengeFiles retrieves files attached to a challenge (both shared and team-specific), omitting the server storage path.
func ListChallengeFiles(challengeID uuid.UUID, teamID *uuid.UUID) ([]models.ChallengeFile, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func DeleteChallengeFile(id uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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
