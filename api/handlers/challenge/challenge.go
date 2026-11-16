package challenge

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// ListChallengesHandler handles GET /api/v1/challenges
func ListChallengesHandler(c *gin.Context) {
	// TODO: implement challenges catalog listing
	httputil.RespondNotImplemented(c, "ListChallengesHandler")
}

// GetChallengeHandler handles GET /api/v1/challenges/:challenge_slug
func GetChallengeHandler(c *gin.Context) {
	// TODO: implement detailed challenge inspection with hints catalog
	httputil.RespondNotImplemented(c, "GetChallengeHandler")
}
