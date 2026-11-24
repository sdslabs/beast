package models

import (
	"time"

	"github.com/google/uuid"
)

type RuntimeType string

const (
	RuntimeTypeImage   RuntimeType = "image"
	RuntimeTypeCompose RuntimeType = "compose"
)

// ChallengeRuntimeConfigBase contains the common runtime configuration fields
// (runtime type and image reference or compose path).
type ChallengeRuntimeConfigBase struct {
	RuntimeType RuntimeType `db:"runtime_type" toml:"type,omitempty" json:"runtime_type" yaml:"type,omitempty"`
	ImageRef    *string     `db:"image_ref" toml:"image,omitempty" json:"image_ref,omitempty" yaml:"image,omitempty"` // prefer a digest-pinned registry reference
	ComposePath *string     `db:"compose_path" toml:"compose_path,omitempty" json:"-" yaml:"compose_path,omitempty"`  // trusted host path
}

type RuntimeConfigBase = ChallengeRuntimeConfigBase

// ChallengeRuntimeConfig maps to challenge_runtime_config (one row per
// non-offline challenge). Exactly one of ImageRef or ComposePath is set,
// matching RuntimeType. For workers and admins only.
type ChallengeRuntimeConfig struct {
	ChallengeID uuid.UUID `db:"challenge_id" json:"challenge_id"`
	ChallengeRuntimeConfigBase
}

// ChallengeInstanceConfig maps to challenge_instance_config (one row per
// instanced challenge).
type ChallengeInstanceConfig struct {
	ChallengeID uuid.UUID `db:"challenge_id" json:"challenge_id"`
	TTLMinutes  int       `db:"ttl_minutes" json:"ttl_minutes"`
	MaxRenewals int       `db:"max_renewals" json:"max_renewals"`
}

// DeploymentStatus mirrors the CHECK constraint on challenge_deployments.status.
type DeploymentStatus string

const (
	DeploymentStatusStarting DeploymentStatus = "starting"
	DeploymentStatusRunning  DeploymentStatus = "running"
	DeploymentStatusStopping DeploymentStatus = "stopping"
	DeploymentStatusStopped  DeploymentStatus = "stopped"
	DeploymentStatusExpired  DeploymentStatus = "expired"
	DeploymentStatusError    DeploymentStatus = "error"
)

// ChallengeDeployment maps to challenge_deployments. OwnerUserID is nil for a
// shared deployment. A row keeps its slot until CleanedUpAt is set, whatever
// Status says: an errored or expired deployment may still own containers.
type ChallengeDeployment struct {
	ID                uuid.UUID        `db:"id" json:"id"`
	ChallengeID       uuid.UUID        `db:"challenge_id" json:"challenge_id"`
	OwnerUserID       *uuid.UUID       `db:"owner_user_id" json:"owner_user_id,omitempty"` // nil = shared deployment
	NodeID            string           `db:"node_id" json:"node_id"`
	Status            DeploymentStatus `db:"status" json:"status"`
	RuntimeIdentifier *string          `db:"runtime_identifier" json:"runtime_identifier,omitempty"` // container ID or Compose project name
	InstanceDir       *string          `db:"instance_dir" json:"-"`                                  // worker path used for cleanup
	RenewalsUsed      int              `db:"renewals_used" json:"renewals_used"`
	CreatedAt         time.Time        `db:"created_at" json:"created_at"`
	StartedAt         *time.Time       `db:"started_at" json:"started_at,omitempty"`
	StoppedAt         *time.Time       `db:"stopped_at" json:"stopped_at,omitempty"`
	CleanedUpAt       *time.Time       `db:"cleaned_up_at" json:"cleaned_up_at,omitempty"`
	ExpiresAt         *time.Time       `db:"expires_at" json:"expires_at,omitempty"` // nil for shared deployments
	LastRenewedAt     *time.Time       `db:"last_renewed_at" json:"last_renewed_at,omitempty"`
}

type EndpointProtocol string

const (
	EndpointProtocolHTTP  EndpointProtocol = "http"
	EndpointProtocolHTTPS EndpointProtocol = "https"
	EndpointProtocolTCP   EndpointProtocol = "tcp"
	EndpointProtocolUDP   EndpointProtocol = "udp"
	EndpointProtocolSSH   EndpointProtocol = "ssh"
)

// DeploymentEndpoint maps to deployment_endpoints: a participant-facing route
// into a deployment, not every internal port the container exposes.
type DeploymentEndpoint struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	DeploymentID uuid.UUID        `db:"deployment_id" json:"deployment_id"`
	Name         string           `db:"name" json:"name"`
	Protocol     EndpointProtocol `db:"protocol" json:"protocol"`
	Host         string           `db:"host" json:"host"`
	Port         int              `db:"port" json:"port"`
	Path         *string          `db:"path" json:"path,omitempty"` // HTTP(S) endpoints only
}
