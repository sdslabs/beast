package utils

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This can be used for global error handling in the application
// Send error to the client

func SendError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}

func ValidatePoints(points int, err error) error {
	if err != nil {
		return err
	}
	if points < 0 {
		return errors.New("Points cannot be negative")
	}
	return nil
}

func FileRecieveError(c *gin.Context, err error) error {
	if err != nil {
		err = fmt.Errorf("Error while receiving file: %s", err)
		SendError(c, err)
		return err
	}
	return nil
}

func FileSaveError(c *gin.Context, err error) error {
	if err != nil {
		err = fmt.Errorf("Error while saving file: %s", err)
		SendError(c, err)
		return err
	}
	return nil
}
