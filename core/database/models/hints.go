package models

import (
	"time"

	"github.com/google/uuid"
)

// ChallengeHint maps to challenge_hints.
type ChallengeHint struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	ChallengeID uuid.UUID  `db:"challenge_id" json:"challenge_id"`
	Content     string     `db:"content" json:"content"` // Markdown; only return it to teams that unlocked the hint
	CostPoints  int        `db:"cost_points" json:"cost_points"`
	ReleaseAt   *time.Time `db:"release_at" json:"release_at,omitempty"` // nil = available immediately
}

// UserHint maps to user_hints: one row per team that unlocked a hint.
// Note: user_hints.hint_id uses ON DELETE RESTRICT to prohibit deleting hints once purchased.
type UserHint struct {
	ID         uuid.UUID `db:"id" json:"id"`
	HintID     uuid.UUID `db:"hint_id" json:"hint_id"`
	TeamID     uuid.UUID `db:"team_id" json:"team_id"`
	UserID     uuid.UUID `db:"user_id" json:"user_id"`         // teammate who bought it
	CostPoints int       `db:"cost_points" json:"cost_points"` // price at purchase time
	UnlockedAt time.Time `db:"unlocked_at" json:"unlocked_at"`
}
