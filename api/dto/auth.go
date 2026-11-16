package dto

import (
	"time"
)

type RegisterRequest struct {
	Username     string  `json:"username" binding:"required,min=3,max=64"`
	Email        string  `json:"email" binding:"required,email,max=255"`
	Password     string  `json:"password" binding:"required,min=8,max=128"`
	PublicSSHKey *string `json:"public_ssh_key,omitempty"`
}

type AuthSessionResponse struct {
	AccessToken      string     `json:"access_token"`
	RefreshToken     string     `json:"refresh_token"`
	TokenType        string     `json:"token_type"`         // "Bearer"
	ExpiresInSeconds int64      `json:"expires_in_seconds"` // 900
	Scope            string     `json:"scope"`              // "account" (unverified) | "user" (verified)
	User             MeResponse `json:"user"`
}

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // username or email
	Password   string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type OTPDispatchResponse struct {
	ExpiresAt         time.Time `json:"expires_at"`
	ResendAvailableAt time.Time `json:"resend_available_at"`
}

type ConfirmOTPRequest struct {
	OTP string `json:"otp" binding:"required,len=6"`
}

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type PasswordResetConfirmRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required,len=6"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

type MeTeamMembership struct {
	TeamName string `json:"team_name"`
	IsLeader bool   `json:"is_leader"`
}

type MeResponse struct {
	Username     string            `json:"username"`
	Email        string            `json:"email"`
	Role         string            `json:"role"` // "contestant" | "admin"
	IsVerified   bool              `json:"is_verified"`
	CanUseSSH    bool              `json:"can_use_ssh"`
	PublicSSHKey *string           `json:"public_ssh_key,omitempty"`
	Team         *MeTeamMembership `json:"team,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type UpdateMeRequest struct {
	Username     *string `json:"username,omitempty" binding:"omitempty,min=3,max=64"`
	PublicSSHKey *string `json:"public_ssh_key,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}
