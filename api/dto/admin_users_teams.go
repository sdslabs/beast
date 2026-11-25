// Administrator user and team management DTOs.
package dto

import "time"

// --- Admin users ---

type AdminListUsersQuery struct {
	Search     string `form:"search"`
	Role       string `form:"role"` // contestant | admin
	IsVerified *bool  `form:"is_verified"`
	IsBanned   *bool  `form:"is_banned"`
	IsHidden   *bool  `form:"is_hidden"`
	Format     string `form:"format"` // json | csv
}

type AdminUserResponse struct {
	Username   string          `json:"username"`
	Email      string          `json:"email"`
	Role       string          `json:"role"`
	IsVerified bool            `json:"is_verified"`
	IsBanned   bool            `json:"is_banned"`
	IsHidden   bool            `json:"is_hidden"`
	CanUseSSH  bool            `json:"can_use_ssh"`
	HasSSHKey  bool            `json:"has_ssh_key"`
	Team       *TeamSummaryRef `json:"team,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type AdminListUsersResponse struct {
	Items []AdminUserResponse `json:"items"`
}

type AdminCreateUserRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=64"`
	Email      string `json:"email" binding:"required,email,max=255"`
	Password   string `json:"password" binding:"required,min=8,max=128"`
	Role       string `json:"role" binding:"required,oneof=contestant admin"`
	IsVerified bool   `json:"is_verified"`
	CanUseSSH  bool   `json:"can_use_ssh"`
}

type AdminUpdateUserRequest struct {
	Role       *string `json:"role,omitempty" binding:"omitempty,oneof=contestant admin"`
	IsVerified *bool   `json:"is_verified,omitempty"`
	IsBanned   *bool   `json:"is_banned,omitempty"`
	IsHidden   *bool   `json:"is_hidden,omitempty"`
	CanUseSSH  *bool   `json:"can_use_ssh,omitempty"`
}

type AdminListTeamsQuery struct {
	Search   string `form:"search"`
	IsBanned *bool  `form:"is_banned"`
	IsHidden *bool  `form:"is_hidden"`
	Format   string `form:"format"` // json | csv
}

type AdminTeamResponse struct {
	Name            string              `json:"name"`
	CreatorUsername *string             `json:"creator_username,omitempty"`
	LeaderUsername  *string             `json:"leader_username,omitempty"`
	Affiliation     *string             `json:"affiliation,omitempty"`
	CountryCode     *string             `json:"country_code,omitempty"`
	IsBanned        bool                `json:"is_banned"`
	IsHidden        bool                `json:"is_hidden"`
	Score           int                 `json:"score"`
	Members         []TeamMemberSummary `json:"members"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type AdminListTeamsResponse struct {
	Items []AdminTeamResponse `json:"items"`
}

type AdminUpdateTeamRequest struct {
	Name           *string `json:"name,omitempty" binding:"omitempty,min=2,max=128"` // URL-safe characters recommended
	Affiliation    *string `json:"affiliation,omitempty" binding:"omitempty,max=128"`
	CountryCode    *string `json:"country_code,omitempty" binding:"omitempty,len=2,uppercase"`
	LeaderUsername *string `json:"leader_username,omitempty"`
	IsBanned       *bool   `json:"is_banned,omitempty"`
	IsHidden       *bool   `json:"is_hidden,omitempty"`
}
