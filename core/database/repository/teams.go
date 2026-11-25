package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateTeam(ctx context.Context, db *gorm.DB, team *models.Team) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if team.ID == uuid.Nil {
		team.ID = uuid.New()
	}
	now := time.Now()
	if team.CreatedAt.IsZero() {
		team.CreatedAt = now
	}
	if team.UpdatedAt.IsZero() {
		team.UpdatedAt = now
	}
	if err := db.Create(team).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetTeamByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.Team, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var team models.Team
	if err := db.Where("id = ?", id).First(&team).Error; err != nil {
		return nil, MapError(err)
	}
	return &team, nil
}

func GetTeamByName(ctx context.Context, db *gorm.DB, name string) (*models.Team, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var team models.Team
	if err := db.Where("name = ?", name).First(&team).Error; err != nil {
		return nil, MapError(err)
	}
	return &team, nil
}

func GetTeamByInviteToken(ctx context.Context, db *gorm.DB, token string) (*models.Team, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var team models.Team
	if err := db.Where("invite_token = ?", token).First(&team).Error; err != nil {
		return nil, MapError(err)
	}
	return &team, nil
}

func GetTeamInviteToken(ctx context.Context, db *gorm.DB, teamID uuid.UUID) (string, error) {
	if db == nil {
		return "", ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var team models.Team
	if err := db.Select("invite_token").Where("id = ?", teamID).First(&team).Error; err != nil {
		return "", MapError(err)
	}
	if team.InviteToken == nil {
		return "", nil
	}
	return *team.InviteToken, nil
}

func ListTeams(ctx context.Context, db *gorm.DB, offset, limit int) ([]models.Team, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var teams []models.Team
	q := db.Model(&models.Team{}).Order("created_at ASC").Limit(limit).Offset(offset)

	if err := q.Find(&teams).Error; err != nil {
		return nil, MapError(err)
	}
	return teams, nil
}

func ListTeamsByIDs(ctx context.Context, db *gorm.DB, ids []uuid.UUID) ([]models.Team, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if len(ids) == 0 {
		return []models.Team{}, nil
	}

	var teams []models.Team
	if err := db.Where("id IN ?", ids).Find(&teams).Error; err != nil {
		return nil, MapError(err)
	}
	return teams, nil
}

func ListMemberCountsByTeam(ctx context.Context, db *gorm.DB, teamIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := make(map[uuid.UUID]int64)
	if len(teamIDs) == 0 {
		return res, nil
	}

	type memberCountResult struct {
		TeamID uuid.UUID `gorm:"column:team_id"`
		Count  int64     `gorm:"column:count"`
	}

	var results []memberCountResult
	if err := db.Model(&models.User{}).
		Select("team_id, count(*) as count").
		Where("team_id IN ?", teamIDs).
		Group("team_id").
		Scan(&results).Error; err != nil {
		return nil, MapError(err)
	}

	for _, id := range teamIDs {
		res[id] = 0
	}
	for _, r := range results {
		res[r.TeamID] = r.Count
	}
	return res, nil
}

func CountTeams(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.Team{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func UpdateTeamDetails(ctx context.Context, db *gorm.DB, team models.Team) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	team.UpdatedAt = time.Now()
	res := db.Model(&models.Team{}).
		Where("id = ?", team.ID).
		Select("name", "affiliation", "country_code", "updated_at").
		Updates(team)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func UpdateTeamLeader(ctx context.Context, db *gorm.DB, teamID uuid.UUID, leaderID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.Team{}).Where("id = ?", teamID).Updates(map[string]any{
		"leader_id":  leaderID,
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

func UpdateTeamInviteToken(ctx context.Context, db *gorm.DB, teamID uuid.UUID, token *string) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.Team{}).Where("id = ?", teamID).Updates(map[string]any{
		"invite_token": token,
		"updated_at":   time.Now(),
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func SetTeamBanned(ctx context.Context, db *gorm.DB, teamID uuid.UUID, banned bool) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.Team{}).Where("id = ?", teamID).Updates(map[string]any{
		"is_banned":  banned,
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
