package instance

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// SpawnInstanceHandler handles POST /api/v1/challenges/:challenge_slug/instance/spawn
func SpawnInstanceHandler(c *gin.Context) {
	// TODO: implement spawning dynamic challenge container
	httputil.RespondNotImplemented(c, "SpawnInstanceHandler")
}

// GetInstanceHandler handles GET /api/v1/challenges/:challenge_slug/instance
func GetInstanceHandler(c *gin.Context) {
	// TODO: implement checking active instance status & endpoints
	httputil.RespondNotImplemented(c, "GetInstanceHandler")
}

// RenewInstanceHandler handles POST /api/v1/challenges/:challenge_slug/instance/renew
func RenewInstanceHandler(c *gin.Context) {
	// TODO: implement instance lifetime extension
	httputil.RespondNotImplemented(c, "RenewInstanceHandler")
}

// StopInstanceHandler handles POST /api/v1/challenges/:challenge_slug/instance/stop
func StopInstanceHandler(c *gin.Context) {
	// TODO: implement voluntary instance destruction
	httputil.RespondNotImplemented(c, "StopInstanceHandler")
}
