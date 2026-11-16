package dto

import "time"

type ListUsersQuery struct {
	Search string `form:"search"`
}

type PublicUserListItem struct {
	Username string          `json:"username"`
	Team     *TeamSummaryRef `json:"team,omitempty"`
}

type ListUsersResponse struct {
	Items []PublicUserListItem `json:"items"`
}

type PublicSolveContribution struct {
	Challenge     ChallengeSummaryRef `json:"challenge"`
	PointsAwarded int                 `json:"points_awarded"`
	IsFirstBlood  bool                `json:"is_first_blood"`
	SolvedAt      time.Time           `json:"solved_at"`
}

type UserProfileResponse struct {
	Username  string                    `json:"username"`
	Team      *TeamSummaryRef           `json:"team,omitempty"`
	Solves    []PublicSolveContribution `json:"solves"`
	CreatedAt time.Time                 `json:"created_at"`
}

type ListTeamsQuery struct {
	Search  string `form:"search"`
	Country string `form:"country"`
}

type PublicTeamListItem struct {
	Name        string  `json:"name"`
	Affiliation *string `json:"affiliation,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	MemberCount int     `json:"member_count"`
	Score       int     `json:"score"`
	Rank        *int    `json:"rank,omitempty"`
}

type ListTeamsResponse struct {
	Items []PublicTeamListItem `json:"items"`
}

type TeamMemberSummary struct {
	Username string `json:"username"`
	IsLeader bool   `json:"is_leader"`
}

type TeamSolveSummary struct {
	Challenge     ChallengeSummaryRef `json:"challenge"`
	SolvedBy      UserSummaryRef      `json:"solved_by"`
	PointsAwarded int                 `json:"points_awarded"`
	IsFirstBlood  bool                `json:"is_first_blood"`
	SolvedAt      time.Time           `json:"solved_at"`
}

type TeamProfileResponse struct {
	Name        string              `json:"name"`
	Affiliation *string             `json:"affiliation,omitempty"`
	CountryCode *string             `json:"country_code,omitempty"`
	Score       int                 `json:"score"`
	Rank        *int                `json:"rank,omitempty"`
	Members     []TeamMemberSummary `json:"members"`
	Solves      []TeamSolveSummary  `json:"solves"`
	CreatedAt   time.Time           `json:"created_at"`
}

type CreateTeamRequest struct {
	Name        string  `json:"name" binding:"required,min=2,max=128"` // URL-safe characters recommended for {team_name} paths
	Affiliation *string `json:"affiliation,omitempty" binding:"omitempty,max=128"`
	CountryCode *string `json:"country_code,omitempty" binding:"omitempty,len=2,uppercase"`
}

type JoinTeamRequest struct {
	InviteToken string `json:"invite_token" binding:"required"`
}

type UpdateTeamRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=2,max=128"` // URL-safe characters recommended
	Affiliation *string `json:"affiliation,omitempty" binding:"omitempty,max=128"`
	CountryCode *string `json:"country_code,omitempty" binding:"omitempty,len=2,uppercase"`
}

type TransferLeadershipRequest struct {
	NewLeaderUsername string `json:"new_leader_username" binding:"required"`
}

type TeamInvitationResponse struct {
	TeamName    string `json:"team_name"`
	InviteToken string `json:"invite_token"`
}
