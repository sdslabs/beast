package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/sdslabs/beastv4/core"
	challengeConfig "github.com/sdslabs/beastv4/core/config"
	tools "github.com/sdslabs/beastv4/templates"
	"github.com/spf13/cobra"
)

var generateTemplateCmd = &cobra.Command{
	Use:   "new",
	Short: "Generate a challenge configuration and public directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		directory, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
		return generateChallengeTemplate(directory)
	},
}

func generateChallengeTemplate(directory string) error {
	configPath := filepath.Join(directory, core.CHALLENGE_CONFIG_FILE_NAME)
	publicPath := filepath.Join(directory, core.PUBLIC)
	for _, path := range []string{configPath, publicPath} {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing path: %s", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect output path %s: %w", path, err)
		}
	}

	var configuration challengeConfig.BeastChallengeConfig
	configuration.PopulateDefaultValues()
	parsed, err := template.New("configfile").Parse(tools.CHALLENGE_CONFIG_FILE_TEMPLATE)
	if err != nil {
		return fmt.Errorf("parse challenge template: %w", err)
	}
	var contents bytes.Buffer
	if err := parsed.Execute(&contents, configuration); err != nil {
		return fmt.Errorf("render challenge template: %w", err)
	}

	file, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create %s: %w", configPath, err)
	}
	keepConfig := false
	defer func() {
		_ = file.Close()
		if !keepConfig {
			_ = os.Remove(configPath)
		}
	}()
	if _, err := file.Write(contents.Bytes()); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", configPath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s: %w", configPath, err)
	}
	if err := os.Mkdir(publicPath, 0750); err != nil {
		return fmt.Errorf("create public directory: %w", err)
	}
	keepConfig = true
	return nil
}
