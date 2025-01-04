package database

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Team struct {
	gorm.Model
	Name       string       `gorm:"unique;not null"`
	Members    []*User      `gorm:"foreignKey:TeamID"`
	Status     uint         `gorm:"not null;default:0"` // 0 for unbanned, 1 for banned
	Score      uint         `gorm:"default:0"`
	Challenges []*Challenge `gorm:"many2many:team_challenges;"` // Solved challenges
}

type TeamChallenges struct {
	gorm.Model
	TeamID      uint      `gorm:"not null"`
	ChallengeID uint      `gorm:"not null"`
	SolverID    uint      `gorm:"not null"` // ID of the team member who solved it
	Challenge   Challenge `gorm:"foreignKey:ChallengeID"`
	Solver      User      `gorm:"foreignKey:SolverID"`
}

// QueryTeamEntries queries all teams where the column matches the value
func QueryTeamEntries(key string, value string) ([]Team, error) {
	queryKey := fmt.Sprintf("%s = ?", key)
	var teams []Team

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(queryKey, value).Find(&teams)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return teams, tx.Error
}

// QueryFirstTeamEntry gets the first team matching the criteria
func QueryFirstTeamEntry(key string, value string) (Team, error) {
	queryKey := fmt.Sprintf("%s = ?", key)
	var team Team

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(queryKey, value).First(&team)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return team, fmt.Errorf("team not found")
	}

	return team, tx.Error
}

// CreateTeamEntry creates a new team
func CreateTeamEntry(team *Team) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()
	if err := tx.Create(team).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create team: %v", err)
	}

	return tx.Commit().Error
}

// UpdateTeam updates a team entry
func UpdateTeam(team *Team, m map[string]interface{}) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Model(team).Updates(m).Error
}

// GetTeamMembers gets all members of a team
func GetTeamMembers(teamID uint) ([]User, error) {
	var members []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("team_id = ?", teamID).Find(&members)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return members, tx.Error
}

// GetTeamRank gets the rank of a team based on score
func GetTeamRank(teamID uint, teamScore uint, updatedAt time.Time) (int64, error) {
	DBMux.Lock()
	defer DBMux.Unlock()

	var rank int64
	tx := Db.Model(&Team{}).
		Where("score > ? OR (score = ? AND updated_at < ?)", teamScore, teamScore, updatedAt).
		Count(&rank)

	return rank + 1, tx.Error
}

// AddUserToTeam adds a user to a team
func AddUserToTeam(userID uint, teamID uint, isCaption bool) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	// Check if user exists and isn't in a team
	var user User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("user not found")
	}

	if user.TeamID != 0 {
		tx.Rollback()
		return fmt.Errorf("user already in a team")
	}

	// Check if team exists
	var team Team
	if err := tx.First(&team, teamID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("team not found")
	}

	// Update user's team
	if err := tx.Model(&user).Updates(map[string]interface{}{
		"TeamID":        teamID,
		"IsTeamCaptain": isCaption,
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add user to team")
	}

	return tx.Commit().Error
}

// RemoveUserFromTeam removes a user from their team
func RemoveUserFromTeam(userID uint) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	// Check if user exists and is in a team
	var user User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("user not found")
	}

	if user.TeamID == 0 {
		tx.Rollback()
		return fmt.Errorf("user not in a team")
	}

	// Remove user from team
	if err := tx.Model(&user).Updates(map[string]interface{}{
		"TeamID":        0,
		"IsTeamCaptain": false,
	}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove user from team")
	}

	return tx.Commit().Error
}

// GetTeamByID gets a team by its ID
func GetTeamByID(teamID uint) (Team, error) {
	var team Team

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.First(&team, teamID)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return team, fmt.Errorf("team not found")
	}

	return team, tx.Error
}

// CheckTeamSolvedChallenge checks if a team has already solved a challenge
func CheckTeamSolvedChallenge(teamID uint, challID uint) (bool, error) {
	var teamChallenges []TeamChallenges
	var count int64
	count = 0

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("team_id = ? AND challenge_id = ?", teamID, challID).Find(&teamChallenges).Count(&count)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return (count >= 1), tx.Error
}

// SaveTeamSolve records a team's solve of a challenge
func SaveTeamSolve(teamID uint, challengeID uint, solverID uint) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Begin()

	// Check if already solved
	solved, err := CheckTeamSolvedChallenge(teamID, challengeID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if solved {
		tx.Rollback()
		return fmt.Errorf("challenge already solved by team")
	}

	// Record the solve
	solve := TeamChallenges{
		TeamID:      teamID,
		ChallengeID: challengeID,
		SolverID:    solverID,
	}
	if err := tx.Create(&solve).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// GetTeamSolves gets all challenges solved by a team
func GetTeamSolves(teamID uint) ([]TeamChallenges, error) {
	var solves []TeamChallenges

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("team_id = ?", teamID).
		Preload("Challenge").
		Preload("Solver").
		Find(&solves)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return solves, tx.Error
}

// GetTeamSolveCount gets the number of challenges solved by a team
func GetTeamSolveCount(teamID uint) (int64, error) {
	DBMux.Lock()
	defer DBMux.Unlock()

	var count int64
	tx := Db.Model(&TeamChallenges{}).Where("team_id = ?", teamID).Count(&count)

	return count, tx.Error
}

func QueryTeamById(teamID uint) (Team, error) {
	var team Team
	if err := Db.First(&team, teamID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return Team{}, errors.New("team not found")
		}
		return Team{}, err
	}
	return team, nil
}

func QueryTeamByName(name string) (*Team, error) {
	var team Team
	if err := Db.Where("name = ?", name).First(&team).Error; err != nil {
		return nil, err
	}
	return &team, nil
}
