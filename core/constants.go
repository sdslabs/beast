package core

import (
	"os"
	"path/filepath"
)

var (
	BEAST_GLOBAL_DIR = filepath.Join(os.Getenv("HOME"), ".beast")
	BEAST_LOG_FILE   = "beast.log"
	BEAST_DATABASE   = "beast.db"
)
