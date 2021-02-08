package api

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"
)

// This reloads the beast global configuration
// @Summary Reloads any changes in beast global configuration, located at ~/.beast/config.toml.
// @Description Populates beast gobal config map by reparsing the config file $HOME/.beast/config.toml.
// @Tags config
// @Accept  json
// @Produce json
// @Param Authorization header string true "Bearer"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Router /api/config/reload/ [patch]
func reloadBeastConfig(c *gin.Context) {
	err := config.ReloadBeastConfig()
	if err != nil {
		log.Errorf("%s", err)
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "CONFIG RELOAD SUCCESSFUL",
	})
}

func updateCompetitionInfoHandler(c *gin.Context) {
	name := c.PostForm("name")
	about := c.PostForm("about")
	prizes := c.PostForm("prizes")
	starting_time := c.PostForm("starting_time")
	ending_time := c.PostForm("ending_time")
	timezone := c.PostForm("timezone")
	logo, err := c.FormFile("logo")

	logoFilePath := ""

	// The file cannot be received.
	if err != nil {
		log.Info("No file recieved from the user")
	} else {
		logoFilePath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, logo.Filename)

		// The file is received, save it
		if err := c.SaveUploadedFile(logo, logoFilePath); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: fmt.Sprintf("Unable to save file: %s", err),
			})
			return
		}
	}

	configInfo := config.CompetitionInfo{
		Name:         name,
		About:        about,
		Prizes:       prizes,
		StartingTime: starting_time,
		EndingTime:   ending_time,
		TimeZone:     timezone,
		LogoURL:      logoFilePath,
	}

	err = config.UpdateCompetitionInfo(&configInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "Competition information updated successfully",
	})
	return
}
