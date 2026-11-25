package models

import "github.com/google/uuid"

// Tag maps to the tags table.
type Tag struct {
	ID   uuid.UUID `db:"id" json:"id"`
	Name string    `db:"name" json:"name"`
}

// ChallengeTag maps to the challenge_tags join table.
type ChallengeTag struct {
	ChallengeID uuid.UUID `db:"challenge_id" json:"challenge_id"`
	TagID       uuid.UUID `db:"tag_id" json:"tag_id"`
}
