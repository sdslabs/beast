package core

import (
	"os"
	"path/filepath"
)

var (
	BEAST_GLOBAL_DIR = filepath.Join(os.Getenv("HOME"), ".beast")
)

const (
	CHALLENGE_CONFIG_FILE_NAME string = "beast.toml"
	BEAST_CONFIG_FILE_NAME     string = "config.toml"
	DEFAULT_CHALLENGE_NAME     string = "Backdoor-Challenge"
	DEFAULT_AUTHOR_NAME        string = "ghost"
	BEAST_LOG_FILE             string = "beast.log"
	BEAST_DATABASE             string = "beast.db"
	DEFAULT_AUTH_KEYS_FILE     string = ".ssh/authorized_keys"
	BEAST_STAGING_DIR          string = "staging"
	BEAST_SCRIPTS_DIR          string = "scripts"
	GIT_REMOTE_DEFAULT_BRANCH  string = "master"
	MAX_PORT_PER_CHALL         uint32 = 3
)

var DEPLOY_STATUS = map[string]string{
	"unknown":    "Unknown",
	"staging":    "Staging",
	"committing": "Commiting",
	"deploying":  "Deploying",
	"deployed":   "Deployed",
	"building":   "Building",
}
