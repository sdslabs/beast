package models

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// OTPType mirrors the CHECK constraint on auth_otps.type.
type OTPType string

const (
	OTPTypeEmailVerification OTPType = "email_verification"
	OTPTypePasswordReset     OTPType = "password_reset"
	OTPTypeLogin2FA          OTPType = "login_2fa"
)

// AuthOTP maps to the auth_otps table.
type AuthOTP struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	UserID      *uuid.UUID `db:"user_id" json:"user_id,omitempty"` // nil when verifying an email that has no account yet
	Email       string     `db:"email" json:"email"`
	OTPHash     string     `db:"otp_hash" json:"-"` // SHA-256 of the 6-digit code
	Type        OTPType    `db:"type" json:"type"`
	Attempts    int        `db:"attempts" json:"attempts"`
	MaxAttempts int        `db:"max_attempts" json:"max_attempts"`
	ExpiresAt   time.Time  `db:"expires_at" json:"expires_at"`
	ConsumedAt  *time.Time `db:"consumed_at" json:"consumed_at,omitempty"`
	IPAddress   netip.Addr `db:"ip_address" json:"ip_address"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}
