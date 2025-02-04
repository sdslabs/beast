package database

import (
	"time"
)

type OTP struct {
	Email  string `gorm:"primaryKey"`
	Code   string
	Expiry time.Time
}

func CreateOTPEntry(otpEntry *OTP) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	var existingOTP OTP
	tx := Db.First(&existingOTP, "email = ?", otpEntry.Email)
	if tx.Error == nil {
		Db.Delete(&existingOTP)
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

func DeleteOTPEntry(email string) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Delete(&OTP{}, email).Error
}
