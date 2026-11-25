package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/database/models"
	"gorm.io/gorm"
)

func CreateOTP(otp *models.AuthOTP) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	if otp.ID == uuid.Nil {
		otp.ID = uuid.New()
	}
	if otp.CreatedAt.IsZero() {
		otp.CreatedAt = time.Now()
	}
	if otp.MaxAttempts <= 0 {
		otp.MaxAttempts = 5
	}
	if err := db.Create(otp).Error; err != nil {
		return MapError(err)
	}
	return nil
}

func GetLatestOTP(email string, otpType models.OTPType) (*models.AuthOTP, error) {
	db := database.GetDb()
	if db == nil {
		return nil, ErrNoDatabase
	}

	var otp models.AuthOTP
	if err := db.Where("email = ? AND type = ?", email, otpType).
		Order("created_at DESC").
		First(&otp).Error; err != nil {
		return nil, MapError(err)
	}
	return &otp, nil
}

func CountRecentOTPs(email string, otpType models.OTPType, since time.Time) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	var count int64
	if err := db.Model(&models.AuthOTP{}).
		Where("email = ? AND type = ? AND created_at >= ?", email, otpType, since).
		Count(&count).Error; err != nil {
		return 0, MapError(err)
	}
	return count, nil
}

func IncrementOTPAttempts(id uuid.UUID) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.AuthOTP{}).
		Where("id = ? AND consumed_at IS NULL", id).
		Update("attempts", gorm.Expr("attempts + 1"))
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func MarkOTPConsumed(id uuid.UUID, consumedAt time.Time) error {
	db := database.GetDb()
	if db == nil {
		return ErrNoDatabase
	}

	res := db.Model(&models.AuthOTP{}).
		Where("id = ?", id).
		Update("consumed_at", consumedAt)
	if res.Error != nil {
		return MapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func DeleteExpiredOTPs(before time.Time) (int64, error) {
	db := database.GetDb()
	if db == nil {
		return 0, ErrNoDatabase
	}

	res := db.Where("expires_at < ?", before).Delete(&models.AuthOTP{})
	if res.Error != nil {
		return 0, MapError(res.Error)
	}
	return res.RowsAffected, nil
}
