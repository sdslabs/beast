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

// ChallengeBase defines the core challenge metadata shared across author manifests
// (e.g. challenge.toml / beast.toml) and the PostgreSQL database entity (challenges table).
type ChallengeBase struct {
	Name              string         `db:"name" toml:"name" json:"name" yaml:"name"`
	Slug              string         `db:"slug" toml:"slug" json:"slug" yaml:"slug"`
	Description       *string        `db:"description" toml:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Author            *string        `db:"author" toml:"author,omitempty" json:"author,omitempty" yaml:"author,omitempty"`
	MaxPoints         int            `db:"max_points" toml:"max_points" json:"max_points" yaml:"max_points"`
	MinPoints         int            `db:"min_points" toml:"min_points" json:"min_points" yaml:"min_points"`
	DecaySolves       int            `db:"decay_solves" toml:"decay_solves" json:"decay_solves" yaml:"decay_solves"`
	MaxAttempts       *int           `db:"max_attempts" toml:"max_attempts,omitempty" json:"max_attempts,omitempty" yaml:"max_attempts,omitempty"`
	FlagTemplate      *string        `db:"flag_template" toml:"flag_template,omitempty" json:"flag_template,omitempty" yaml:"flag_template,omitempty"`
	IsHidden          bool           `db:"is_hidden" toml:"is_hidden,omitempty" json:"is_hidden" yaml:"is_hidden,omitempty"`
	ReleaseAt         *time.Time     `db:"release_at" toml:"release_at,omitempty" json:"release_at,omitempty" yaml:"release_at,omitempty"`
	DeploymentMode    DeploymentMode `db:"deployment_mode" toml:"deployment_mode" json:"deployment_mode" yaml:"deployment_mode"`
	GeneratorPath     *string        `db:"generator_path" toml:"generator_path,omitempty" json:"-" yaml:"generator_path,omitempty"`
	GeneratorTimeoutS int            `db:"generator_timeout_s" toml:"generator_timeout_s,omitempty" json:"generator_timeout_s" yaml:"generator_timeout_s,omitempty"`
}

type Challenge struct {
	ID uuid.UUID `db:"id" json:"id"`
	ChallengeBase
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
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
