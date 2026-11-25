package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateUser(ctx context.Context, db *gorm.DB, user *models.User) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func GetUserByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*models.User, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var user models.User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, MapError(err)
	}
	return &user, nil
}

func GetUserByEmail(ctx context.Context, db *gorm.DB, email string) (*models.User, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, MapError(err)
	}
	return &user, nil
}

func GetUserByUsername(ctx context.Context, db *gorm.DB, username string) (*models.User, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, MapError(err)
	}
	return &user, nil
}

func ListUsers(ctx context.Context, db *gorm.DB, offset, limit int) ([]models.User, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if limit <= 0 {
		return nil, ErrInvalidLimit
	}

	var users []models.User
	q := db.Model(&models.User{}).Order("created_at ASC").Limit(limit).Offset(offset)

	if err := q.Find(&users).Error; err != nil {
		return nil, MapError(err)
	}
	return users, nil
}

func ListUsersByIDs(ctx context.Context, db *gorm.DB, ids []uuid.UUID) ([]models.User, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	if len(ids) == 0 {
		return []models.User{}, nil
	}

	var users []models.User
	if err := db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, MapError(err)
	}
	return users, nil
}

func ListUsersByTeamID(ctx context.Context, db *gorm.DB, teamID uuid.UUID) ([]models.User, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var users []models.User
	if err := db.Where("team_id = ?", teamID).Order("created_at ASC").Find(&users).Error; err != nil {
		return nil, MapError(err)
	}
	return users, nil
}

func CountUsersByTeamID(ctx context.Context, db *gorm.DB, teamID uuid.UUID) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.User{}).Where("team_id = ?", teamID).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func CountUsers(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func UpdateUsername(ctx context.Context, db *gorm.DB, id uuid.UUID, username string) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"username":   username,
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

func UpdateUserPublicSSHKey(ctx context.Context, db *gorm.DB, id uuid.UUID, publicSSHKey *string) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
	}

	res := db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]any{
		"public_ssh_key": publicSSHKey,
		"updated_at":     time.Now(),
	})
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func UpdateUserPasswordHash(ctx context.Context, db *gorm.DB, id uuid.UUID, passwordHash string, saltUUID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func SetUserVerified(ctx context.Context, db *gorm.DB, id uuid.UUID, verified bool) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func UpdateUserRole(ctx context.Context, db *gorm.DB, id uuid.UUID, role string) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func SetUserTeamID(ctx context.Context, db *gorm.DB, userID uuid.UUID, teamID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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

func ClearUserTeamID(ctx context.Context, db *gorm.DB, userID uuid.UUID) error {
	if db == nil {
		return ErrNoDatabase
	}
	if ctx != nil {
		db = db.WithContext(ctx)
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
