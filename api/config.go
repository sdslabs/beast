package api

import (
	"fmt"
	"net/http"
	"os"
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

// This updates competition info in the beast global configuration
// @Summary Updates competition info in the beast global configuration, located at ~/.beast/config.toml.
// @Description Populates beast gobal config map by reparsing the config file $HOME/.beast/config.toml.
// @Tags config
// @Accept  json
// @Produce json
// @Param name formData string true "Competition Name"
// @Param about formData string true "Some information about competition"
// @Param prizes formData string false "Competitions Prizes for the winners"
// @Param starting_time formData string true "Competition's starting time"
// @Param ending_time formData string true "Competition's ending time"
// @Param timezone formData string true "Competition's timezone"
// @Param logo formData file false "Competition's logo"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/config/competition-info [post]
func updateCompetitionInfoHandler(c *gin.Context) {
	var logoFilePath string

	name := c.PostForm("name")
	about := c.PostForm("about")
	prizes := c.PostForm("prizes")
	starting_time := c.PostForm("starting_time")
	ending_time := c.PostForm("ending_time")
	timezone := c.PostForm("timezone")
	logo, err := c.FormFile("logo")

	// The file cannot be received.
	if err != nil {
		log.Info("No file recieved from the user")
	} else {
		logoFilePath = filepath.Join(
			core.BEAST_GLOBAL_DIR,
			core.BEAST_ASSETS_DIR,
			core.BEAST_LOGO_DIR,
			logo.Filename,
		)

		competitionInfo, err := config.GetCompetitionInfo()
		if err != nil {
			log.Info("Unable to load previous config")
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: fmt.Sprintf("Unable to load previous config: %s", err),
			})
			return
		}

		// Delete previously uploaded logo file
		if competitionInfo.LogoURL != "" {
			if err := os.Remove(competitionInfo.LogoURL); err != nil {
				log.Info("Unable to delete previous logo file")
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("Unable to delete previous logo file: %s", err),
				})
				return
			}
		}

		// The file is received, save it
		if err := c.SaveUploadedFile(logo, logoFilePath); err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
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
