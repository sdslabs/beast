package core

import (
	"os"
	"path/filepath"
	"time"
)

var (
	// BEAST_GLOBAL_DIR should always be used only on local deployment
	BEAST_GLOBAL_DIR     = filepath.Join(os.Getenv("HOME"), ".beast")
	AUTHORIZED_KEYS_FILE = filepath.Join(os.Getenv("HOME"), ".ssh", "authorized_keys")
	BEAST_TEMP_DIR       = filepath.Join(os.TempDir(), "beast")
	BEAST_MOUNT_DIR      = func() string {
		if hostDir := os.Getenv("BEAST_HOST_DIR"); hostDir != "" {
			return hostDir
		}
		return BEAST_GLOBAL_DIR
	}()
)

const ( //names
	BEAST_LOCAL_SERVER          string = "BEAST_LOCAL_SERVER"
	CHALLENGE_CONFIG_FILE_NAME  string = "beast.toml"
	BEAST_CONFIG_FILE_NAME      string = "config.toml"
	BEAST_EX_CONFIG_FILE_NAME   string = "example.config.toml"
	BEAST_LOG_FILE              string = "beast.log"
	BEAST_CHEAT_LOG_FILE        string = "cheat.log"
	BEAST_FLAG_LOG_FILE         string = "flag.log"
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
	LOCALHOST                   string = "localhost"
	LOCALHOST_IP                string = "127.0.0.1"
	BEAST_REMOTE_GLOBAL_DIR     string = "~/.beast" // This should always be used for remote only.
	DOCKER_PID                  string = "/var/run/docker.pid"
	BEAST_GRAPH_CACHE           string = "graph_cache.json"
	BEAST_LEADERBOARD_CACHE     string = "leaderboard.json"
	POSTGRES_SUPER_USER         string = "postgres"
	REDIS_DEFAULT_USER          string = "default"
	SAD_CHECK_SCRIPT            string = "check.sh"
)

const ( //paths
	BEAST_DOCKER_CHALLENGE_DIR     string = "/challenge"
	BEAST_CHALLENGE_LOGS_DIR       string = "logs"
	DEFAULT_AUTH_KEYS_FILE         string = "beast_authorized_keys"
	BEAST_STAGING_DIR              string = "staging"
	BEAST_SCRIPTS_DIR              string = "scripts"
	BEAST_REMOTES_DIR              string = "remote"
	BEAST_STAGING_AREA_MOUNT_POINT string = "/beast"
	BEAST_UPLOADS_DIR              string = "uploads"
	BEAST_ASSETS_DIR               string = "assets"
	BEAST_LOGO_DIR                 string = "logo"
	BEAST_EMAIL_TEMPLATE_DIR       string = "mailTemplates"
	BEAST_SECRETS_DIR              string = "secrets"
	BEAST_EXAMPLE_DIR              string = "_examples"
	BEAST_CACHE_DIR                string = "cache"
	SAD_CHECK_SCRIPT_LOCATION      string = BEAST_DOCKER_CHALLENGE_DIR + "/" + SAD_CHECK_SCRIPT
	BEAST_BACKUP_DIR               string = "backup"
	DB_BACKUP_DIR                  string = "db"
	CACHE_BACKUP_DIR               string = "cache"
)

const ( //chall types
	STATIC_CHALLENGE_TYPE_NAME  string = "static"
	SERVICE_CHALLENGE_TYPE_NAME string = "service"
	WEB_CHALLENGE_TYPE_NAME     string = "web"
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
	DEFAULT_BASE_IMAGE           string = "ubuntu:24.03"
	DEFAULT_XINETD_CONF_FILE     string = "xinetd.conf"
	BEAST_STATIC_AUTH_FILE       string = ".static.beast.htpasswd"
	ALLOWED_MIN_PORT_VALUE       uint32 = 10000
	ALLOWED_MAX_PORT_VALUE       uint32 = 20000
)
const ( // default config
	IMAGE_NA                 string  = "IMAGE_NA"
	CONTAINER_NA             string  = "CONTAINER_NA"
	MAX_QUEUE_SIZE           uint32  = 100
	DEFAULT_TICKER_FREQUENCY int     = 1500
	DEFAULT_PROBE_TIMEOUT    int     = 10
	DEFAULT_USER_NAME        string  = "ghost"
	DEFAULT_USER_EMAIL       string  = "ghost@ghost.com"
	DEFAULT_CPU_SHARE        int64   = (1 << 9)
	DEFAULT_MEMORY_LIMIT     int64   = (1 << 29)
	DEFAULT_PIDS_LIMIT       int64   = 100
	DEFAULT_CPU_LIMIT        float32 = .25
	ITERATIONS               int     = 65536
	HASH_LENGTH              int     = 32
	TIMEPERIOD               int64   = 6 * 60 * 60
	SSH_PORT                 uint32  = 22
)

const ( // roles
	ADMIN   int = 1 << 0
	MANAGER int = 1 << 1
	USER    int = 1 << 2
)

const (
	DEFAULT_MINIMUM_EXTEND_TIME        int64 = 300
	DEFAULT_MAXIMUM_EXTEND_TIME        int64 = 600
	DEFAULT_MAXIMUM_INSTANCES_PER_USER int   = 3
)

var (
	DEFAULT_REMOTE_PERIODIC_SYNC_TIME = time.Second * 120
	DEFAULT_HEALTH_CHECK_TIME         = time.Second * 30
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

var DEPLOYMENT_TYPES = map[string]string{
	"docker_compose":  "docker_compose",
	"standard_docker": "standard_docker",
}

var USER_ROLES = map[string]string{
	"contestant": "contestant",
	"admin":      "admin",
	"author":     "author",
	"maintainer": "maintainer",
}

// Available challenge types
var AVAILABLE_CHALLENGE_TYPES = []string{STATIC_CHALLENGE_TYPE_NAME, SERVICE_CHALLENGE_TYPE_NAME, BARE_CHALLENGE_TYPE_NAME, WEB_CHALLENGE_TYPE_NAME}

var DockerBaseImageForWebChall = map[string]map[string]map[string]string{
	"php": {
		"8.2": {
			"cli":     "php:8.2-cli",
			"apache":  "php:8.2-apache",
			"fpm":     "php:8.2-fpm",
			"nginx":   "php:8.2-fpm",
			"default": "php:8.2-cli",
		},
		"default": {
			"default": "php:8.2-cli",
		},
	},
	"node": {
		"20": {
			"default": "node:20-bookworm",
		},
		"22": {
			"default": "node:22-bookworm",
		},
		"default": {
			"default": "node:20-bookworm",
		},
	},
	"python": {
		"3.11": {
			"flask":   "python:3.11-bookworm",
			"django":  "python:3.11-bookworm",
			"default": "python:3.11-bookworm",
		},
		"3.12": {
			"flask":   "python:3.12-bookworm",
			"django":  "python:3.12-bookworm",
			"default": "python:3.12-bookworm",
		},
		"default": {
			"default": "python:3.12-bookworm",
		},
	},
	"default": {
		"default": {
			"default": DEFAULT_BASE_IMAGE,
		},
	},
}

var USER_STATUS = map[string]string{
	"ban":    "ban",
	"unban":  "unban",
	"hide":   "hide",
	"unhide": "unhide",
}

const (
	LEADERBOARD_SIZE       = 25
	LEADERBOARD_GRAPH_SIZE = 12
	SUBMISSIONS_PAGE_SIZE  = 10
)

var NOTIFICATION_SERVICES = []string{
	"slack",
	"discord",
}

const MappingDelimiter = ":"
