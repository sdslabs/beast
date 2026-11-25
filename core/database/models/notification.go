package models

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

// TODO : currently these are some dummy notification types. need to change it later
const (
	NotificationTypeAnnouncement    NotificationType = "announcement"
	NotificationTypeChallengeUpdate NotificationType = "challenge_update"
	NotificationTypeFirstBlood      NotificationType = "first_blood"
	NotificationTypeTeam            NotificationType = "team"
)

func (n NotificationType) IsValid() bool {
	switch n {
	case NotificationTypeAnnouncement,
		NotificationTypeChallengeUpdate,
		NotificationTypeFirstBlood,
		NotificationTypeTeam:
		return true
	default:
		return false
	}
}

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

type UserNotificationRead struct {
	UserID         uuid.UUID `db:"user_id" json:"user_id"`
	NotificationID uuid.UUID `db:"notification_id" json:"notification_id"`
	ReadAt         time.Time `db:"read_at" json:"read_at"`
}
