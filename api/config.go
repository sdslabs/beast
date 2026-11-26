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
	fileUtils "github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

// This reloads the beast global configuration
// @Summary Reloads any changes in beast global configuration, located at ~/.beast/config.toml.
// @Description Populates beast gobal config map by reparsing the config file $HOME/.beast/config.toml.
// @Tags config
// @Produce json
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Security ApiKeyAuth
// @Router /api/config/reload [patch]
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
// @Produce json
// @Param name formData string false "Competition Name"
// @Param about formData string false "Some information about competition"
// @Param prizes formData string false "Competitions Prizes for the winners"
// @Param starting_time formData string false "Competition's starting time"
// @Param ending_time formData string false "Competition's ending time"
// @Param timezone formData string false "Competition's timezone"
// @Param dynamic_score formData bool false "Enable dynamic scoring"
// @Param logo formData file false "Competition's logo"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPErrorResp
// @Accept x-www-form-urlencoded,mpfd
// @Security ApiKeyAuth
// @Router /api/config/competition-info [post]
func updateCompetitionInfoHandler(c *gin.Context) {
	configInfo, err := config.GetCompetitionInfo()
	if err != nil {
		log.Info("Unable to load previous config")
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: fmt.Sprintf("Unable to load previous config: %s", err),
		})
		return
	}

	if v, ok := c.GetPostForm("name"); ok {
		configInfo.Name = v
	}
	if v, ok := c.GetPostForm("about"); ok {
		configInfo.About = v
	}
	if v, ok := c.GetPostForm("prizes"); ok {
		configInfo.Prizes = v
	}
	if v, ok := c.GetPostForm("starting_time"); ok {
		configInfo.StartingTime = v
	}
	if v, ok := c.GetPostForm("ending_time"); ok {
		configInfo.EndingTime = v
	}
	if v, ok := c.GetPostForm("timezone"); ok {
		configInfo.TimeZone = v
	}
	if v, ok := c.GetPostForm("dynamic_score"); ok {
		parsed, parseErr := strconv.ParseBool(strings.TrimSpace(v))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Invalid dynamic_score value: %s", v),
			})
			return
		}
		configInfo.DynamicScore = parsed
	}

	logo, err := c.FormFile("logo")
	if err != nil {
		log.Info("No file recieved from the user")
	} else {
		logoDir := filepath.Join(
			core.BEAST_GLOBAL_DIR,
			core.BEAST_ASSETS_DIR,
			core.BEAST_LOGO_DIR,
		)
		if dirErr := fileUtils.CreateIfNotExistDir(logoDir); dirErr != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: fmt.Sprintf("Unable to create logo directory: %s", dirErr),
			})
			return
		}
		logoFilePath := filepath.Join(logoDir, filepath.Base(logo.Filename))

		// Delete previously uploaded logo file
		if configInfo.LogoURL != "" {
			if removeErr := os.Remove(configInfo.LogoURL); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Info("Unable to delete previous logo file")
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: fmt.Sprintf("Unable to delete previous logo file: %s", removeErr),
				})
				return
			}
		}

		// The file is received, save it
		if saveErr := c.SaveUploadedFile(logo, logoFilePath); saveErr != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: fmt.Sprintf("Unable to save file: %s", saveErr),
			})
			return
		}
		configInfo.LogoURL = logoFilePath
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
}

// This updates challenge info in the respective challenge configuration
// @Summary Updates challenge info in the database, located at ~/.beast/beast.db.
// @Description Updates challenge info in the database, located at ~/.beast/beast.db.
// @Tags config
// @Produce json
// @Param name formData string true "Challenge Name"
// @Param desc formData string false "Challenge description. The description field is also accepted."
// @Param points formData int false "Challenge's points"
// @Param flag formData string false "Challenge's flag"
// @Param tags formData string false "Challenge's tags"
// @Param ports formData string false "Comma-separated allocated host ports"
// @Param assets formData string false "Asset names separated by ::::"
// @Param additionalLinks formData string false "Links separated by ::::"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPErrorResp
// @Accept x-www-form-urlencoded,mpfd
// @Security ApiKeyAuth
// @Router /api/config/challenge-info [post]
func updateChallengeInfoHandler(c *gin.Context) {
	name, exist := c.GetPostForm("name")
	name = strings.TrimSpace(name)
	if !exist || name == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Can't edit challenge without challenge name",
		})
		return
	}

	configInfo := map[string]interface{}{
		"Name": name,
	}

	desc, exist := c.GetPostForm("desc")
	if !exist {
		desc, exist = c.GetPostForm("description")
	}
	if exist {
		configInfo["Description"] = desc
	}

	points, exist := c.GetPostForm("points")
	pointsUpdated := false
	if exist {
		parsedPoints, err := strconv.ParseUint(strings.TrimSpace(points), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPPlainResp{
				Message: fmt.Sprintf("Invalid points value: %s", points),
			})
			return
		}
		configInfo["Points"] = uint(parsedPoints)
		pointsUpdated = true
	}

	flag, exist := c.GetPostForm("flag")
	if exist {
		configInfo["Flag"] = flag
	}

	assets, exist := c.GetPostForm("assets")
	if exist {
		configInfo["Assets"] = strings.Join(fileUtils.SplitDelimited(assets, core.DELIMITER), core.DELIMITER)
	}

	additionalLinks, exist := c.GetPostForm("additionalLinks")
	if exist {
		configInfo["AdditionalLinks"] = strings.Join(fileUtils.SplitDelimited(additionalLinks, core.DELIMITER), core.DELIMITER)
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
	if chall.ID == 0 || chall.Name == "" {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: fmt.Sprintf("Challenge %s not found", name),
		})
		return
	}

	// Update challenge
	if e := database.UpdateChallenge(&chall, configInfo); e != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: fmt.Sprintf("Error while updating challenge info: %s", e.Error()),
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
		ports = strings.ReplaceAll(ports, " ", "")
		if ports != "" {
			portsArray := strings.Split(ports, ",")
			for _, port := range portsArray {
				if port == "" {
					continue
				}
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
	}

	// Update Tags
	tags, exist := c.GetPostForm("tags")
	if exist {
		tagsArray := fileUtils.SplitDelimited(tags, core.DELIMITER)
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
			return
		}
	}

	if pointsUpdated {
		invalidateLeaderboardCaches()
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: fmt.Sprintf("Succesfully updated challenge: %s", name),
	})
}
