package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateSolve(ctx context.Context, db *gorm.DB, solve *models.Solve) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if solve.ID == uuid.Nil {
		solve.ID = uuid.New()
	}
	if solve.SolvedAt.IsZero() {
		solve.SolvedAt = time.Now()
	}
	if err := db.Create(solve).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetSolveByTeamAndChallenge(ctx context.Context, db *gorm.DB, teamID, challengeID uuid.UUID) (*models.Solve, bool, error) {
	if db == nil {
		return nil, false, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var solve models.Solve
	err := db.Where("team_id = ? AND challenge_id = ?", teamID, challengeID).First(&solve).Error
	if err != nil {
		mappedErr := MapError(err)
		if errors.Is(mappedErr, ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, mappedErr
	}
	return &solve, true, nil
}

func ListSolvesByChallengeID(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) ([]models.Solve, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var solves []models.Solve
	if err := db.Where("challenge_id = ?", challengeID).Order("solved_at ASC").Find(&solves).Error; err != nil {
		return nil, MapError(err)
	}
	return solves, nil
}

func GetFirstBloodByChallenge(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) (*models.Solve, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var solve models.Solve
	err := db.Where("challenge_id = ? AND is_first_blood = TRUE", challengeID).First(&solve).Error
	if err != nil {
		return nil, MapError(err)
	}
	return &solve, nil
}

func ListSolves(ctx context.Context, db *gorm.DB, offset, limit int) ([]models.Solve, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var solves []models.Solve
	q := db.Model(&models.Solve{}).Order("solved_at ASC").Limit(limit).Offset(offset)

	if err := q.Find(&solves).Error; err != nil {
		return nil, MapError(err)
	}
	return solves, nil
}

func ListSolvesBefore(ctx context.Context, db *gorm.DB, before time.Time, offset, limit int) ([]models.Solve, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var solves []models.Solve
	q := db.Model(&models.Solve{}).
		Where("solved_at <= ?", before).
		Order("solved_at ASC").
		Limit(limit).
		Offset(offset)

	if err := q.Find(&solves).Error; err != nil {
		return nil, MapError(err)
	}
	return solves, nil
}

func ListSolvesByTeamID(ctx context.Context, db *gorm.DB, teamID uuid.UUID) ([]models.Solve, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var solves []models.Solve
	if err := db.Where("team_id = ?", teamID).Order("solved_at ASC").Find(&solves).Error; err != nil {
		return nil, MapError(err)
	}
	return solves, nil
}

func ListSolvesByUserID(ctx context.Context, db *gorm.DB, userID uuid.UUID) ([]models.Solve, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var solves []models.Solve
	if err := db.Where("user_id = ?", userID).Order("solved_at ASC").Find(&solves).Error; err != nil {
		return nil, MapError(err)
	}
	return solves, nil
}

func ListSolvedChallengeIDsByTeam(ctx context.Context, db *gorm.DB, teamID uuid.UUID) ([]uuid.UUID, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var ids []uuid.UUID
	if err := db.Model(&models.Solve{}).
		Where("team_id = ?", teamID).
		Pluck("challenge_id", &ids).Error; err != nil {
		return nil, MapError(err)
	}
	return ids, nil
}

func CountSolves(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Solve{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSolvesByChallenge(ctx context.Context, db *gorm.DB, challengeID uuid.UUID) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Solve{}).Where("challenge_id = ?", challengeID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountSolvesByTeamID(ctx context.Context, db *gorm.DB, teamID uuid.UUID) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Solve{}).Where("team_id = ?", teamID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func ListSolveCountsByChallenge(ctx context.Context, db *gorm.DB) (map[uuid.UUID]int64, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := make(map[uuid.UUID]int64)

	type challSolveCount struct {
		ChallengeID uuid.UUID `gorm:"column:challenge_id"`
		Count       int64     `gorm:"column:count"`
	}

	var rows []challSolveCount
	err := db.Model(&models.Solve{}).
		Select("challenge_id, count(*) as count").
		Group("challenge_id").
		Scan(&rows).Error
	if err != nil {
		return nil, MapError(err)
	}

	for _, r := range rows {
		res[r.ChallengeID] = r.Count
	}
	return res, nil
}
