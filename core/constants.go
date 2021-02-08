package core

import (
	"os"
	"path/filepath"
	"time"
)

var (
	BEAST_GLOBAL_DIR     = filepath.Join(os.Getenv("HOME"), ".beast")
	AUTHORIZED_KEYS_FILE = filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys")
	BEAST_TEMP_DIR       = filepath.Join(os.TempDir(), "beast")
)

const ( //names
	BEAST_LOCAL_SERVER          string = "BEAST_LOCAL_SERVER"
	CHALLENGE_CONFIG_FILE_NAME  string = "beast.toml"
	BEAST_CONFIG_FILE_NAME      string = "config.toml"
	BEAST_LOG_FILE              string = "beast.log"
	BEAST_DATABASE              string = "beast.db"
	DEFAULT_CHALLENGE_NAME      string = "Backdoor-Challenge"
	DEFAULT_AUTHOR_NAME         string = "ghost"
	GIT_REMOTE_DEFAULT_BRANCH   string = "master"
	GIT_DEFAULT_REMOTE          string = "origin"
	DEFAULT_DOCKER_FILE         string = "Dockerfile"
	BEAST_REMOTE_CHALLENGE_DIR  string = "challenges"
	BEAST_STATIC_CONTAINER_NAME string = "beast-static"
	BEAST_STATIC_FOLDER         string = "static"
	PUBLIC                      string = "public"
	HIDDEN                      string = ".hidden"
	ISSUER                      string = "beast-sds"
	DELIMITER                   string = "::::"
)

const ( //paths
	BEAST_DOCKER_CHALLENGE_DIR     string = "/challenge"
	BEAST_CHALLENGE_LOGS_DIR       string = "logs"
	DEFAULT_AUTH_KEYS_FILE         string = ".ssh/authorized_keys"
	BEAST_STAGING_DIR              string = "staging"
	BEAST_SCRIPTS_DIR              string = "scripts"
	BEAST_REMOTES_DIR              string = "remote"
	BEAST_STAGING_AREA_MOUNT_POINT string = "/beast"
	BEAST_UPLOADS_DIR              string = "uploads"
	BEAST_ASSETS_DIR               string = "assets"
)

const ( //chall types
	STATIC_CHALLENGE_TYPE_NAME  string = "static"
	SERVICE_CHALLENGE_TYPE_NAME string = "service"
	DOCKER_CHALLENGE_TYPE_NAME  string = "docker"
	BARE_CHALLENGE_TYPE_NAME    string = "bare"
)

const ( // chall actions
	MANAGE_ACTION_UNDEPLOY string = "undeploy"
	MANAGE_ACTION_DEPLOY   string = "deploy"
	MANAGE_ACTION_PURGE    string = "purge"
	MANAGE_ACTION_REDEPLOY string = "redeploy"
	MANAGE_ACTION_SHOW     string = "show"
)

const ( // chall env
	MAX_PORT_PER_CHALL           uint32 = 3
	BEAST_CHALLENGES_STATIC_PORT uint32 = 80
	DEFAULT_BASE_IMAGE           string = "ubuntu:16.04"
	DEFAULT_XINETD_CONF_FILE     string = "xinetd.conf"
	BEAST_STATIC_AUTH_FILE       string = ".static.beast.htpasswd"
	ALLOWED_MIN_PORT_VALUE       uint32 = 10000
	ALLOWED_MAX_PORT_VALUE       uint32 = 20000
)
const ( // default config
	IMAGE_NA                 string = "IMAGE_NA"
	CONTAINER_NA             string = "CONTAINER_NA"
	MAX_QUEUE_SIZE           uint32 = 100
	DEFAULT_TICKER_FREQUENCY int    = 1500
	DEFAULT_PROBE_TIMEOUT    int    = 10
	DEFAULT_USER_NAME        string = "ghost"
	DEFAULT_USER_EMAIL       string = "ghost@ghost.com"
	DEFAULT_CPU_SHARE        int64  = (1 << 9)
	DEFAULT_MEMORY_LIMIT     int64  = (1 << 29)
	DEFAULT_PIDS_LIMIT       int64  = 100
	ITERATIONS               int    = 65536
	HASH_LENGTH              int    = 32
	TIMEPERIOD               int64  = 6 * 60 * 60
)

const ( // roles
	ADMIN   int = 1 << 0
	MANAGER int = 1 << 1
	USER    int = 1 << 2
)

var (
	DEFAULT_REMOTE_PERIODIC_SYNC_TIME = time.Second * 120
)

var DEPLOY_STATUS = map[string]string{
	"undeployed": "Undeployed",
	"staging":    "Staging",
	"committing": "Commiting",
	"deploying":  "Deploying",
	"deployed":   "Deployed",
	"building":   "Building",
	"queued":     "Queued",
}

var USER_ROLES = map[string]string{
	"contestant": "contestant",
	"admin":      "admin",
	"author":     "author",
	"maintainer": "maintainer",
}

const MYSQL_SIDECAR_HOST = "mysql"
const MONGO_SIDECAR_HOST = "mongo"

var SIDECAR_CONTAINER_MAP = map[string]string{
	"mysql": "mysql",
	"mongo": "mongo",
}

var SIDECAR_NETWORK_MAP = map[string]string{
	"mysql": "beast-mysql",
	"mongo": "beast-mongo",
}

var SIDECAR_ENV_PREFIX = map[string]string{
	"mysql": "MYSQL",
	"mongo": "MONGO",
}

// Available challenge types
var AVAILABLE_CHALLENGE_TYPES = []string{STATIC_CHALLENGE_TYPE_NAME, SERVICE_CHALLENGE_TYPE_NAME, BARE_CHALLENGE_TYPE_NAME, DOCKER_CHALLENGE_TYPE_NAME}

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
			"flask":   "python:2.7-jessie",
			"django":  "python:2.7-jessie",
			"default": "python:2.7-jessie",
		},
		"3.5": {
			"flask":   "python:3.5-jessie",
			"django":  "python:3.5-jessie",
			"default": "python:3.5-jessie",
		},
		"3.6": {
			"flask":   "python:3.6-jessie",
			"django":  "python:3.6-jessie",
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

var USER_STATUS = map[string]string{
	"ban":   "ban",
	"unban": "unban",
}
