package dto

type FieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type ErrorResponse struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Fields    []FieldError `json:"fields,omitempty"`
	RequestID string       `json:"request_id"` // random non-UUID trace ID, e.g. ULID/NanoID
}

type GenericMessageResponse struct {
	Message string `json:"message"`
}

type ChallengeSummaryRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type TeamSummaryRef struct {
	TeamName string `json:"team_name"`
}

type UserSummaryRef struct {
	Username string `json:"username"`
}

type TagResponse struct {
	Name string `json:"name"`
}

type DeploymentEndpointResponse struct {
	Name     string  `json:"name"`
	Protocol string  `json:"protocol"` // http | https | tcp | udp | ssh
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Path     *string `json:"path,omitempty"`
}

type InstancePolicySummary struct {
	TTLMinutes  int `json:"ttl_minutes"`
	MaxRenewals int `json:"max_renewals"`
}

type ComponentHealth struct {
	Status  string `json:"status"` // ok | degraded | down
	Message string `json:"message,omitempty"`
}
