package core

import (
	"os"
	"path/filepath"
)

var (
	BEAST_GLOBAL_DIR = filepath.Join(os.Getenv("HOME"), ".beast")
)

const (
	CONFIG_FILE_NAME       string = "beast.toml"
	DEFAULT_CHALLENGE_NAME string = "Backdoor-Challenge"
	DEFAULT_AUTHOR_NAME    string = "ghost"
	BEAST_LOG_FILE         string = "beast.log"
	BEAST_DATABASE         string = "beast.db"
	BEAST_STAGING_DIR      string = "staging"
	MAX_PORT_PER_CHALL     uint32 = 3
)

var DEPLOY_STATUS = map[string]string{
	"unknown": "Unknown",
	"stage":   "Staging",
	"commit":  "Commiting",
	"deploy":  "Deploying",
	"build":   "Building",
}
