package models

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// Submission maps to the submissions table: one row per flag attempt.
type Submission struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	ChallengeID    uuid.UUID  `db:"challenge_id" json:"challenge_id"`
	TeamID         uuid.UUID  `db:"team_id" json:"team_id"`
	UserID         uuid.UUID  `db:"user_id" json:"user_id"`
	SubmittedFlag  string     `db:"submitted_flag" json:"submitted_flag"`
	IsCorrect      bool       `db:"is_correct" json:"is_correct"`
	Sus            *uuid.UUID `db:"sus" json:"sus,omitempty"` // leak alert: team UUID if a dynamic flag was leaked
	IPAddress      netip.Addr `db:"ip_address" json:"ip_address"`
	UserAgent      *string    `db:"user_agent" json:"user_agent,omitempty"`
	TLSFingerprint *string    `db:"tls_fingerprint" json:"tls_fingerprint,omitempty"` // JA3/TLS handshake fingerprint
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// Solve maps to the solves table: one row per team per solved challenge.
type Solve struct {
	ID            uuid.UUID `db:"id" json:"id"`
	ChallengeID   uuid.UUID `db:"challenge_id" json:"challenge_id"`
	TeamID        uuid.UUID `db:"team_id" json:"team_id"`
	UserID        uuid.UUID `db:"user_id" json:"user_id"`
	PointsAwarded int       `db:"points_awarded" json:"points_awarded"` // dynamic-score snapshot at solve time
	IsFirstBlood  bool      `db:"is_first_blood" json:"is_first_blood"`
	SolvedAt      time.Time `db:"solved_at" json:"solved_at"`
}
