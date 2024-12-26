package utils

import (
	"github.com/gin-gonic/gin"
)

func FileDownload(c *gin.Context, filename string, path string) string {
	file, err := c.FormFile(filename)
	if FileRecieveError(c, err) != nil {
		return ""
	}
	err = c.SaveUploadedFile(file, path)
	if FileSaveError(c, err) != nil {
		return ""
	}
	return file.Filename
}
