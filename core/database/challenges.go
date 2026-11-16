package models

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentMode string

const (
	DeploymentModeOffline   DeploymentMode = "offline"
	DeploymentModeShared    DeploymentMode = "shared"
	DeploymentModeInstanced DeploymentMode = "instanced"
)

type Challenge struct {
	ID                uuid.UUID      `db:"id" json:"id"`
	Name              string         `db:"name" json:"name"`
	Slug              string         `db:"slug" json:"slug"`
	Description       *string        `db:"description" json:"description,omitempty"`
	Author            *string        `db:"author" json:"author,omitempty"`
	MaxPoints         int            `db:"max_points" json:"max_points"`
	MinPoints         int            `db:"min_points" json:"min_points"`
	DecaySolves       int            `db:"decay_solves" json:"decay_solves"`
	MaxAttempts       *int           `db:"max_attempts" json:"max_attempts,omitempty"` // per team; nil = unlimited
	FlagTemplate      *string        `db:"flag_template" json:"flag_template,omitempty"`
	IsHidden          bool           `db:"is_hidden" json:"is_hidden"`
	ReleaseAt         *time.Time     `db:"release_at" json:"release_at,omitempty"` // nil = available immediately
	DeploymentMode    DeploymentMode `db:"deployment_mode" json:"deployment_mode"`
	GeneratorPath     *string        `db:"generator_path" json:"-"`
	GeneratorTimeoutS int            `db:"generator_timeout_s" json:"generator_timeout_s"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}
type ChallengePrerequisite struct {
	ChallengeID uuid.UUID `db:"challenge_id" json:"challenge_id"`
	RequiresID  uuid.UUID `db:"requires_id" json:"requires_id"`
}
type ChallengeFile struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	ChallengeID uuid.UUID  `db:"challenge_id" json:"challenge_id"`
	TeamID      *uuid.UUID `db:"team_id" json:"team_id,omitempty"` // nil = public file shared by all teams
	DisplayName string     `db:"display_name" json:"display_name"`
	StoragePath string     `db:"storage_path" json:"-"` // server-side location; never sent to clients
	SHA256      string     `db:"sha256" json:"sha256"`
	SizeBytes   int64      `db:"size_bytes" json:"size_bytes"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

type ChallengeFlag struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	ChallengeID   uuid.UUID  `db:"challenge_id" json:"challenge_id"`
	TeamID        *uuid.UUID `db:"team_id" json:"team_id,omitempty"` // nil = static flag shared by all teams
	FlagHash      string     `db:"flag_hash" json:"-"`
	CaseSensitive bool       `db:"case_sensitive" json:"case_sensitive"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}
