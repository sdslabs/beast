package dto

import "time"

type ManageChallengeResponse struct {
	Slug            string                `json:"slug"`
	Name            string                `json:"name"`
	Description     *string               `json:"description,omitempty"`
	Author          *string               `json:"author,omitempty"`
	Owner           *UserSummaryRef       `json:"owner,omitempty"`
	MaxPoints       int                   `json:"max_points"`
	MinPoints       int                   `json:"min_points"`
	DecaySolves     int                   `json:"decay_solves"`
	MaxAttempts     *int                  `json:"max_attempts,omitempty"`
	IsHidden        bool                  `json:"is_hidden"`
	ReleaseAt       *time.Time            `json:"release_at,omitempty"`
	DeploymentMode  string                `json:"deployment_mode"`
	Tags            []string              `json:"tags"`
	Prerequisites   []ChallengeSummaryRef `json:"prerequisites"`
	Maintainers     []UserSummaryRef      `json:"maintainers"`
	CurrentRevision *int                  `json:"current_revision,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type ManageUpdateChallengeRequest struct {
	Name              *string    `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	Description       *string    `json:"description,omitempty"`
	Author            *string    `json:"author,omitempty"`
	MaxPoints         *int       `json:"max_points,omitempty" binding:"omitempty,min=0"`
	MinPoints         *int       `json:"min_points,omitempty" binding:"omitempty,min=0"`
	DecaySolves       *int       `json:"decay_solves,omitempty" binding:"omitempty,min=1"`
	MaxAttempts       *int       `json:"max_attempts,omitempty"`
	IsHidden          *bool      `json:"is_hidden,omitempty"`
	ReleaseAt         *time.Time `json:"release_at,omitempty"`
	DeploymentMode    *string    `json:"deployment_mode,omitempty" binding:"omitempty,oneof=offline shared instanced"`
	Tags              []string   `json:"tags,omitempty"`
	PrerequisiteSlugs []string   `json:"prerequisite_slugs,omitempty"`
}

type RuntimeEndpointSpec struct {
	Name          string `json:"name" binding:"required"`
	Protocol      string `json:"protocol" binding:"required,oneof=http https tcp udp ssh"`
	ContainerPort int    `json:"container_port" binding:"required,min=1,max=65535"`
	ServiceName   string `json:"service_name,omitempty"` // required for compose
	Path          string `json:"path,omitempty"`         // HTTP(S) only
}

type RuntimeResourceLimits struct {
	CPUs      float64 `json:"cpus,omitempty"`
	MemoryMB  int     `json:"memory_mb,omitempty"`
	PidsLimit int     `json:"pids_limit,omitempty"`
}

type RuntimeReadinessCheck struct {
	Protocol        string `json:"protocol" binding:"omitempty,oneof=tcp http"`
	Port            int    `json:"port,omitempty"`
	Path            string `json:"path,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
	Retries         int    `json:"retries,omitempty"`
}

type ManageRuntimeConfigResponse struct {
	ChallengeSlug string                 `json:"challenge_slug"`
	RuntimeType   string                 `json:"runtime_type"` // image | compose
	ImageRef      *string                `json:"image_ref,omitempty"`
	ComposeConfig *string                `json:"compose_config,omitempty"` // sanitized YAML or revision ref
	Endpoints     []RuntimeEndpointSpec  `json:"endpoints"`
	Resources     *RuntimeResourceLimits `json:"resources,omitempty"`
	Environment   map[string]string      `json:"environment,omitempty"`
	Readiness     *RuntimeReadinessCheck `json:"readiness,omitempty"`
}

type ManageInstancePolicyResponse struct {
	ChallengeSlug      string `json:"challenge_slug"`
	TTLMinutes         int    `json:"ttl_minutes"`
	MaxRenewals        int    `json:"max_renewals"`
	MaxActivePerUser   int    `json:"max_active_per_user"`
	MaxActivePerTeam   int    `json:"max_active_per_team"`
	RenewalCooldownSec int    `json:"renewal_cooldown_sec"`
}

type ManageReplaceInstancePolicyRequest struct {
	TTLMinutes         int `json:"ttl_minutes" binding:"required,min=1"`
	MaxRenewals        int `json:"max_renewals" binding:"min=0"`
	MaxActivePerUser   int `json:"max_active_per_user" binding:"min=1"`
	MaxActivePerTeam   int `json:"max_active_per_team" binding:"min=1"`
	RenewalCooldownSec int `json:"renewal_cooldown_sec" binding:"min=0"`
}

type ManageHintResponse struct {
	Index         int        `json:"index"` // 1-based deterministic order within challenge
	ChallengeSlug string     `json:"challenge_slug"`
	Content       string     `json:"content"`
	CostPoints    int        `json:"cost_points"`
	ReleaseAt     *time.Time `json:"release_at,omitempty"`
	UnlockCount   int64      `json:"unlock_count"`
}

type ManageListHintsResponse struct {
	Items []ManageHintResponse `json:"items"`
}

type ManageCreateHintRequest struct {
	Content    string     `json:"content" binding:"required"`
	CostPoints int        `json:"cost_points" binding:"min=0"`
	ReleaseAt  *time.Time `json:"release_at,omitempty"`
}

type ManageUpdateHintRequest struct {
	Content    *string    `json:"content,omitempty"`
	CostPoints *int       `json:"cost_points,omitempty" binding:"omitempty,min=0"`
	ReleaseAt  *time.Time `json:"release_at,omitempty"`
}

type ManageCreateTagRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"` // URL-safe tag name
}

type ManageUpdateTagRequest struct {
	Name string `json:"name" binding:"required,min=1,max=64"`
}
