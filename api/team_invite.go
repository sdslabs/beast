package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/utils"
)

// generateInviteCode generates a random invite code
func generateInviteCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8] // 8 characters is enough
}

// generateInviteLinkHandler generates a team invite link
// @Summary Generate a team invite link
// @Tags team
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401,404 {object} HTTPErrorResp
// @Router /api/team/invite/generate [post]
func generateInviteLinkHandler(c *gin.Context) {
	// Get the captain's username from the Authorization header
	username, err := utils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Get the captain's user info
	captain, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Verify the user is a team captain
	if !captain.IsTeamCaptain || captain.TeamID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Only team captains can generate invite links"})
		return
	}

	// Generate invite code
	inviteCode := generateInviteCode()

	// Store the invite code in the team's record
	team := &database.Team{}
	if err := database.Db.First(team, captain.TeamID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Team not found"})
		return
	}

	team.InviteCode = inviteCode
	team.InviteExpiry = time.Now().Add(24 * time.Hour) // Expires in 24 hours
	if err := database.Db.Save(team).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to generate invite"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invite link generated successfully",
		"code":    inviteCode,
	})
}

// joinTeamHandler handles joining a team with an invite code
// @Summary Join a team using invite code
// @Tags team
// @Accept json
// @Produce json
// @Param code path string true "Invite code"
// @Success 200 {object} HTTPPlainResp
// @Failure 400,404 {object} HTTPErrorResp
// @Router /api/team/join/{code} [post]
func joinTeamHandler(c *gin.Context) {
	code := c.Param("code")

	// Get the user from the Authorization header
	username, err := utils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Check if user is already in a team
	if user.TeamID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "You are already in a team"})
		return
	}

	// Find team by invite code
	team := &database.Team{}
	if err := database.Db.Where("invite_code = ?", code).First(team).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Invalid invite code"})
		return
	}

	// Check if invite is expired
	if team.InviteExpiry.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invite code has expired"})
		return
	}

	// Get team members count
	var memberCount int64
	if err := database.Db.Model(&database.User{}).Where("team_id = ?", team.ID).Count(&memberCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to check team size"})
		return
	}

	// Check team size against config limit
	if memberCount >= int64(config.Cfg.CompetitionInfo.TeamSize) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Team has reached maximum size limit"})
		return
	}

	// Add user to team
	updateData := map[string]interface{}{
		"TeamID": team.ID,
	}
	if err := database.UpdateUser(&user, updateData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to join team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined team"})
}
