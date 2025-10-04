package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var (
	AUTHORIZED_KEYS_FILE = filepath.Join(core.BEAST_GLOBAL_DIR, core.DEFAULT_AUTH_KEYS_FILE)

	BEAST_GLOBAL_CONFIG  = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
	BEAST_EXAMPLE_CONFIG = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_EXAMPLE_DIR, core.BEAST_EX_CONFIG_FILE_NAME)
)

func initAuthorizedKeysFile() error {
	log.Infoln("Defaulting Authorized keys file:", AUTHORIZED_KEYS_FILE, "... can be changed later")
	return os.WriteFile(AUTHORIZED_KEYS_FILE, []byte("auth_keys"), 0666)
}

func downloadExampleBeastConfig() error {
	response, err := http.Get("https://raw.githubusercontent.com/sdslabs/beast/master/_examples/example.config.toml")
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("error while downloading: " + response.Status)
	}

	exampleConfig, err := os.Create(BEAST_EXAMPLE_CONFIG)
	if err != nil {
		return err
	}
	defer exampleConfig.Close()

	_, err = io.Copy(exampleConfig, response.Body)
	return err
}

func promptServerDetails(configuration *config.BeastConfig) {
	for utils.PromptBinary("Configure an available server?") {
		var server config.AvailableServer

		server.Host = utils.PromptString("Enter Host Name, leave empty for localhost")
		server.Username = utils.PromptString("Enter Username")
		server.SSHKeyPath = utils.PromptString("Enter SSH Key Path")
		server.Active = utils.PromptBinary("Enable this server?")

		configuration.AvailableServers[server.Username] = server
	}
}

func promptResourceLimits(configuration *config.BeastConfig) {
	configuration.Memory = utils.PromptInt64("Default CPU Share:", 1024)
	configuration.PidsLimit = utils.PromptInt64("Default PIDs Limit:", 100)
	configuration.CPUShares = utils.PromptInt64("Default Memory Limit:", 1024)
}

func promptRemoteRepository(configuration *config.BeastConfig) {
	for utils.PromptBinary("Configure a Remote Repository?") {
		var remote config.GitRemote

		remote.Url = utils.PromptString("Remote Repository URL, must be SSH based")
		remote.Active = utils.PromptBinary("Enable this repository?")
		remote.RemoteName = utils.PromptString("Remote Repository Name")
		remote.Branch = utils.PromptString("Remote Repository Branch")
		remote.Secret = utils.PromptSecret("Remote Repository SSH Key")

		err := remote.ValidateGitConfig()
		if err != nil {
			log.Errorln(err)
			log.Errorln("Skipping further repository configurations")
			break
		}

		configuration.GitRemotes = append(configuration.GitRemotes, remote)
	}
}

func promptCompetitionDetails(configuration *config.BeastConfig) {
	configuration.CompetitionInfo.Name = utils.PromptString("Enter Competition Name")
	configuration.CompetitionInfo.About = utils.PromptString("Enter Competition About Text")
	configuration.CompetitionInfo.Prizes = utils.PromptString("Enter Competition Prizes Text")
	configuration.CompetitionInfo.StartingTime = utils.PromptString("Enter Competition Start Time Text")
	configuration.CompetitionInfo.EndingTime = utils.PromptString("Enter Competition End Time Text")
	configuration.CompetitionInfo.LogoURL = utils.PromptString("Enter Competition Logo URL")
	configuration.CompetitionInfo.DynamicScore = utils.PromptBinary("Enable Dynamic Scoring")
}

func promptNotificationWebhooks(configuration *config.BeastConfig) {
	for utils.PromptBinary("Configure a Notification Webhook?") {
		var notification config.NotificationWebhook

		notification.ServiceName = utils.PromptSelection("Notification Service", core.NOTIFCIATION_SERVICES)
		notification.URL = utils.PromptString("Notification Service URL")
		notification.Active = utils.PromptBinary("Enable this webhook?")

		configuration.NotificationWebhooks = append(configuration.NotificationWebhooks, notification)
	}
}

func promptBeastConfiguration(configuration *config.BeastConfig) {
	promptServerDetails(configuration)
	promptResourceLimits(configuration)
	promptRemoteRepository(configuration)
	promptCompetitionDetails(configuration)
	promptNotificationWebhooks(configuration)
}

func tryCopyExampleConfig() error {
	var configuration config.BeastConfig

	if _, err := os.Stat(BEAST_EXAMPLE_CONFIG); os.IsNotExist(err) {
		if err = downloadExampleBeastConfig(); err != nil {
			return err
		}
	}

	if _, err := toml.Decode(BEAST_EXAMPLE_CONFIG, &configuration); err != nil {
		return err
	}

	promptBeastConfiguration(&configuration)

	file, err := os.Create(BEAST_GLOBAL_CONFIG)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err = encoder.Encode(configuration); err != nil {
		return err
	}

	return nil
}

func initBeastConfig() error {
	if err := initAuthorizedKeysFile(); err != nil {
		return err
	}

	if _, err := os.Stat(BEAST_GLOBAL_CONFIG); os.IsNotExist(err) {
		return tryCopyExampleConfig()
	}

	log.Infoln("Found global config file:", BEAST_GLOBAL_CONFIG)
	return nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Run interactive beast configuration setup",
	Long:  "Creates the Global Beast Config file while prompting the user interactively whenever needed.",

	Run: func(cmd *cobra.Command, args []string) {
		err := initBeastConfig()

		if err != nil {
			log.Errorln(err.Error())
			log.Errorln("Failed to create global beast config file... fix the above errors and try again")
			return
		}

		log.Infoln(fmt.Sprintf("Created global beast configuration file: %s", BEAST_GLOBAL_CONFIG))
	},
}
