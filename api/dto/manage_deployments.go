package dto

import "time"

type ManageListDeploymentsQuery struct {
	ChallengeSlug string `form:"challenge"`
	OwnerUsername string `form:"owner"`
	NodeID        string `form:"node"`
	Status        string `form:"status"` // starting | running | stopping | stopped | expired | error
	Scope         string `form:"scope"`  // shared | instanced
}

type ManageDeploymentResponse struct {
	DeploymentID      string                       `json:"deployment_id"`
	Challenge         ChallengeSummaryRef          `json:"challenge"`
	Scope             string                       `json:"scope"` // "shared" | "instanced"
	Owner             *UserSummaryRef              `json:"owner,omitempty"`
	Team              *TeamSummaryRef              `json:"team,omitempty"`
	NodeID            string                       `json:"node_id"`
	Status            string                       `json:"status"`
	RuntimeIdentifier *string                      `json:"runtime_identifier,omitempty"`
	ErrorMessage      *string                      `json:"error_message,omitempty"`
	Endpoints         []DeploymentEndpointResponse `json:"endpoints"`
	RenewalsUsed      int                          `json:"renewals_used"`
	CreatedAt         time.Time                    `json:"created_at"`
	StartedAt         *time.Time                   `json:"started_at,omitempty"`
	StoppedAt         *time.Time                   `json:"stopped_at,omitempty"`
	CleanedUpAt       *time.Time                   `json:"cleaned_up_at,omitempty"`
	ExpiresAt         *time.Time                   `json:"expires_at,omitempty"`
	LastRenewedAt     *time.Time                   `json:"last_renewed_at,omitempty"`
}

type ManageListDeploymentsResponse struct {
	Items []ManageDeploymentResponse `json:"items"`
}

type ManageLaunchSharedDeploymentRequest struct {
	Revision *int    `json:"revision,omitempty"`
	NodeID   *string `json:"node_id,omitempty"`
}

type ManageReplaceDeploymentRequest struct {
	Revision *int    `json:"revision,omitempty"`
	NodeID   *string `json:"node_id,omitempty"`
}

type ManageDeploymentReplacementResponse struct {
	PreviousDeployment ManageDeploymentResponse `json:"previous_deployment"`
	NewDeployment      ManageDeploymentResponse `json:"new_deployment"`
}

type ManageDeploymentLogsQuery struct {
	Service string `form:"service"` // Compose service selector
	Follow  bool   `form:"follow"`
	Tail    int    `form:"tail,default=500" binding:"min=1,max=5000"`
}
