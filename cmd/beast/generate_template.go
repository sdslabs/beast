package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	tools "github.com/sdslabs/beastv4/templates"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var generateTemplateCmd = &cobra.Command{
	Use:   "new",
	Short: "generate config file",
	Long:  "generate basic challenge config and public directory ",

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("find challenge directory: %w", err)
		}

		challName := filepath.Base(path)
		if challName == "." {
			challName = ""
		}

		var challengeCfg config.BeastChallengeConfig
		challengeCfg.PopulateDefaultValues()
		if challName != "" {
			challengeCfg.Challenge.Metadata.Name = challName
		} else {
			challengeCfg.Challenge.Metadata.Name = "my-challenge"
		}
		challengeCfg.Author.Name = "Author"
		challengeCfg.Author.Email = "author@example.com"
		challengeCfg.Author.SSHKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleAuthorKeyPlaceholder author@example.com"
		challengeCfg.Challenge.Metadata.Type = core.BARE_CHALLENGE_TYPE_NAME
		challengeCfg.Challenge.Metadata.Flag = "beast{example_flag}"
		challengeCfg.Challenge.Metadata.Difficulty = "medium"
		challengeCfg.Challenge.Metadata.Description = "Describe the challenge here"
		challengeCfg.Challenge.Metadata.Points = 100
		challengeCfg.Challenge.Env.StaticContentDir = ""
		challengeCfg.Challenge.Env.BaseImage = core.DEFAULT_BASE_IMAGE
		challengeCfg.Challenge.Env.RunCmd = "sleep infinity"

		var configfile bytes.Buffer
		configfileTemplate, err := template.New("configfile").Parse(tools.CHALLENGE_CONFIG_FILE_TEMPLATE)
		if err != nil {
			return fmt.Errorf("parse challenge template: %w", err)
		}
		if err := configfileTemplate.Execute(&configfile, challengeCfg); err != nil {
			return fmt.Errorf("render challenge template: %w", err)
		}

		file, err := os.OpenFile(core.CHALLENGE_CONFIG_FILE_NAME, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("create challenge config: %w", err)
		}
		complete := false
		defer func() {
			if !complete {
				_ = os.Remove(core.CHALLENGE_CONFIG_FILE_NAME)
			}
		}()
		_, writeErr := configfile.WriteTo(file)
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("write challenge config: %w", writeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close challenge config: %w", closeErr)
		}
		if err := os.MkdirAll(core.PUBLIC, 0755); err != nil {
			return fmt.Errorf("create public directory: %w", err)
		}
		complete = true
		log.Debug("beast.toml generated for the challenge")
		return nil
	},
}
