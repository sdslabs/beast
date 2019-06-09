package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// This is the global beast configuration structure
// AuthorizedKeysFile: authorized_keys file to give challenge authors access
// 	to the deployed containers. So each time a deployment is made the authorized_keys
//	file should be modified to give user the access to challenge container.
// GitRemote: It represents git remote we fetch the details from.
// BeastScriptsDir: Directory containing beast scripts for user login etc.
// JWTSecret: It contains the string used for HMAC signing the token.
//
// An example of a config file
//
// ```toml
// authorized_keys_file = "~/.beast/auth_file"
// scripts_dir = "~/.beast/scripts"
//
// [remote]
// url = "git@github.com:sdslabs/hack-test.git"
// name = "hack-test"
// branch = "master"
// ssh_key = "~/.beast/secrets/key.priv"
// ```
type BeastConfig struct {
	AuthorizedKeysFile string    `toml:"authorized_keys_file"`
	BeastScriptsDir    string    `toml:"scripts_dir"`
	AllowedBaseImages  []string  `toml:"allowed_base_images"`
	AvailableSidecars  []string  `toml:"available_sidecars"`
	GitRemote          GitRemote `toml:"remote"`
	JWTSecret          string    `toml:"jwt_secret"`
	SlackWebHookURL    string    `toml:"slack_webhook"`
	DiscordWebHookURL  string    `toml:"disocrd_webhook"`
	TickerFrequency    int       `toml:"ticker_frequency"`
}

func (config *BeastConfig) ValidateConfig() error {
	log.Debug("Validating BeastConfig structure")

	if config.AuthorizedKeysFile != "" {
		err := utils.CreateFileIfNotExist(config.AuthorizedKeysFile)
		if err != nil {
			log.Errorf("Error while creating authorized_keys file : %s", config.AuthorizedKeysFile)
		}

		config.AuthorizedKeysFile, err = filepath.Abs(config.AuthorizedKeysFile)
		if err != nil {
			return fmt.Errorf("Error while getting absolute path : %s", err)
		}
	} else {
		defaultAuthKeyFile := filepath.Join(os.Getenv("HOME"), core.DEFAULT_AUTH_KEYS_FILE)
		log.Warnf("No authorized_keys file path provided, using default : %s", defaultAuthKeyFile)
		config.AuthorizedKeysFile = defaultAuthKeyFile
	}

	if config.BeastScriptsDir == "" {
		defaultBeastScriptDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR)
		log.Warn("No scripts directory provided for beast, using default : %s", defaultBeastScriptDir)
		config.BeastScriptsDir = defaultBeastScriptDir
	} else {
		err := utils.CreateIfNotExistDir(config.BeastScriptsDir)
		if err != nil {
			log.Error("Error while creating beast scripts directory")
			return err
		}
	}

	if !utils.StringInSlice(core.DEFAULT_BASE_IMAGE, config.AllowedBaseImages) {
		config.AllowedBaseImages = append(config.AllowedBaseImages, core.DEFAULT_BASE_IMAGE)
	}

	if config.JWTSecret == "" {
		log.Error("The secret string is empty in beast config")
		return fmt.Errorf("Invalid config")
	}

	err := config.GitRemote.ValidateGitConfig()
	if err != nil {
		return err
	}

	if config.TickerFrequency <= 0 {
		log.Error("Time is not provided or is less than equal to zero so default time is taken")
		config.TickerFrequency = core.DEFAULT_TICKER_FREQUENCY
	}

	return nil
}

type GitRemote struct {
	Url        string `toml:"url"`
	RemoteName string `toml:"name"`
	Branch     string `toml:"branch"`
	Secret     string `toml:"ssh_key"`
}

func (config *GitRemote) ValidateGitConfig() error {
	if config.Url == "" || config.RemoteName == "" || config.Secret == "" {
		log.Error("One of url, RemoteName or ssh_key is missing in the config")
		return errors.New("Git remote config not valid, config parameters missing")
	}

	gitUrlRegexp, err := regexp.Compile(config.Url)
	if err != nil {
		eMsg := fmt.Errorf("Error while compiling git url regex : %s", err)
		return eMsg
	}

	if !gitUrlRegexp.MatchString(config.Url) {
		return errors.New("The provided git url is not valid.")
	}

	if config.Branch == "" {
		log.Warn("Branch for git remote not provided, using %s", core.GIT_REMOTE_DEFAULT_BRANCH)
		config.Branch = core.GIT_REMOTE_DEFAULT_BRANCH
	}

	err = utils.ValidateFileExists(config.Secret)
	log.Debugf("Using git ssh secret : %s", config.Secret)
	if err != nil {
		return fmt.Errorf("Provided ssh key file(%s) does not exists : %s", config.Secret, err)
	}

	return nil
}

// From the path of the config file provided as an arguement this function
// loads the parse the config file and load it into the BeastConfig
// structure. After parsing it validates the data in the config file and returns
// error if the validation fails.
func LoadBeastConfig(configPath string) (BeastConfig, error) {
	var config BeastConfig

	err := utils.ValidateFileExists(configPath)
	if err != nil {
		return config, err
	}

	_, err = toml.DecodeFile(configPath, &config)
	if err != nil {
		return config, err
	}

	log.Debugf("Parsed beast global config file is : %s", config)
	err = config.ValidateConfig()
	if err != nil {
		return config, err
	}

	log.Debug("Global beast config file config.toml has been verified")
	return config, nil
}

// Update the USED_PORT_LIST variable in config.
// Don't do this very often, we do this once during syncing the git repository
// then whenever you need updated used port list you need to sync the git remote
// by beast.
func UpdateUsedPortList() {
	beastRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	challengeDir := filepath.Join(beastRemoteDir, Cfg.GitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR)

	dirs := utils.GetAllDirectoriesName(challengeDir)
	for _, dir := range dirs {
		configFilePath := filepath.Join(dir, core.CHALLENGE_CONFIG_FILE_NAME)
		var config BeastChallengeConfig
		_, err := toml.DecodeFile(configFilePath, &config)
		if err == nil {
			USED_PORTS_LIST = append(USED_PORTS_LIST, config.Challenge.Env.Ports...)
		}
	}

	log.Debugf("Used port list updated: %v", USED_PORTS_LIST)
}

var Cfg BeastConfig = InitConfig()
var SkipAuthorization bool
var USED_PORTS_LIST []uint32

func InitConfig() BeastConfig {
	configPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
	cfg, err := LoadBeastConfig(configPath)

	if err != nil {
		log.Errorf("Error while loading the beast global config : %s", err)
		os.Exit(1)
	}

	log.Debugf("CONFIG LOAD: New Config : %v", cfg)
	return cfg
}

func ReloadBeastConfig() error {
	configPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
	cfg, err := LoadBeastConfig(configPath)

	if err != nil {
		return fmt.Errorf("Error while loading beast config: %s", err)
	}

	Cfg = cfg
	log.Debugf("CONFIG LOAD: New Config : %v", Cfg)
	return nil
}
