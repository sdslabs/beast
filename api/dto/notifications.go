package dto

import "time"

type ListNotificationsQuery struct {
	Unread   *bool      `form:"unread"`
	Type     string     `form:"type"` // announcement | challenge_update | first_blood | team
	IsPinned *bool      `form:"is_pinned"`
	Since    *time.Time `form:"since"` // RFC3339; omit to fetch all matching notifications
}

type NotificationResponse struct {
	Index         int        `json:"index"` // 1-based deterministic order (created_at ASC, id ASC)
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	Type          string     `json:"type"`
	ChallengeSlug *string    `json:"challenge_slug,omitempty"`
	TeamName      *string    `json:"team_name,omitempty"`
	Username      *string    `json:"username,omitempty"`
	IsPinned      bool       `json:"is_pinned"`
	IsRead        bool       `json:"is_read"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ListNotificationsResponse struct {
	Items []NotificationResponse `json:"items"`
}

type ReadAllNotificationsResponse struct {
	MarkedCount int64 `json:"marked_count"`
}

type NotificationSSEEvent struct {
	EventID      int                  `json:"event_id"`   // monotonic / matches notification index
	EventType    string               `json:"event_type"` // "notification.created" | "notification.updated" | "notification.deleted"
	Notification NotificationResponse `json:"notification"`
}
