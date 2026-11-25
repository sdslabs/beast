package middleware

import (
	"github.com/gin-gonic/gin"
)

// AccountAuthorize ensures caller has an active session (email verification may be pending).
func AccountAuthorize(c *gin.Context) {
	// TODO: authenticate bearer session token
	c.Next()
}

// UserAuthorize ensures caller is authenticated, email-verified, and permitted to participate.
func UserAuthorize(c *gin.Context) {
	// TODO: verify user is logged in, email-verified, and not banned
	c.Next()
}

// TeamAuthorize ensures caller belongs to the team owning the requested resource.
func TeamAuthorize(c *gin.Context) {
	// TODO: verify user has an active team membership
	c.Next()
}

// WithoutTeamAuthorize ensures caller is a verified user who is not currently in any team.
func WithoutTeamAuthorize(c *gin.Context) {
	// TODO: verify user does not belong to any team
	c.Next()
}

// LeaderAuthorize ensures caller is the current captain/leader of the target team.
func LeaderAuthorize(c *gin.Context) {
	// TODO: verify user is leader of the team specified in path/context
	c.Next()
}

// AdminAuthorize ensures caller is authenticated and has administrative privileges.
func AdminAuthorize(c *gin.Context) {
	// TODO: verify caller has role == 'admin'
	c.Next()
}

// OperatorAuthorize ensures caller has operational/health probe access.
func OperatorAuthorize(c *gin.Context) {
	// TODO: verify operational token / localhost probe access
	c.Next()
}

// DocsAuthorize ensures caller is permitted to view API documentation.
func DocsAuthorize(c *gin.Context) {
	// TODO: verify docs permission if restricted
	c.Next()
}
