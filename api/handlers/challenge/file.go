package challenge

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// GetChallengeFileHandler handles GET /api/v1/challenges/:challenge_slug/files/:file_id
func GetChallengeFileHandler(c *gin.Context) {
	// TODO: implement challenge attachment file binary stream
	httputil.RespondNotImplemented(c, "GetChallengeFileHandler")
}
