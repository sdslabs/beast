package models

func (User) TableName() string                  { return "users" }
func (Team) TableName() string                  { return "teams" }
func (AuthSession) TableName() string           { return "auth_sessions" }
func (AuthOTP) TableName() string               { return "auth_otps" }
func (Challenge) TableName() string             { return "challenges" }
func (Tag) TableName() string                   { return "tags" }
func (ChallengeTag) TableName() string          { return "challenge_tags" }
func (ChallengePrerequisite) TableName() string { return "challenge_prerequisites" }
func (ChallengeHint) TableName() string         { return "challenge_hints" }
func (UserHint) TableName() string              { return "user_hints" }
func (ChallengeFile) TableName() string         { return "challenge_files" }
func (ChallengeFlag) TableName() string         { return "challenge_flags" }
func (ChallengeRuntimeConfig) TableName() string {
	return "challenge_runtime_config"
}
func (ChallengeInstanceConfig) TableName() string {
	return "challenge_instance_config"
}
func (ChallengeDeployment) TableName() string  { return "challenge_deployments" }
func (DeploymentEndpoint) TableName() string   { return "deployment_endpoints" }
func (Submission) TableName() string           { return "submissions" }
func (Solve) TableName() string                { return "solves" }
func (Notification) TableName() string         { return "notifications" }
func (UserNotificationRead) TableName() string { return "user_notification_reads" }
func (CompetitionConfig) TableName() string    { return "competition_config" }
func (AuditEvent) TableName() string           { return "audit_events" }
