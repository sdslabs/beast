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
// @Router /api/config/reaload/ [patch]
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
