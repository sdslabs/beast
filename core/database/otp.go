package database

import (
	"crypto/subtle"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrOTPRateLimited = errors.New("OTP was requested too recently")
	ErrOTPInvalid     = errors.New("invalid OTP")
	ErrOTPExpired     = errors.New("OTP expired")
	ErrOTPAttempts    = errors.New("too many OTP attempts")
)

const (
	otpResendCooldown = time.Minute
	maxOTPAttempts    = 5
)

type OTP struct {
	Email    string `gorm:"primaryKey"`
	Expiry   time.Time
	Verified bool
	Purpose  string
	CodeHash []byte
	Attempts uint
	SentAt   time.Time
}

func ClearLegacyOTPSecrets() error {
	if !Db.Migrator().HasColumn(&OTP{}, "code") {
		return nil
	}
	return Db.Exec("UPDATE otps SET code = ''").Error
}

func IssueOTP(email, purpose string, codeHash []byte, now, expiry time.Time) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Transaction(func(tx *gorm.DB) error {
		var entry OTP
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", email).First(&entry).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && !entry.SentAt.IsZero() && now.Before(entry.SentAt.Add(otpResendCooldown)) {
			return ErrOTPRateLimited
		}

		entry.Email = email
		entry.Purpose = purpose
		entry.CodeHash = append([]byte(nil), codeHash...)
		entry.Expiry = expiry
		entry.SentAt = now
		entry.Attempts = 0
		entry.Verified = false
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&entry).Error
		}
		return tx.Save(&entry).Error
	})
}

func VerifyOTPCode(email, purpose string, codeHash []byte, now time.Time) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	var result error
	err := Db.Transaction(func(tx *gorm.DB) error {
		var entry OTP
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", email).First(&entry).Error; err != nil {
			return err
		}
		switch {
		case entry.Verified || entry.Purpose != purpose || len(entry.CodeHash) == 0:
			result = ErrOTPInvalid
		case now.After(entry.Expiry):
			result = ErrOTPExpired
		case entry.Attempts >= maxOTPAttempts:
			result = ErrOTPAttempts
		case subtle.ConstantTimeCompare(entry.CodeHash, codeHash) != 1:
			entry.Attempts++
			result = ErrOTPInvalid
			return tx.Model(&entry).Update("attempts", entry.Attempts).Error
		default:
			result = nil
			return tx.Model(&entry).Updates(map[string]interface{}{
				"verified":  true,
				"code_hash": []byte(nil),
			}).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	return result
}

func DeleteOTPEntry(email string) error {
	DBMux.Lock()
	defer DBMux.Unlock()
	return Db.Delete(&OTP{}, "email = ?", email).Error
}

func ConsumeVerifiedOTP(email, purpose string, now time.Time) error {
	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Transaction(func(tx *gorm.DB) error {
		var entry OTP
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", email).First(&entry).Error; err != nil {
			return err
		}
		if !entry.Verified || entry.Purpose != purpose || now.After(entry.Expiry) {
			return ErrOTPInvalid
		}
		return tx.Delete(&entry).Error
	})
}

func QueryOTPEntry(email string) (OTP, error) {
	var otpEntry OTP

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("email = ?", email).First(&otpEntry)

	return otpEntry, tx.Error
}
