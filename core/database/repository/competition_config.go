package repository

import (
	"context"
	"errors"
	"time"

	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

// TODO : justify the need of this function
func EnsureCompetitionConfig(ctx context.Context, db *gorm.DB, defaultConfig *models.CompetitionConfig) (*models.CompetitionConfig, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var cfg models.CompetitionConfig
	err := db.Where("id = 1").First(&cfg).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, MapError(err)
	}

	if defaultConfig == nil {
		defaultConfig = &models.CompetitionConfig{
			ID:                        1,
			Title:                     "Beast CTF",
			ScoringPolicy:             "dynamic",
			RegistrationOpen:          true,
			MaxTeamSize:               4,
			SubmissionRateLimitPerMin: 10,
		}
	} else {
		defaultConfig.ID = 1
	}
	now := time.Now()
	if defaultConfig.CreatedAt.IsZero() {
		defaultConfig.CreatedAt = now
	}
	if defaultConfig.UpdatedAt.IsZero() {
		defaultConfig.UpdatedAt = now
	}

	if err := db.Create(defaultConfig).Error; err != nil {
		return nil, MapError(err)
	}
	return defaultConfig, nil
}

func GetCompetitionConfig(ctx context.Context, db *gorm.DB) (*models.CompetitionConfig, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var cfg models.CompetitionConfig
	if err := db.Where("id = 1").First(&cfg).Error; err != nil {
		return nil, MapError(err)
	}
	return &cfg, nil
}

func UpdateCompetitionConfig(ctx context.Context, db *gorm.DB, cfg models.CompetitionConfig) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	cfg.ID = 1
	cfg.UpdatedAt = time.Now()
	res := db.Model(&models.CompetitionConfig{}).
		Where("id = 1").
		Select(
			"title",
			"about",
			"description",
			"rules",
			"scoring_policy",
			"registration_open",
			"starts_at",
			"ends_at",
			"max_team_size",
			"submission_rate_limit_per_min",
			"updated_at",
		).
		Updates(cfg)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func SetCompetitionLogoPath(ctx context.Context, db *gorm.DB, logoPath *string) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.CompetitionConfig{}).Where("id = 1").Updates(map[string]any{
		"logo_path":  logoPath,
		"updated_at": time.Now(),
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func SetLeaderboardFreeze(ctx context.Context, db *gorm.DB, frozen bool, frozenAt *time.Time) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	now := time.Now()
	if frozen && frozenAt == nil {
		frozenAt = &now
	}
	res := db.Model(&models.CompetitionConfig{}).Where("id = 1").Updates(map[string]any{
		"is_frozen":  frozen,
		"frozen_at":  frozenAt,
		"updated_at": now,
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func ClearLeaderboardFreeze(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.CompetitionConfig{}).Where("id = 1").Updates(map[string]any{
		"is_frozen":  false,
		"frozen_at":  nil,
		"updated_at": time.Now(),
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
