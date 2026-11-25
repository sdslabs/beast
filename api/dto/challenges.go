package dto

import "time"

type ListChallengesQuery struct {
	Tag            string `form:"tag"`
	Search         string `form:"search"`
	DeploymentMode string `form:"deployment_mode"` // offline | shared | instanced
	Solved         *bool  `form:"solved"`
}

type ChallengeListItemResponse struct {
	Slug           string     `json:"slug"`
	Name           string     `json:"name"`
	Points         int        `json:"points"`      // current dynamic value
	SolveCount     int        `json:"solve_count"` // respects freeze
	SolvedByTeam   bool       `json:"solved_by_team"`
	IsLocked       bool       `json:"is_locked"` // unmet prerequisites
	DeploymentMode string     `json:"deployment_mode"`
	Tags           []string   `json:"tags"`
	ReleaseAt      *time.Time `json:"release_at,omitempty"`
}

type ListChallengesResponse struct {
	Items []ChallengeListItemResponse `json:"items"`
}

type ChallengeTeamProgress struct {
	Solved       bool       `json:"solved"`
	SolvedAt     *time.Time `json:"solved_at,omitempty"`
	AttemptsUsed int        `json:"attempts_used"`
	MaxAttempts  *int       `json:"max_attempts,omitempty"`
}

type ActiveInstanceSummary struct {
	Status    string                       `json:"status"`
	Owner     UserSummaryRef               `json:"owner"`
	ExpiresAt *time.Time                   `json:"expires_at,omitempty"`
	Endpoints []DeploymentEndpointResponse `json:"endpoints,omitempty"` // only when running & unexpired
}

type ChallengeDetailResponse struct {
	Slug             string                       `json:"slug"`
	Name             string                       `json:"name"`
	Description      string                       `json:"description"`
	Author           *string                      `json:"author,omitempty"`
	Points           int                          `json:"points"`
	MaxPoints        int                          `json:"max_points"`
	MinPoints        int                          `json:"min_points"`
	SolveCount       int                          `json:"solve_count"` // respects freeze
	DeploymentMode   string                       `json:"deployment_mode"`
	Tags             []string                     `json:"tags"`
	Prerequisites    []ChallengeSummaryRef        `json:"prerequisites"`
	IsLocked         bool                         `json:"is_locked"`
	ReleaseAt        *time.Time                   `json:"release_at,omitempty"`
	InstancePolicy   *InstancePolicySummary       `json:"instance_policy,omitempty"`
	SharedEndpoints  []DeploymentEndpointResponse `json:"shared_endpoints,omitempty"` // only when shared deployment is running
	TeamProgress     *ChallengeTeamProgress       `json:"team_progress,omitempty"`
	ActiveInstance   *ActiveInstanceSummary       `json:"active_instance,omitempty"`
	ParticipantFiles []ParticipantFileResponse    `json:"participant_files"`
	Hints            []ParticipantHintResponse    `json:"hints"`
}

type ListTagsResponse struct {
	Items []TagResponse `json:"items"`
}

type ParticipantFileResponse struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type ParticipantFile = ParticipantFileResponse

type ParticipantHintResponse struct {
	Index      int        `json:"index"` // 1-based deterministic order within challenge
	CostPoints int        `json:"cost_points"`
	IsUnlocked bool       `json:"is_unlocked"`
	Content    *string    `json:"content,omitempty"` // omitted if locked
	UnlockedAt *time.Time `json:"unlocked_at,omitempty"`
	ReleaseAt  *time.Time `json:"release_at,omitempty"`
}

type UnlockedHintResponse struct {
	Index          int            `json:"index"`
	ChallengeSlug  string         `json:"challenge_slug"`
	Content        string         `json:"content"`
	CostPoints     int            `json:"cost_points"`
	UnlockedBy     UserSummaryRef `json:"unlocked_by"`
	UnlockedAt     time.Time      `json:"unlocked_at"`
	AlreadyExisted bool           `json:"already_existed"`
}

type ParticipantInstanceResponse struct {
	Challenge     ChallengeSummaryRef          `json:"challenge"`
	Owner         UserSummaryRef               `json:"owner"`
	Status        string                       `json:"status"` // starting | running | stopping | stopped | expired | error
	CleanupDone   bool                         `json:"cleanup_done"`
	RenewalsUsed  int                          `json:"renewals_used"`
	MaxRenewals   int                          `json:"max_renewals"`
	Endpoints     []DeploymentEndpointResponse `json:"endpoints,omitempty"` // populated ONLY when status == "running" && unexpired
	CreatedAt     time.Time                    `json:"created_at"`
	StartedAt     *time.Time                   `json:"started_at,omitempty"`
	ExpiresAt     *time.Time                   `json:"expires_at,omitempty"`
	LastRenewedAt *time.Time                   `json:"last_renewed_at,omitempty"`
	StoppedAt     *time.Time                   `json:"stopped_at,omitempty"`
}
