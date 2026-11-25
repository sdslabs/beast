package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateChallenge(ctx context.Context, db *gorm.DB, challenge *models.Challenge) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func UpsertChallenge(ctx context.Context, db *gorm.DB, challenge *models.Challenge) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

	err := db.Clauses(
		clause.OnConflict{
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
		},
		clause.Returning{},
	).Create(challenge).Error

	if err != nil {
		return MapError(err)
	}
	return nil
}

func GetChallengeBySlug(ctx context.Context, db *gorm.DB, slug string) (*models.Challenge, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var chall models.Challenge
	if err := db.Where("slug = ?", slug).First(&chall).Error; err != nil {
		return nil, MapError(err)
	}
	return &chall, nil
}

func GetChallengeByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.Challenge, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var chall models.Challenge
	if err := db.Where("id = ?", id).First(&chall).Error; err != nil {
		return nil, MapError(err)
	}
	return &chall, nil
}

func ListChallenges(ctx context.Context, db *gorm.DB, includeHidden bool) ([]models.Challenge, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func ListChallengesByIDs(ctx context.Context, db *gorm.DB, ids []uuid.UUID) ([]models.Challenge, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if len(ids) == 0 {
		return []models.Challenge{}, nil
	}

	var challs []models.Challenge
	if err := db.Where("id IN ?", ids).Find(&challs).Error; err != nil {
		return nil, MapError(err)
	}
	return challs, nil
}

func CountChallenges(ctx context.Context, db *gorm.DB, includeHidden bool) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func DeleteChallenge(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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
