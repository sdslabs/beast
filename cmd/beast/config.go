package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	MINIMUM_MEMORY_LIMIT int64 = (1 << 23) /* a little over 6MB */

	BEAST_GLOBAL_CONFIG string = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
)

func generateConfigSecret(size int) (string, error) {
	secret := make([]byte, size)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func promptServerDetails(configuration *config.BeastConfig) {
	for utils.PromptBinary("Configure an available server?") {
		var server config.AvailableServer

		server.Host = utils.PromptString("Enter Host Name, leave empty for localhost")
		if server.Host == "" {
			server.Host = core.LOCALHOST
		}
		server.Username = utils.PromptString("Enter Username")
		server.SSHKeyPath = utils.PromptString("Enter SSH Key Path")
		server.Active = utils.PromptBinary("Enable this server?")

		configuration.AvailableServers[server.Host] = server
	}
}

func promptResourceLimits(configuration *config.BeastConfig) {
	configuration.CPUShares = utils.PromptInt64("Default CPU shares", core.DEFAULT_CPU_SHARE)
	configuration.CPUsLimit = utils.PromptFloat32("Default CPU Limit", core.DEFAULT_CPU_LIMIT)
	configuration.PidsLimit = utils.PromptInt64("Default PIDs Limit:", core.DEFAULT_PIDS_LIMIT)
	configuration.Memory = utils.PromptInt64("Default Memory Limit:", core.DEFAULT_MEMORY_LIMIT)

	if configuration.Memory < MINIMUM_MEMORY_LIMIT {
		log.Warnln(fmt.Sprintf("Memory limit provided is below 6MB... setting limit to %v bytes", MINIMUM_MEMORY_LIMIT))
		configuration.Memory = MINIMUM_MEMORY_LIMIT
	}
}

func promptRemoteRepository(configuration *config.BeastConfig) {
	for utils.PromptBinary("Configure a Remote Repository?") {
		var remote config.GitRemote

		remote.Url = utils.PromptString("Remote Repository URL, must be SSH based")
		remote.Active = utils.PromptBinary("Enable this repository?")
		remote.RemoteName = utils.PromptString("Remote Repository Name")
		remote.Branch = utils.PromptString("Remote Repository Branch")
		remote.Secret = utils.PromptString("Path to remote repository SSH private key")

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
	if !utils.PromptBinary("Configure competition metadata?") {
		return
	}
	configuration.CompetitionInfo.Name = utils.PromptString("Enter Competition Name")
	configuration.CompetitionInfo.About = utils.PromptString("Enter Competition About Text")
	configuration.CompetitionInfo.Prizes = utils.PromptString("Enter Competition Prizes Text")
	configuration.CompetitionInfo.LogoURL = utils.PromptString("Enter Competition Logo URL")
	configuration.CompetitionInfo.DynamicScore = utils.PromptBinary("Enable Dynamic Scoring")

	var startTime time.Time
	var endTime time.Time

	for {
		startTime = utils.PromptDateTime("Enter Competition Start Time")
		endTime = utils.PromptDateTime("Enter Competition End Time")

		if startTime.Before(endTime) {
			break
		}

		log.Errorln("Competition start time is after end time")
	}

	configuration.CompetitionInfo.StartingTime = utils.FormatTime(startTime)
	configuration.CompetitionInfo.EndingTime = utils.FormatTime(endTime)
}

func promptNotificationWebhooks(configuration *config.BeastConfig) {
	for utils.PromptBinary("Configure a Notification Webhook?") {
		var notification config.NotificationWebhook

		notification.ServiceName = utils.PromptSelection("Notification Service", core.NOTIFICATION_SERVICES)
		notification.URL = utils.PromptString("Notification Service URL")
		notification.Active = utils.PromptBinary("Enable this webhook?")

		configuration.NotificationWebhooks = append(configuration.NotificationWebhooks, notification)
	}
}

func promptCacheConnectionDetails(configuration *config.BeastConfig) error {
	configuration.RedisConf.User = utils.PromptString("Enter Redis User Name (this user will be created if does not exist)... leaving it empty will default it to beast")
	if configuration.RedisConf.User == "" {
		configuration.RedisConf.User = "beast"
	}

	configuration.RedisConf.Password = utils.PromptSecret(fmt.Sprintf("Enter Redis user %s password (leave empty to generate one)", configuration.RedisConf.User))
	if configuration.RedisConf.Password == "" {
		password, err := generateConfigSecret(32)
		if err != nil {
			return err
		}
		configuration.RedisConf.Password = password
		log.Info("Generated a Redis password in the private Beast configuration")
	}

	configuration.RedisConf.Host = utils.PromptString("Enter Redis Host Name, leave empty for localhost")
	if configuration.RedisConf.Host == "" {
		configuration.RedisConf.Host = core.LOCALHOST
	}

	configuration.RedisConf.Port = strconv.FormatInt(utils.PromptInt64("Enter Redis Port", 6379), 10)
	configuration.RedisConf.TLS = utils.PromptBinary("Use TLS for Redis?")
	if configuration.RedisConf.TLS {
		configuration.RedisConf.CAFile = utils.PromptString("Redis CA certificate path (empty uses system roots)")
		configuration.RedisConf.ServerName = utils.PromptString("Redis TLS server name (empty uses host)")
	}

	log.Infoln("Setting Redis DB to 0...")
	configuration.RedisConf.Db = 0
	return nil
}

func promptDatabaseConnectionDetails(configuration *config.BeastConfig) error {
	configuration.PsqlConf.User = utils.PromptString("Enter Postgres User Name (this user will be created if does not exist)... leaving it empty will default it to beast")
	if configuration.PsqlConf.User == "" {
		configuration.PsqlConf.User = "beast"
	}

	configuration.PsqlConf.Dbname = utils.PromptString("Enter Postgres Database Name... leaving it empty will default it to beast")
	if configuration.PsqlConf.Dbname == "" {
		configuration.PsqlConf.Dbname = "beast"
	}

	configuration.PsqlConf.Password = utils.PromptSecret(fmt.Sprintf("Enter Postgres user %s password (leave empty to generate one)", configuration.PsqlConf.User))
	if configuration.PsqlConf.Password == "" {
		password, err := generateConfigSecret(32)
		if err != nil {
			return err
		}
		configuration.PsqlConf.Password = password
		log.Info("Generated a PostgreSQL password in the private Beast configuration")
	}

	configuration.PsqlConf.Host = utils.PromptString("Enter Postgres Host Name, leave empty for localhost")
	if configuration.PsqlConf.Host == "" {
		configuration.PsqlConf.Host = core.LOCALHOST
	}
	configuration.PsqlConf.Port = strconv.FormatInt(utils.PromptInt64("Enter Postgres Port", 5432), 10)
	configuration.PsqlConf.SslMode = utils.PromptSelection("Enter Postgres SSL Mode", []string{
		"disable",
		"allow",
		"prefer",
		"require",
		"verify-ca",
		"verify-full",
	})
	if configuration.PsqlConf.SslMode == "verify-ca" || configuration.PsqlConf.SslMode == "verify-full" {
		configuration.PsqlConf.SSLRootCert = utils.PromptString("Postgres root CA certificate path")
	}
	return nil
}

func promptBeastConfiguration(configuration *config.BeastConfig) error {
	promptServerDetails(configuration)
	promptResourceLimits(configuration)
	promptRemoteRepository(configuration)
	promptCompetitionDetails(configuration)
	promptNotificationWebhooks(configuration)
	if err := promptCacheConnectionDetails(configuration); err != nil {
		return err
	}
	return promptDatabaseConnectionDetails(configuration)
}

func tryCopyExampleConfig() error {
	jwtSecret, err := generateConfigSecret(48)
	if err != nil {
		return err
	}
	configuration := config.BeastConfig{
		AllowedBaseImages: []string{"ubuntu:24.04", "debian:bookworm"},
		AvailableServers: map[string]config.AvailableServer{
			core.LOCALHOST: {Host: core.LOCALHOST, Active: true, PortRange: "10000:20000"},
		},
		JWTSecret:       jwtSecret,
		TickerFrequency: core.DEFAULT_TICKER_FREQUENCY,
		CPUShares:       core.DEFAULT_CPU_SHARE,
		CPUsLimit:       core.DEFAULT_CPU_LIMIT,
		Memory:          core.DEFAULT_MEMORY_LIMIT,
		PidsLimit:       core.DEFAULT_PIDS_LIMIT,
		PsqlConf:        config.PsqlConfig{User: "beast", Dbname: "beast", Host: core.LOCALHOST, Port: "5432", SslMode: "disable"},
		RedisConf:       config.RedisConfig{User: "beast", Host: core.LOCALHOST, Port: "6379"},
		InstanceConfig:  config.InstanceConfig{DefaultExpiration: 300, MaxExtension: 600, MaxInstancesPerUser: 3},
		ServerConfig: config.ServerConfig{
			TLSCertFile: filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR, "tls.crt"),
			TLSKeyFile:  filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR, "tls.key"),
		},
	}
	if err := promptBeastConfiguration(&configuration); err != nil {
		return err
	}
	if err := ensureLocalTLSCertificate(); err != nil {
		return err
	}
	if err := configuration.ValidateConfig(); err != nil {
		return fmt.Errorf("validate generated configuration: %w", err)
	}

	var encoded bytes.Buffer
	if err := toml.NewEncoder(&encoded).Encode(configuration); err != nil {
		return err
	}
	file, err := os.OpenFile(BEAST_GLOBAL_CONFIG, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(encoded.Bytes()); err != nil {
		file.Close()
		_ = os.Remove(BEAST_GLOBAL_CONFIG)
		return err
	}
	return file.Close()
}

func initBeastConfig() error {
	if _, err := os.Lstat(BEAST_GLOBAL_CONFIG); os.IsNotExist(err) {
		if err := initDirectories(); err != nil {
			return err
		}
		return tryCopyExampleConfig()
	} else if err != nil {
		return err
	}

	log.Infoln("Found global config file:", BEAST_GLOBAL_CONFIG)
	return nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Run interactive beast configuration setup",
	Long:  "Creates the global Beast config file while prompting the user interactively whenever needed.",

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initBeastConfig(); err != nil {
			return fmt.Errorf("initialize Beast configuration: %w", err)
		}

		log.Infoln(fmt.Sprintf("Beast global config file initiliazed at %s", BEAST_GLOBAL_CONFIG))
		return nil
	},
}
