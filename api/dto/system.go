package dto

import "time"

type CompetitionPublicResponse struct {
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Rules            string     `json:"rules"`
	ScoringPolicy    string     `json:"scoring_policy"` // e.g. "dynamic" | "static"
	RegistrationOpen bool       `json:"registration_open"`
	StartsAt         *time.Time `json:"starts_at,omitempty"`
	EndsAt           *time.Time `json:"ends_at,omitempty"`
	IsFrozen         bool       `json:"is_frozen"`
	FrozenAt         *time.Time `json:"frozen_at,omitempty"`
	MaxTeamSize      int        `json:"max_team_size"`
	LogoURL          *string    `json:"logo_url,omitempty"`
}

type AdminSystemStatusResponse struct {
	Database    ComponentHealth `json:"database"`
	Redis       ComponentHealth `json:"redis"`
	Storage     ComponentHealth `json:"storage"`
	Workers     ComponentHealth `json:"workers"`
	ActiveNodes int             `json:"active_nodes"`
	TotalNodes  int             `json:"total_nodes"`
	QueueDepth  int64           `json:"queue_depth"`
	CheckedAt   time.Time       `json:"checked_at"`
}

type AdminConfigReloadResponse struct {
	Reloaded   bool      `json:"reloaded"`
	Message    string    `json:"message"`
	ReloadedAt time.Time `json:"reloaded_at"`
}

type HealthzResponse struct {
	Status string `json:"status"` // "ok"
}

type ReadyzResponse struct {
	Ready  bool                       `json:"ready"`
	Checks map[string]ComponentHealth `json:"checks"`
}
