package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

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
		"Name": name,
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

	assets, exist := c.GetPostForm("assets")
	if exist {
		configInfo["Assets"] = assets
	}

	additionalLinks, exist := c.GetPostForm("additionalLinks")
	if exist {
		configInfo["AdditionalLinks"] = additionalLinks
	}

	log.Debug(fmt.Sprintf("Starting to update the challenge : %s", name))
	chall, err := database.QueryFirstChallengeEntry("name", name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, HTTPErrorResp{Error: "Challenge not found"})
			return
		}
		log.Errorf("DB_ACCESS_ERROR : %s", err.Error())
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: fmt.Sprintf("DB_ACCESS_ERROR : %s", err.Error()),
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
		portsArray := strings.Split(ports, ",")
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
		tagsArray := strings.Split(tags, core.DELIMITER)
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
