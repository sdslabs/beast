package dto

import "time"

type SubmitFlagRequest struct {
	Flag string `json:"flag" binding:"required,max=512"`
}
type TeamSubmissionResponse struct {
	Challenge     ChallengeSummaryRef `json:"challenge"`
	SubmittedBy   UserSummaryRef      `json:"submitted_by"`
	IsCorrect     bool                `json:"is_correct"`
	PointsAwarded *int                `json:"points_awarded,omitempty"`
	IsFirstBlood  bool                `json:"is_first_blood"`
	CreatedAt     time.Time           `json:"created_at"`
}

type ListSolvesQuery struct {
	ChallengeSlug string     `form:"challenge"`
	TeamName      string     `form:"team"`
	Username      string     `form:"username"`
	Since         *time.Time `form:"since"` // RFC3339; omit to fetch all published solves
}

type PublicSolveItem struct {
	Challenge     ChallengeSummaryRef `json:"challenge"`
	Team          TeamSummaryRef      `json:"team"`
	SolvedBy      UserSummaryRef      `json:"solved_by"`
	PointsAwarded int                 `json:"points_awarded"`
	IsFirstBlood  bool                `json:"is_first_blood"`
	SolvedAt      time.Time           `json:"solved_at"`
}

type ListSolvesResponse struct {
	Items    []PublicSolveItem `json:"items"`
	IsFrozen bool              `json:"is_frozen"`
}

type LeaderboardEntry struct {
	Rank        int        `json:"rank"`
	TeamName    string     `json:"team_name"`
	Affiliation *string    `json:"affiliation,omitempty"`
	CountryCode *string    `json:"country_code,omitempty"`
	Score       int        `json:"score"`
	SolveCount  int        `json:"solve_count"`
	LastSolveAt *time.Time `json:"last_solve_at,omitempty"`
}

type LeaderboardResponse struct {
	Items    []LeaderboardEntry `json:"items"`
	IsFrozen bool               `json:"is_frozen"`
	FrozenAt *time.Time         `json:"frozen_at,omitempty"`
}

type LeaderboardHistoryQuery struct {
	Teams []string `form:"team"` // optional team names; defaults to top N teams
	Top   int      `form:"top,default=10" binding:"min=1,max=25"`
}

type ScorePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     int       `json:"score"`
}

type TeamScoreSeries struct {
	TeamName string       `json:"team_name"`
	Points   []ScorePoint `json:"points"`
}

type LeaderboardHistoryResponse struct {
	Series   []TeamScoreSeries `json:"series"`
	IsFrozen bool              `json:"is_frozen"`
	FrozenAt *time.Time        `json:"frozen_at,omitempty"`
}
