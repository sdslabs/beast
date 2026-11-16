package dto

import "time"

type NodeCapacitySummary struct {
	ActiveDeployments int     `json:"active_deployments"`
	CPUsTotal         float64 `json:"cpus_total"`
	MemoryMBTotal     int     `json:"memory_mb_total"`
	MemoryMBUsed      int     `json:"memory_mb_used"`
}

type AdminNodeResponse struct {
	NodeID           string              `json:"node_id"` // configured string identifier
	DisplayName      string              `json:"display_name"`
	PublicHost       string              `json:"public_host"`
	SSHAddress       string              `json:"ssh_address"`
	SSHPort          int                 `json:"ssh_port"`
	PortRangeStart   int                 `json:"port_range_start"`
	PortRangeEnd     int                 `json:"port_range_end"`
	CredentialRef    string              `json:"credential_ref"`   // secret name, never secret value
	SchedulingState  string              `json:"scheduling_state"` // active | draining | disabled
	HealthState      string              `json:"health_state"`     // healthy | degraded | unreachable
	Capacity         NodeCapacitySummary `json:"capacity"`
	LastCheckedAt    *time.Time          `json:"last_checked_at,omitempty"`
	LastReconciledAt *time.Time          `json:"last_reconciled_at,omitempty"` // populated by automatic background reconciler or manual trigger
}

type AdminListNodesResponse struct {
	Items []AdminNodeResponse `json:"items"`
}

type AdminRegisterNodeRequest struct {
	NodeID          string `json:"node_id" binding:"required,min=1,max=64"`
	DisplayName     string `json:"display_name" binding:"required,max=128"`
	PublicHost      string `json:"public_host" binding:"required"`
	SSHAddress      string `json:"ssh_address" binding:"required"`
	SSHPort         int    `json:"ssh_port" binding:"min=1,max=65535"`
	PortRangeStart  int    `json:"port_range_start" binding:"min=1024,max=65535"`
	PortRangeEnd    int    `json:"port_range_end" binding:"min=1024,max=65535"`
	CredentialRef   string `json:"credential_ref" binding:"required"`
	SchedulingState string `json:"scheduling_state" binding:"required,oneof=active draining disabled"`
}

type AdminUpdateNodeRequest struct {
	DisplayName     *string `json:"display_name,omitempty"`
	PublicHost      *string `json:"public_host,omitempty"`
	SSHAddress      *string `json:"ssh_address,omitempty"`
	SSHPort         *int    `json:"ssh_port,omitempty" binding:"omitempty,min=1,max=65535"`
	PortRangeStart  *int    `json:"port_range_start,omitempty" binding:"omitempty,min=1024,max=65535"`
	PortRangeEnd    *int    `json:"port_range_end,omitempty" binding:"omitempty,min=1024,max=65535"`
	CredentialRef   *string `json:"credential_ref,omitempty"`
	SchedulingState *string `json:"scheduling_state,omitempty" binding:"omitempty,oneof=active draining disabled"`
}

type NodeProbeResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Latency int64  `json:"latency_ms"`
}

type AdminNodeCheckResponse struct {
	NodeID         string          `json:"node_id"`
	SSH            NodeProbeResult `json:"ssh"`
	DockerDaemon   NodeProbeResult `json:"docker_daemon"`
	Networks       NodeProbeResult `json:"networks"`
	ArtifactAccess NodeProbeResult `json:"artifact_access"`
	CheckedAt      time.Time       `json:"checked_at"`
}

type AdminNodeReconciliationResponse struct {
	NodeID               string    `json:"node_id"`
	Status               string    `json:"status"` // running | completed | failed
	RecoveredDeployments int       `json:"recovered_deployments"`
	CleanedOrphans       int       `json:"cleaned_orphans"`
	StartedAt            time.Time `json:"started_at"`
}
