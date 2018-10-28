package core

import (
	"os"
	"path/filepath"
)

var (
	BEAST_GLOBAL_DIR = filepath.Join(os.Getenv("HOME"), ".beast")
)

const (
	CHALLENGE_CONFIG_FILE_NAME     string = "beast.toml"
	BEAST_CONFIG_FILE_NAME         string = "config.toml"
	DEFAULT_CHALLENGE_NAME         string = "Backdoor-Challenge"
	DEFAULT_AUTHOR_NAME            string = "ghost"
	BEAST_LOG_FILE                 string = "beast.log"
	BEAST_DATABASE                 string = "beast.db"
	DEFAULT_AUTH_KEYS_FILE         string = ".ssh/authorized_keys"
	BEAST_STAGING_DIR              string = "staging"
	BEAST_SCRIPTS_DIR              string = "scripts"
	BEAST_REMOTES_DIR              string = "remote"
	BEAST_CHALLENGE_LOGS_DIR       string = "logs"
	GIT_REMOTE_DEFAULT_BRANCH      string = "master"
	GIT_DEFAULT_REMOTE             string = "origin"
	BEAST_REMOTE_CHALLENGE_DIR     string = "challenges"
	MAX_PORT_PER_CHALL             uint32 = 3
	BEAST_CHALLENGES_STATIC_PORT   uint32 = 80
	BEAST_STATIC_CONTAINER_NAME    string = "beast-static"
	BEAST_STAGING_AREA_MOUNT_POINT string = "/beast"
	BEAST_STATIC_FOLDER            string = "static"
	PUBLIC                         string = "public"
)

var DEPLOY_STATUS = map[string]string{
	"unknown":    "Unknown",
	"staging":    "Staging",
	"committing": "Commiting",
	"deploying":  "Deploying",
	"deployed":   "Deployed",
	"building":   "Building",
}
