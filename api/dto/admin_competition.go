package dto

import (
	"net/netip"
	"time"
)

type AdminCompetitionResponse struct {
	Name                      string     `json:"name"`
	Description               string     `json:"description"`
	Rules                     string     `json:"rules"`
	ScoringPolicy             string     `json:"scoring_policy"` // dynamic | static
	RegistrationOpen          bool       `json:"registration_open"`
	StartsAt                  *time.Time `json:"starts_at,omitempty"`
	EndsAt                    *time.Time `json:"ends_at,omitempty"`
	IsFrozen                  bool       `json:"is_frozen"`
	FrozenAt                  *time.Time `json:"frozen_at,omitempty"`
	MaxTeamSize               int        `json:"max_team_size"`
	MaxInstancesPerUser       int        `json:"max_instances_per_user"`
	MaxInstancesPerTeam       int        `json:"max_instances_per_team"`
	SubmissionRateLimitPerMin int        `json:"submission_rate_limit_per_min"`
	LogoURL                   *string    `json:"logo_url,omitempty"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

type AdminUpdateCompetitionRequest struct {
	Name                      *string    `json:"name,omitempty"`
	Description               *string    `json:"description,omitempty"`
	Rules                     *string    `json:"rules,omitempty"`
	ScoringPolicy             *string    `json:"scoring_policy,omitempty" binding:"omitempty,oneof=dynamic static"`
	RegistrationOpen          *bool      `json:"registration_open,omitempty"`
	StartsAt                  *time.Time `json:"starts_at,omitempty"`
	EndsAt                    *time.Time `json:"ends_at,omitempty"`
	MaxTeamSize               *int       `json:"max_team_size,omitempty" binding:"omitempty,min=1"`
	MaxInstancesPerUser       *int       `json:"max_instances_per_user,omitempty" binding:"omitempty,min=1"`
	MaxInstancesPerTeam       *int       `json:"max_instances_per_team,omitempty" binding:"omitempty,min=1"`
	SubmissionRateLimitPerMin *int       `json:"submission_rate_limit_per_min,omitempty" binding:"omitempty,min=1"`
}

type AdminStatisticsResponse struct {
	TotalUsers         int64 `json:"total_users"`
	VerifiedUsers      int64 `json:"verified_users"`
	TotalTeams         int64 `json:"total_teams"`
	TotalChallenges    int64 `json:"total_challenges"`
	TotalSubmissions   int64 `json:"total_submissions"`
	CorrectSubmissions int64 `json:"correct_submissions"`
	TotalSolves        int64 `json:"total_solves"`
	ActiveDeployments  int64 `json:"active_deployments"`
	SharedDeployments  int64 `json:"shared_deployments"`
	InstancedActive    int64 `json:"instanced_active"`
}

type AdminListSubmissionsQuery struct {
	ChallengeSlug string     `form:"challenge"`
	TeamName      string     `form:"team"`
	Username      string     `form:"username"`
	IsCorrect     *bool      `form:"is_correct"`
	OnlySus       *bool      `form:"only_sus"` // filter suspected dynamic-flag leaks
	Since         *time.Time `form:"since"`    // RFC3339; omit to fetch all matching submissions
	Format        string     `form:"format"`   // json | csv
}

type AdminSubmissionDetailResponse struct {
	Challenge      ChallengeSummaryRef `json:"challenge"`
	Team           TeamSummaryRef      `json:"team"`
	User           UserSummaryRef      `json:"user"`
	SubmittedFlag  string              `json:"submitted_flag"`
	IsCorrect      bool                `json:"is_correct"`
	SusTeam        *TeamSummaryRef     `json:"sus_team,omitempty"` // populated if flag belonged to another team
	IPAddress      netip.Addr          `json:"ip_address"`
	UserAgent      *string             `json:"user_agent,omitempty"`
	TLSFingerprint *string             `json:"tls_fingerprint,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
}

type AdminListSubmissionsResponse struct {
	Items []AdminSubmissionDetailResponse `json:"items"`
}

type AdminListAuditEventsQuery struct {
	ActorUsername string     `form:"actor"`
	Action        string     `form:"action"`
	TargetType    string     `form:"target_type"`
	Since         *time.Time `form:"since"` // RFC3339; omit to fetch all matching events
}

type AdminAuditEventResponse struct {
	ID         int64          `json:"id"`
	Actor      UserSummaryRef `json:"actor"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"` // natural key (username, team_name, challenge_slug, node_id, etc.);
	Metadata   map[string]any `json:"metadata,omitempty"`
	IPAddress  netip.Addr     `json:"ip_address"`
	CreatedAt  time.Time      `json:"created_at"`
}

type AdminListAuditEventsResponse struct {
	Items []AdminAuditEventResponse `json:"items"`
}

type AdminLeaderboardQuery struct {
	IncludeHidden bool   `form:"include_hidden"`
	Format        string `form:"format"` // json | csv
}

type AdminFreezeLeaderboardRequest struct {
	FreezeAt *time.Time `json:"freeze_at,omitempty"` // defaults to current server timestamp
}

type LeaderboardFreezeResponse struct {
	IsFrozen bool      `json:"is_frozen"`
	FrozenAt time.Time `json:"frozen_at"`
}

type AdminCreateNotificationRequest struct {
	Title         string  `json:"title" binding:"required,max=255"`
	Content       string  `json:"content" binding:"required"`
	Type          string  `json:"type" binding:"required,oneof=announcement challenge_update first_blood team"`
	ChallengeSlug *string `json:"challenge_slug,omitempty"`
	TeamName      *string `json:"team_name,omitempty"` // nil = user/broadcast
	Username      *string `json:"username,omitempty"`  // nil = team/broadcast
	IsPinned      bool    `json:"is_pinned"`
}

type AdminUpdateNotificationRequest struct {
	Title    *string `json:"title,omitempty" binding:"omitempty,max=255"`
	Content  *string `json:"content,omitempty"`
	Type     *string `json:"type,omitempty" binding:"omitempty,oneof=announcement challenge_update first_blood team"`
	IsPinned *bool   `json:"is_pinned,omitempty"`
}
