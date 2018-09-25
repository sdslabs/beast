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
)
