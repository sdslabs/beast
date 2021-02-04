package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	logo_url := c.PostForm("logo_url")

	configInfo := config.CompetitionInfo{
		Name:         name,
		About:        about,
		Prizes:       prizes,
		StartingTime: starting_time,
		EndingTime:   ending_time,
		TimeZone:     timezone,
		LogoURL:      logo_url,
	}

	err := config.UpdateCompetitionInfo(&configInfo)
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
