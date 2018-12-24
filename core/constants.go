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
	MAX_PORT_PER_CHALL             uint32 = 3
	BEAST_CHALLENGES_STATIC_PORT   uint32 = 80
	BEAST_STATIC_CONTAINER_NAME    string = "beast-static"
	BEAST_STAGING_AREA_MOUNT_POINT string = "/beast"
	BEAST_STATIC_FOLDER            string = "static"
	STATIC_CHALLENGE_TYPE_NAME     string = "static"
	PUBLIC                         string = "public"
	DEFAULT_BASE_IMAGE             string = "ubuntu:16.04"
)

var DEPLOY_STATUS = map[string]string{
	"unknown":    "Unknown",
	"staging":    "Staging",
	"committing": "Commiting",
	"deploying":  "Deploying",
	"deployed":   "Deployed",
	"building":   "Building",
}

var DockerBaseImageForWebChall = map[string]map[string]map[string]string{
	"php": map[string]map[string]string{
		"7.1": map[string]string{
			"cli":     "php:7.1-cli",
			"apache":  "php:7.1-apache",
			"default": "php:7.1-cli",
		},
		"5.6": map[string]string{
			"cli":     "php:5.6-cli",
			"apache":  "php:5.6-apache",
			"default": "php:5.6-cli",
		},
		"default": map[string]string{
			"default": "php:5.6-cli",
		},
	},
	"node": map[string]map[string]string{
		"10": map[string]string{
			"default": "node:10-jessie",
		},
		"default": map[string]string{
			"default": "node:10-jessie",
		},
	},
	"python": map[string]map[string]string{
		"2.7": map[string]string{
			"default": "python:2.7-jessie",
		},
		"3.5": map[string]string{
			"default": "python:3.5-jessie",
		},
		"3.6": map[string]string{
			"default": "python:3.6-jessie",
		},
		"default": map[string]string{
			"default": "python:2.7-jessie",
		},
	},
	"default": map[string]map[string]string{
		"default": map[string]string{
			"default": DEFAULT_BASE_IMAGE,
		},
	},
}
