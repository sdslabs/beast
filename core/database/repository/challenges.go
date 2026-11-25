package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm/clause"
)

func CreateChallenge(challenge *models.Challenge) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if challenge.ID == uuid.Nil {
		challenge.ID = uuid.New()
	}
	now := time.Now()
	if challenge.CreatedAt.IsZero() {
		challenge.CreatedAt = now
	}
	if challenge.UpdatedAt.IsZero() {
		challenge.UpdatedAt = now
	}
	if challenge.DeploymentMode == "" {
		challenge.DeploymentMode = models.DeploymentModeOffline
	}
	if challenge.DecaySolves <= 0 {
		challenge.DecaySolves = 50
	}
	if challenge.GeneratorTimeoutS <= 0 {
		challenge.GeneratorTimeoutS = 30
	}
	if err := db.Create(challenge).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func UpsertChallenge(challenge *models.Challenge) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if challenge.ID == uuid.Nil {
		challenge.ID = uuid.New()
	}
	now := time.Now()
	if challenge.CreatedAt.IsZero() {
		challenge.CreatedAt = now
	}
	challenge.UpdatedAt = now

	if challenge.DeploymentMode == "" {
		challenge.DeploymentMode = models.DeploymentModeOffline
	}
	if challenge.DecaySolves <= 0 {
		challenge.DecaySolves = 50
	}
	if challenge.GeneratorTimeoutS <= 0 {
		challenge.GeneratorTimeoutS = 30
	}

	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "slug"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"description",
			"author",
			"max_points",
			"min_points",
			"decay_solves",
			"max_attempts",
			"flag_template",
			"is_hidden",
			"release_at",
			"deployment_mode",
			"generator_path",
			"generator_timeout_s",
			"updated_at",
		}),
	}).Create(challenge).Error

	if err != nil {
		return MapError(err)
	}
	return nil
}

func GetChallengeBySlug(slug string) (*models.Challenge, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var chall models.Challenge
	if err := db.Where("slug = ?", slug).First(&chall).Error; err != nil {
		return nil, MapError(err)
	}
	return &chall, nil
}

func GetChallengeByID(id uuid.UUID) (*models.Challenge, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var chall models.Challenge
	if err := db.Where("id = ?", id).First(&chall).Error; err != nil {
		return nil, MapError(err)
	}
	return &chall, nil
}

func ListChallenges(includeHidden bool) ([]models.Challenge, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var challs []models.Challenge
	q := db.Model(&models.Challenge{}).Order("created_at ASC")
	if !includeHidden {
		q = q.Where("is_hidden = FALSE")
	}
	if err := q.Find(&challs).Error; err != nil {
		return nil, MapError(err)
	}
	return challs, nil
}

func ListChallengesByIDs(ids []uuid.UUID) ([]models.Challenge, error) {
	if len(ids) == 0 {
		return []models.Challenge{}, nil
	}
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var challs []models.Challenge
	if err := db.Where("id IN ?", ids).Find(&challs).Error; err != nil {
		return nil, MapError(err)
	}
	return challs, nil
}

func CountChallenges(includeHidden bool) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	q := db.Model(&models.Challenge{})
	if !includeHidden {
		q = q.Where("is_hidden = FALSE")
	}
	if err := q.Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func DeleteChallenge(id uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Where("id = ?", id).Delete(&models.Challenge{})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
