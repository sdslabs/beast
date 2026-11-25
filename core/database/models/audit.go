package models

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// AuditEvent maps to the audit_events table.
type AuditEvent struct {
	ID          int64          `db:"id" json:"id"`
	ActorUserID *uuid.UUID     `db:"actor_user_id" json:"actor_user_id,omitempty"`
	Action      string         `db:"action" json:"action"`
	TargetType  string         `db:"target_type" json:"target_type"`
	TargetID    string         `db:"target_id" json:"target_id"`
	Metadata    map[string]any `db:"metadata" json:"metadata,omitempty"`
	IPAddress   netip.Addr     `db:"ip_address" json:"ip_address"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
}
