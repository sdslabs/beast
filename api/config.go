package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var challengeTagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)

// This updates challenge info in the respective challenge configuration
// @Summary Updates challenge info in the database, located at ~/.beast/beast.db.
// @Description Updates challenge info in the database, located at ~/.beast/beast.db.
// @Tags config
// @Accept  json
// @Produce json
// @Param name formData string true "Challenge Name"
// @Param desc formData string false "Challenge's description"
// @Param points formData string false "Challenge's points"
// @Param flag formData string false "Challenge's flag"
// @Param tags formData string false "Challenge's tags"
// @Param ports formData string false "Challenge's ports"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPPlainResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /api/config/challenge-info [post]
func updateChallengeInfoHandler(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{Message: "Can't edit challenge without challenge name"})
		return
	}

	chall, err := database.QueryFirstChallengeEntry("name", name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, HTTPErrorResp{Error: "Challenge not found"})
			return
		}
		log.Errorf("database error while querying challenge %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Database error while querying challenge"})
		return
	}

	updates, ports, tags, err := parseChallengeUpdate(c, chall)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPPlainResp{Message: err.Error()})
		return
	}
	if err := database.UpdateChallengeConfiguration(chall.ID, updates, ports, tags); err != nil {
		log.Errorf("failed to update challenge %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, HTTPPlainResp{Message: "Failed to update challenge"})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{Message: fmt.Sprintf("Successfully updated challenge: %s", name)})
}

func parseChallengeUpdate(c *gin.Context, challenge database.Challenge) (map[string]interface{}, *[]uint32, *[]string, error) {
	updates := make(map[string]interface{})
	if description, exists := c.GetPostForm("desc"); exists {
		if len(description) > 64<<10 {
			return nil, nil, nil, errors.New("description exceeds 64 KiB")
		}
		updates["description"] = description
	}
	if pointsValue, exists := c.GetPostForm("points"); exists {
		points, err := strconv.ParseUint(strings.TrimSpace(pointsValue), 10, 32)
		if err != nil {
			return nil, nil, nil, errors.New("points must be an unsigned 32-bit integer")
		}
		if challenge.MaxPoints > 0 && uint(points) > challenge.MaxPoints || uint(points) < challenge.MinPoints {
			return nil, nil, nil, errors.New("points are outside the configured scoring range")
		}
		updates["points"] = uint(points)
	}
	if flag, exists := c.GetPostForm("flag"); exists {
		if (!challenge.DynamicFlag && flag == "") || len(flag) > 4096 {
			return nil, nil, nil, errors.New("flag must contain between 1 and 4096 bytes")
		}
		updates["flag"] = flag
	}
	if assets, exists := c.GetPostForm("assets"); exists {
		if err := validateAssetList(assets); err != nil {
			return nil, nil, nil, err
		}
		updates["assets"] = assets
	}
	if links, exists := c.GetPostForm("additionalLinks"); exists {
		if err := validateAdditionalLinks(links); err != nil {
			return nil, nil, nil, err
		}
		updates["additional_links"] = links
	}

	ports, err := parsePorts(c)
	if err != nil {
		return nil, nil, nil, err
	}
	tags, err := parseTags(c)
	if err != nil {
		return nil, nil, nil, err
	}
	return updates, ports, tags, nil
}

func parsePorts(c *gin.Context) (*[]uint32, error) {
	value, exists := c.GetPostForm("ports")
	if !exists {
		return nil, nil
	}
	ports := make([]uint32, 0)
	seen := make(map[uint32]struct{})
	for _, rawPort := range strings.Split(value, ",") {
		if strings.TrimSpace(rawPort) == "" {
			continue
		}
		port, err := strconv.ParseUint(strings.TrimSpace(rawPort), 10, 16)
		if err != nil || port == 0 {
			return nil, errors.New("ports must be comma-separated integers from 1 to 65535")
		}
		portNumber := uint32(port)
		if _, duplicate := seen[portNumber]; duplicate {
			return nil, fmt.Errorf("duplicate port %d", portNumber)
		}
		seen[portNumber] = struct{}{}
		ports = append(ports, portNumber)
		if len(ports) > 256 {
			return nil, errors.New("at most 256 ports are allowed")
		}
	}
	return &ports, nil
}

func parseTags(c *gin.Context) (*[]string, error) {
	value, exists := c.GetPostForm("tags")
	if !exists {
		return nil, nil
	}
	tags := make([]string, 0)
	seen := make(map[string]struct{})
	for _, rawTag := range strings.Split(value, core.DELIMITER) {
		tag := strings.TrimSpace(rawTag)
		if tag == "" {
			continue
		}
		if !challengeTagPattern.MatchString(tag) {
			return nil, fmt.Errorf("invalid tag %q", tag)
		}
		if _, duplicate := seen[tag]; duplicate {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
		if len(tags) > 32 {
			return nil, errors.New("at most 32 tags are allowed")
		}
	}
	return &tags, nil
}

func validateAssetList(value string) error {
	assets := strings.Split(value, core.DELIMITER)
	if len(assets) > 128 {
		return errors.New("at most 128 assets are allowed")
	}
	for _, asset := range assets {
		if asset == "" {
			continue
		}
		if len(asset) > 255 {
			return errors.New("asset path exceeds 255 bytes")
		}
		cleaned := filepath.Clean(asset)
		if filepath.IsAbs(asset) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || strings.Contains(asset, `\`) {
			return fmt.Errorf("invalid asset path %q", asset)
		}
	}
	return nil
}

func validateAdditionalLinks(value string) error {
	links := strings.Split(value, core.DELIMITER)
	if len(links) > 32 {
		return errors.New("at most 32 additional links are allowed")
	}
	for _, link := range links {
		if link == "" {
			continue
		}
		if len(link) > 2048 {
			return errors.New("additional link exceeds 2048 bytes")
		}
		parsed, err := url.ParseRequestURI(link)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
			return fmt.Errorf("invalid additional link %q", link)
		}
	}
	return nil
}
