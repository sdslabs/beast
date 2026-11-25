package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
)

func CreateUser(user *models.User) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	if user.SaltUUID == uuid.Nil {
		user.SaltUUID = uuid.New()
	}
	if user.Role == "" {
		user.Role = "contestant"
	}
	if err := db.Create(user).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetUserByID(id uuid.UUID) (*models.User, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var user models.User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, MapError(err)
	}
	return &user, nil
}

func GetUserByEmail(email string) (*models.User, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, MapError(err)
	}
	return &user, nil
}

func GetUserByUsername(username string) (*models.User, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, MapError(err)
	}
	return &user, nil
}

func ListUsers(offset, limit int) ([]models.User, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var users []models.User
	if limit == 0 {
		// TODO : error
	}
	q := db.Model(&models.User{}).Order("created_at ASC").Limit(limit).Offset(offset)

	if err := q.Find(&users).Error; err != nil {
		return nil, MapError(err)
	}
	return users, nil
}

func ListUsersByIDs(ids []uuid.UUID) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var users []models.User
	if err := db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, MapError(err)
	}
	return users, nil
}

func ListUsersByTeamID(teamID uuid.UUID) ([]models.User, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var users []models.User
	if err := db.Where("team_id = ?", teamID).Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, MapError(err)
	}
	return users, nil
}

func CountUsersByTeamID(teamID uuid.UUID) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.User{}).Where("team_id = ?", teamID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountUsers() (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func UpdateUserProfile(user models.User) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	user.UpdatedAt = time.Now()
	res := db.Model(&models.User{}).
		Where("id = ?", user.ID).
		Select("username", "email", "public_ssh_key", "updated_at").
		Updates(user)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func UpdateUserPasswordHash(id uuid.UUID, passwordHash string, saltUUID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"password_hash": passwordHash,
		"salt_uuid":     saltUUID,
		"updated_at":    time.Now(),
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func SetUserVerified(id uuid.UUID, verified bool) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"is_verified": verified,
		"updated_at":  time.Now(),
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func UpdateUserRole(id uuid.UUID, role string) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"role":       role,
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

func SetUserTeamID(userID uuid.UUID, teamID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"team_id":    teamID,
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

func ClearUserTeamID(userID uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"team_id":    nil,
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
