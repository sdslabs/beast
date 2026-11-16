package models

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Team maps to the teams table. Competition scoring is always team-based.
type Team struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	CreatorID   *uuid.UUID `db:"creator_id" json:"creator_id,omitempty"`
	LeaderID    *uuid.UUID `db:"leader_id" json:"leader_id,omitempty"`
	InviteToken *string    `db:"invite_token" json:"-"` // secret join token; expose it explicitly, never by accident
	Affiliation *string    `db:"affiliation" json:"affiliation,omitempty"`
	CountryCode *string    `db:"country_code" json:"country_code,omitempty"` // ISO 3166-1 alpha-2
	IsBanned    bool       `db:"is_banned" json:"is_banned"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// User maps to the users table.
type User struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	TeamID       *uuid.UUID `db:"team_id" json:"team_id,omitempty"`
	Username     string     `db:"username" json:"username"`
	Email        string     `db:"email" json:"email"`
	PasswordHash string     `db:"password_hash" json:"-"`
	SaltUUID     uuid.UUID  `db:"salt_uuid" json:"-"`
	Role         string     `db:"role" json:"role"`
	PublicSSHKey *string    `db:"public_ssh_key" json:"public_ssh_key,omitempty"`
	IsVerified   bool       `db:"is_verified" json:"is_verified"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

type AuthSession struct {
	ID               int64      `db:"id" json:"id"`
	UserID           uuid.UUID  `db:"user_id" json:"user_id"`
	RefreshTokenHash string     `db:"refresh_token_hash" json:"-"`
	UserAgent        *string    `db:"user_agent" json:"user_agent,omitempty"`
	IPAddress        netip.Addr `db:"ip_address" json:"ip_address"`
	IsRevoked        bool       `db:"is_revoked" json:"is_revoked"`
	ExpiresAt        time.Time  `db:"expires_at" json:"expires_at"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	LastUsedAt       *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
}
