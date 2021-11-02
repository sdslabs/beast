package main

import (
	"bytes"
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

	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("Generating Config Template")

		path, err := os.Getwd()
		if err != nil {
			log.Errorf("Error while finding directory's path :: %s : using empty string instead", err)
		}

		challName := filepath.Base(path)
		if challName == "." {
			challName = ""
		}

		var config config.BeastChallengeConfig
		config.PopulateDefaultValues()

		var configfile bytes.Buffer
		log.Debugf("Preparing Config template")
		configfileTemplate, err := template.New("configfile").Parse(tools.CHALLENGE_CONFIG_FILE_TEMPLATE)
		if err != nil {
			log.Errorf("Error while parsing configfile template :: %s", err)
			return
		}

		log.Debugf("Executing dockerfile template with challenge config")
		err = configfileTemplate.Execute(&configfile, config)
		if err != nil {
			log.Errorf("Error while executing configfile template :: %s", err)
			return
		}

		err = createFile()
		if err != nil {
			log.Errorf("Error while creating beast.toml :: %s", err)
			return
		}

		var file, erro = os.OpenFile(core.CHALLENGE_CONFIG_FILE_NAME, os.O_RDWR, 0644)
		if erro != nil {
			log.Fatal(erro)
		}

		_, err = file.WriteString(configfile.String())
		if err != nil {
			log.Errorf("Error while writing beast.toml :: %s", err)
		}

		defer file.Close()
		log.Debugf("beast.toml generated for the challenge")

		err = createPublicDir()
		if err != nil {
			log.Errorf("Error while creating public directory")
		}

	},
}

func createFile() error {
	// check if file exists
	_, err := os.Stat(core.CHALLENGE_CONFIG_FILE_NAME)

	// create file if not exists
	if os.IsNotExist(err) {

		file, err := os.Create(core.CHALLENGE_CONFIG_FILE_NAME)
		if err != nil {
			return err
		}
		defer file.Close()
		return nil

	} else if err != nil {
		return err
	}

	log.Errorf("%s already exists", core.CHALLENGE_CONFIG_FILE_NAME)

	return nil
}

func createPublicDir() error {
	// check if public directory exists
	_, err := os.Stat(core.PUBLIC)

	// create public directory if not exists
	if os.IsNotExist(err) {
		err = os.MkdirAll(core.PUBLIC, 0755)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Errorf("%s directory already exists", core.PUBLIC)
	return nil
}
