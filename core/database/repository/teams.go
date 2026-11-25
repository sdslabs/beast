package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateTeam(team *models.Team) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func GetTeamByID(id uuid.UUID) (*models.Team, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var team models.Team
	if err := db.Where("id = ?", id).First(&team).Error; err != nil {
		return nil, MapError(err)
	}
	return &team, nil
}

func GetTeamByName(name string) (*models.Team, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var team models.Team
	if err := db.Where("name = ?", name).First(&team).Error; err != nil {
		return nil, MapError(err)
	}
	return &team, nil
}

func GetTeamByInviteToken(token string) (*models.Team, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var team models.Team
	if err := db.Where("invite_token = ?", token).First(&team).Error; err != nil {
		return nil, MapError(err)
	}
	return &team, nil
}

func GetTeamInviteToken(teamID uuid.UUID) (string, error) {
	db := database.GetDb()
	if db == nil {
		return "", ErrNoDatabase
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

func ListTeams(offset, limit int) ([]models.Team, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func ListTeamsByIDs(ids []uuid.UUID) ([]models.Team, error) {
	if len(ids) == 0 {
		return []models.Team{}, nil
	}
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var teams []models.Team
	if err := db.Where("id IN ?", ids).Find(&teams).Error; err != nil {
		return nil, MapError(err)
	}
	return teams, nil
}

func ListMemberCountsByTeam(teamIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	res := make(map[uuid.UUID]int64)
	if len(teamIDs) == 0 {
		return res, nil
	}
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
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

func CountTeams() (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.Team{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func UpdateTeamDetails(team models.Team) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func UpdateTeamLeader(teamID uuid.UUID, leaderID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func UpdateTeamInviteToken(teamID uuid.UUID, token *string) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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

func SetTeamBanned(teamID uuid.UUID, banned bool) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
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
