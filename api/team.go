package api

import (
	"fmt"
	"net/http"
	"sort"

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
	name := c.PostForm("name")

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
		Name:   name,
		Status: 0, // Active
		Score:  0,
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
		"Team":          &team, // Add the team directly here
	}

	// Call UpdateUser with both arguments
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
	// Extract the username to be removed from the request body
	var requestBody struct {
		UserName string `json:"user_name"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid request body.",
		})
		return
	}

	// Get the captain's username (captain is validated by middleware)
	username, err := coreUtils.GetUser(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized user",
		})
		return
	}

	// Fetch the captain's details
	captain, err := database.QueryFirstUserEntry("username", username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, HTTPPlainResp{
			Message: "Unauthorized user",
		})
		return
	}

	// Ensure the captain is associated with the correct team using GetTeamByName
	captainTeam, err := database.GetTeamByName(captain.Name)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "You are not a member of a team.",
		})
		return
	}

	// Fetch the user to be removed from the database by userName
	var user database.User
	if err := database.Db.Where("name = ?", requestBody.UserName).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User not found.",
		})
		return
	}

	// Ensure the user is part of the same team as the captain using GetTeamByName
	userTeam, err := database.GetTeamByName(user.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "User is not a member of any team.",
		})
		return
	}

	if userTeam.ID != captainTeam.ID {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "User is not a member of your team.",
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
