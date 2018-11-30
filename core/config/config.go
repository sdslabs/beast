package config

import (
	"crypto/rand"
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
// SecretString: It contains the string used for HMAC signing the token.
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
	GitRemote          GitRemote `toml:"remote"`
	SecretString       string    `toml:"secret_string"`
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

	if config.SecretString == "" {
		buff := make([]byte, 64)
		rand.Read(buff)
		log.Infof("Secret string not provided using \"%s\" as secret", string(buff))
		config.SecretString = string(buff)
	}

	err := config.GitRemote.ValidateGitConfig()
	if err != nil {
		return err
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

var Cfg BeastConfig = InitConfig()

func InitConfig() BeastConfig {
	configPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
	cfg, err := LoadBeastConfig(configPath)

	if err != nil {
		log.Errorf("Error while loading the beast global config : %s", err)
		os.Exit(1)
	}

	return cfg
}
