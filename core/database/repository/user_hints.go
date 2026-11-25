package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func UnlockUserHint(uh *models.UserHint) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if uh.ID == uuid.Nil {
		uh.ID = uuid.New()
	}
	if uh.UnlockedAt.IsZero() {
		uh.UnlockedAt = time.Now()
	}
	if err := db.Create(uh).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetUserHintByHintAndTeam(hintID, teamID uuid.UUID) (*models.UserHint, bool, error) {
	db := database.GetDb()
	if db == nil {
		return nil, false, ErrNoDatabase
	}

	var uh models.UserHint
	err := db.Where("hint_id = ? AND team_id = ?", hintID, teamID).First(&uh).Error
	if err != nil {
		mappedErr := MapError(err)
		if errors.Is(mappedErr, ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, mappedErr
	}
	return &uh, true, nil
}

func ListUserHintsByTeamID(teamID uuid.UUID) ([]models.UserHint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var list []models.UserHint
	if err := db.Where("team_id = ?", teamID).Order("unlocked_at ASC").Find(&list).Error; err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListUserHintsByChallengeID(challengeID uuid.UUID) ([]models.UserHint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var list []models.UserHint
	err := db.Table("user_hints").
		Select("user_hints.*").
		Joins("INNER JOIN challenge_hints ON challenge_hints.id = user_hints.hint_id").
		Where("challenge_hints.challenge_id = ?", challengeID).
		Order("user_hints.unlocked_at ASC").
		Find(&list).Error
	if err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func ListUnlockedHintsByTeamAndChallenge(challengeID, teamID uuid.UUID) ([]models.UserHint, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var list []models.UserHint
	err := db.Table("user_hints").
		Select("user_hints.*").
		Joins("INNER JOIN challenge_hints ON challenge_hints.id = user_hints.hint_id").
		Where("challenge_hints.challenge_id = ? AND user_hints.team_id = ?", challengeID, teamID).
		Order("user_hints.unlocked_at ASC").
		Find(&list).Error
	if err != nil {
		return nil, MapError(err)
	}
	return list, nil
}

func CountUserHintsByTeamID(teamID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.UserHint{}).Where("team_id = ?", teamID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountUserHintsByHintID(hintID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.UserHint{}).Where("hint_id = ?", hintID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func SumHintCostByTeamID(teamID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var total int64
	err := db.Model(&models.UserHint{}).
		Where("team_id = ?", teamID).
		Select("COALESCE(SUM(cost_points), 0)").
		Row().
		Scan(&total)
	if err != nil {
		return 0, MapError(err)
	}
	return total, nil
}

func ListHintCostTotalsPerTeam() (map[uuid.UUID]int64, error) {
	res := make(map[uuid.UUID]int64)
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	type teamCostRow struct {
		TeamID uuid.UUID `gorm:"column:team_id"`
		Total  int64     `gorm:"column:total"`
	}

	var rows []teamCostRow
	err := db.Model(&models.UserHint{}).
		Select("team_id, COALESCE(SUM(cost_points), 0) as total").
		Group("team_id").
		Scan(&rows).Error
	if err != nil {
		return nil, MapError(err)
	}

	for _, r := range rows {
		res[r.TeamID] = r.Total
	}
	return res, nil
}
