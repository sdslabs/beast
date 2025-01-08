package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/utils"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
)

type ScoreboardEntry struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Score uint   `json:"score"`
}

type TeamMember struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	IsCaptain bool   `json:"is_captain"`
	Score     uint   `json:"score"`
}

// getTeamMembersHandler retrieves all members of a team and their scores
func getTeamMembersHandler(c *gin.Context) {
	// Get the team ID from the URL parameter
	teamID := c.Param("id")

	// Convert teamID to uint
	var teamIDUint uint
	if _, err := fmt.Sscanf(teamID, "%d", &teamIDUint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid team_id",
		})
		return
	}

	// Fetch team members and their details
	members, err := database.GetTeamMembers(teamIDUint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Error fetching team members",
		})
		return
	}

	// Prepare response with team members and their score
	var memberDetails []TeamMember
	for _, member := range members {
		memberDetails = append(memberDetails, TeamMember{
			ID:        member.ID,
			Username:  member.Username,
			IsCaptain: member.IsTeamCaptain,
			Score:     member.Score,
		})
	}

	// Return the team members and their details as JSON
	c.JSON(http.StatusOK, memberDetails)
}

// scoreboardHandler returns the sorted list of teams by score
func scoreboardHandler(c *gin.Context) {
	var scoreboard []ScoreboardEntry

	// Get all teams
	var teams []database.Team
	if err := database.Db.Find(&teams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("Error fetching teams: %v", err),
		})
		return
	}

	// Add teams to scoreboard
	for _, team := range teams {
		scoreboard = append(scoreboard, ScoreboardEntry{
			ID:    team.ID,
			Name:  team.Name,
			Score: team.Score,
		})
	}

	// Sort scoreboard by score in descending order
	sort.Slice(scoreboard, func(i, j int) bool {
		return scoreboard[i].Score > scoreboard[j].Score
	})

	// Return scoreboard as JSON
	c.JSON(http.StatusOK, scoreboard)
}

// CreateTeamHandler handles the creation of a new team
func createTeamHandler(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	
	// Validate team name
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Team name cannot be empty.",
		})
		return
	}

	// Get the username from the Authorization header
	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized access. Please provide a valid token.",
		})
		return
	}

	// Query the user by username
	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User not found.",
		})
		return
	}

	// Check if user is already in a team
	if user.TeamID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "You are already in a team.",
		})
		return
	}

	// Check if the team name already exists
	existingTeam, err := database.QueryTeamByName(name)
	if err == nil && existingTeam != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "A team with this name already exists.",
		})
		return
	}

	// Create the team
	team := database.Team{
		Name:         name,
		Status:       0, // Active
		Score:        0,
		InviteCode:   "",
		InviteExpiry: time.Time{}, // Zero time
	}

	if err := database.CreateTeamEntry(&team); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to create team.",
		})
		return
	}

	// Update user to be the team captain and include team data
	user.TeamID = team.ID
	user.IsTeamCaptain = true

	updateData := map[string]interface{}{
		"TeamID":        team.ID,
		"IsTeamCaptain": true,
		"Team":          &team,
	}

	if err := database.UpdateUser(&user, updateData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to update user with team info.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Team created successfully.",
		"team_id": team.ID,
	})
}

func teamCaptainAuthorize(c *gin.Context) {
	// Get the user from the authorization header (JWT or other)
	username, err := utils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized access. Please provide a valid token.",
		})
		c.Abort() // Stop further processing
		return
	}

	// Fetch user details from the database
	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User not found.",
		})
		c.Abort() // Stop further processing
		return
	}

	// Check if the user is a team captain
	if !user.IsTeamCaptain {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "You are not the team captain, and cannot perform this action.",
		})
		c.Abort() // Stop further processing
		return
	}

	// Proceed with the request if the user is the team captain
	c.Next()
}

func removeMemberHandler(c *gin.Context) {
	// Get username from form data
	username := strings.TrimSpace(c.PostForm("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Username is required.",
		})
		return
	}

	// Get the captain's username (captain is validated by middleware)
	captainUsername, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized user",
		})
		return
	}

	// Fetch the captain's details
	captain, err := database.QueryFirstUserEntry("username", captainUsername)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Unauthorized user",
		})
		return
	}

	// Verify captain is actually a team captain
	if !captain.IsTeamCaptain {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "Only team captains can remove members.",
		})
		return
	}

	// Fetch the user to be removed from the database by username
	user, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User not found.",
		})
		return
	}

	// Check if user is in captain's team
	if user.TeamID != captain.TeamID {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "User is not a member of your team.",
		})
		return
	}

	// Prevent removing self
	if user.ID == captain.ID {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Team captain cannot remove themselves.",
		})
		return
	}

	// Remove the user from the team
	err = database.RemoveUserFromTeam(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("Failed to remove user: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User removed from the team successfully.",
	})
}
