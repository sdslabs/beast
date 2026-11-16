package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType lists the categories documented on notifications.type.
// The column has no CHECK constraint, so other values are possible in the DB.
type NotificationType string

const (
	NotificationTypeAnnouncement    NotificationType = "announcement"
	NotificationTypeChallengeUpdate NotificationType = "challenge_update"
	NotificationTypeFirstBlood      NotificationType = "first_blood"
	NotificationTypeTeam            NotificationType = "team"
)

// Notification maps to the notifications table.
type Notification struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	Title       string           `db:"title" json:"title"`
	Content     string           `db:"content" json:"content"` // Markdown
	Type        NotificationType `db:"type" json:"type"`
	ChallengeID *uuid.UUID       `db:"challenge_id" json:"challenge_id,omitempty"`
	TeamID      *uuid.UUID       `db:"team_id" json:"team_id,omitempty"` // nil = broadcast to all
	UserID      *uuid.UUID       `db:"user_id" json:"user_id,omitempty"` // nil = team or broadcast
	IsPinned    bool             `db:"is_pinned" json:"is_pinned"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
}

// UserNotificationRead maps to user_notification_reads.
type UserNotificationRead struct {
	UserID         uuid.UUID `db:"user_id" json:"user_id"`
	NotificationID uuid.UUID `db:"notification_id" json:"notification_id"`
	ReadAt         time.Time `db:"read_at" json:"read_at"`
}
