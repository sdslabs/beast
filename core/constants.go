package core

import (
	"os"
	"path/filepath"
)

var (
	BEAST_GLOBAL_DIR     = filepath.Join(os.Getenv("HOME"), ".beast")
	AUTHORIZED_KEYS_FILE = filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys")
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
	BEAST_DOCKER_CHALLENGE_DIR     string = "/challenge"
	MAX_PORT_PER_CHALL             uint32 = 3
	BEAST_CHALLENGES_STATIC_PORT   uint32 = 80
	BEAST_STATIC_CONTAINER_NAME    string = "beast-static"
	BEAST_STAGING_AREA_MOUNT_POINT string = "/beast"
	BEAST_STATIC_FOLDER            string = "static"
	STATIC_CHALLENGE_TYPE_NAME     string = "static"
	SERVICE_CHALLENGE_TYPE_NAME    string = "service"
	BARE_CHALLENGE_TYPE_NAME       string = "bare"
	PUBLIC                         string = "public"
	DEFAULT_BASE_IMAGE             string = "ubuntu:16.04"
	DEFAULT_XINETD_CONF_FILE       string = "xinetd.conf"
	BEAST_STATIC_AUTH_FILE         string = ".static.beast.htpasswd"
)

var DEPLOY_STATUS = map[string]string{
	"unknown":    "Unknown",
	"staging":    "Staging",
	"committing": "Commiting",
	"deploying":  "Deploying",
	"deployed":   "Deployed",
	"building":   "Building",
}

const MYSQL_SIDECAR_HOST = "mysql"

var SIDECAR_CONTAINER_MAP = map[string]string{
	"mysql": "mysql",
}

var SIDECAR_NETWORK_MAP = map[string]string{
	"mysql": "beast-mysql",
}

var SIDECAR_ENV_PREFIX = map[string]string{
	"mysql": "MYSQL",
}

// Available challenge types
var AVAILABLE_CHALLENGE_TYPES = []string{STATIC_CHALLENGE_TYPE_NAME, SERVICE_CHALLENGE_TYPE_NAME, BARE_CHALLENGE_TYPE_NAME}

var DockerBaseImageForWebChall = map[string]map[string]map[string]string{
	"php": {
		"7.1": {
			"cli":     "php:7.1-cli",
			"apache":  "php:7.1-apache",
			"fpm":     "php:7.1-fpm",
			"default": "php:7.1-cli",
		},
		"5.6": {
			"cli":     "php:5.6-cli",
			"apache":  "php:5.6-apache",
			"fpm":     "php:5.6-fpm",
			"default": "php:5.6-cli",
		},
		"default": {
			"default": "php:5.6-cli",
		},
	},
	"node": {
		"8": {
			"default": "node:8-jessie",
		},
		"10": {
			"default": "node:10-jessie",
		},
		"default": {
			"default": "node:10-jessie",
		},
	},
	"python": {
		"2.7": {
			"default": "python:2.7-jessie",
		},
		"3.5": {
			"default": "python:3.5-jessie",
		},
		"3.6": {
			"default": "python:3.6-jessie",
		},
		"default": {
			"default": "python:2.7-jessie",
		},
	},
	"default": {
		"default": {
			"default": DEFAULT_BASE_IMAGE,
		},
	},
}
