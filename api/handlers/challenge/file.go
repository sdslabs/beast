package challenge

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetChallengeFileHandler handles GET /api/v1/challenges/:challenge_slug/files/:file_id
func GetChallengeFileHandler(c *gin.Context) {
	slug := c.Param("challenge_slug")
	fileID := c.Param("file_id")
	_ = slug
	_ = fileID

	// TODO: implement challenge attachment file binary stream
	c.Data(http.StatusOK, "application/octet-stream", []byte{})
}
