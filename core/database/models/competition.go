package models

import "time"

type CompetitionConfig struct {
	ID                        int        `db:"id" json:"id"`
	Title                     string     `db:"title" json:"title"`
	About                     *string    `db:"about" json:"about,omitempty"`
	Description               *string    `db:"description" json:"description,omitempty"`
	Rules                     *string    `db:"rules" json:"rules,omitempty"`
	ScoringPolicy             string     `db:"scoring_policy" json:"scoring_policy"`
	RegistrationOpen          bool       `db:"registration_open" json:"registration_open"`
	StartsAt                  *time.Time `db:"starts_at" json:"starts_at,omitempty"`
	EndsAt                    *time.Time `db:"ends_at" json:"ends_at,omitempty"`
	IsFrozen                  bool       `db:"is_frozen" json:"is_frozen"`
	FrozenAt                  *time.Time `db:"frozen_at" json:"frozen_at,omitempty"`
	MaxTeamSize               int        `db:"max_team_size" json:"max_team_size"`
	SubmissionRateLimitPerMin int        `db:"submission_rate_limit_per_min" json:"submission_rate_limit_per_min"`
	LogoPath                  *string    `db:"logo_path" json:"logo_path,omitempty"`
	CreatedAt                 time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time  `db:"updated_at" json:"updated_at"`
}
