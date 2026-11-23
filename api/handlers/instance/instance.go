package instance

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/dto"
	"github.com/sdslabs/beastv4/api/httputil"
)

// SpawnInstanceHandler handles POST /api/v1/challenges/:challenge_slug/instance/spawn
func SpawnInstanceHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement spawning dynamic challenge container
	var resp dto.ParticipantInstanceResponse
	httputil.RespondJSON(c, http.StatusAccepted, resp)
}

// GetInstanceHandler handles GET /api/v1/challenges/:challenge_slug/instance
func GetInstanceHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement checking active instance status & endpoints
	var resp dto.ParticipantInstanceResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// RenewInstanceHandler handles POST /api/v1/challenges/:challenge_slug/instance/renew
func RenewInstanceHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement instance lifetime extension
	var resp dto.ParticipantInstanceResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}

// StopInstanceHandler handles POST /api/v1/challenges/:challenge_slug/instance/stop
func StopInstanceHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	_ = slug

	// TODO: implement voluntary instance destruction
	var resp dto.ParticipantInstanceResponse
	httputil.RespondJSON(c, http.StatusOK, resp)
}
