package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
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

// This updates challenge info in the respective challenge configuration
// @Summary Updates challenge info in the database, located at ~/.beast/beast.db.
// @Description Updates challenge info in the database, located at ~/.beast/beast.db.
// @Tags config
// @Accept  json
// @Produce json
// @Param name formData string true "Challenge Name"
// @Param hints formData string true "Challenge's hints"
// @Param desc formData string false "Challenge's description"
// @Param points formData string true "Challenge's points"
// @Param flag formData string true "Challenge's flag"
// @Param tags formData string true "Challenge's tags"
// @Param ports formData file false "Challenge's ports"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/config/challenge-info [post]
func updateChallengeInfoHandler(c *gin.Context) {
	name, exist := c.GetPostForm("name")
	if !exist {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Can't edit challenge without challenge name"),
		})
		return
	}

	configInfo := map[string]interface{}{
		"Name": c.PostForm("name"),
	}

	hints, exist := c.GetPostForm("hints")
	if exist {
		configInfo["Hints"] = hints
	}
	desc, exist := c.GetPostForm("desc")
	if exist {
		configInfo["Description"] = desc
	}
	points, exist := c.GetPostForm("points")
	if exist {
		configInfo["Points"] = points
	}
	flag, exist := c.GetPostForm("flag")
	if exist {
		configInfo["Flag"] = flag
	}

	log.Debug(fmt.Sprintf("Starting to update the challenge : %s", name))
	chall, err := database.QueryFirstChallengeEntry("name", name)
	if err != nil {
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: fmt.Sprintf("DB_ACCESS_ERROR : %s", err.Error()),
		})
		return
	}

	// Update challenge
	if e := database.UpdateChallenge(&chall, configInfo); e != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Error while updating challenge info: %s", err.Error()),
		})
		return
	}

	// Update ports
	ports, exist := c.GetPostForm("ports")
	if exist {
		if err := database.UpdatePorts(&chall); err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Error: %s", err.Error()),
			})
			return
		}

		portsArray := strings.Split(ports, " ")
		for _, port := range portsArray {
			u64, err := strconv.ParseUint(port, 10, 32)
			if err != nil {
				c.JSON(http.StatusBadRequest, HTTPPlainResp{
					Message: fmt.Sprintf("Error: %s", err.Error()),
				})
				return
			}

			newPort := database.Port{ChallengeID: chall.ID, PortNo: uint32(u64)}
			_, err = database.PortEntryGetOrCreate(&newPort)
			if err != nil {
				c.JSON(http.StatusBadRequest, HTTPPlainResp{
					Message: fmt.Sprintf("Error while updating challenge ports: %s", err.Error()),
				})
				return
			}
		}
	}

	// Update Tags
	tags, exist := c.GetPostForm("tags")
	if exist {
		tagsArray := strings.Split(tags, " ")
		tagArr := make([]*database.Tag, len(tagsArray))

		for index, tag := range tagsArray {
			tagArr[index] = &database.Tag{
				TagName: tag,
			}
		}

		if err = database.UpdateTags(tagArr, &chall); err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Error while updating tags: %s", err.Error()),
			})
		}
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Succesfully updated challenge: %s", name),
	})
	return
}
