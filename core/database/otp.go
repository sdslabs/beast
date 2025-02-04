package database

import (
	"time"
)

type OTP struct {
	Email    string `gorm:"primaryKey"`
	Code     string
	Expiry   time.Time
	Verified bool
}

func CreateOTPEntry(otpEntry *OTP) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	var existingOTP OTP
	tx := Db.First(&existingOTP, "email = ?", otpEntry.Email)
	if tx.Error == nil {
		existingOTP.Code = otpEntry.Code
		existingOTP.Expiry = otpEntry.Expiry
		return Db.Save(&existingOTP).Error
	}

	return Db.Create(otpEntry).Error
}

func QueryOTPEntry(email string) (OTP, error) {
	var otpEntry OTP

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.First(&otpEntry, email)
	return otpEntry, tx.Error
}

func VerifyOTPEntry(email string) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Model(&OTP{}).Where("email = ?", email).Update("verified", true).Error
}
