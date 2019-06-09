package database

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sdslabs/beastv4/core"
	log "github.com/sirupsen/logrus"
)

var (
	DBMux *sync.Mutex
	Db    *gorm.DB
	dberr error
)

var (
	BEAST_GLOBAL_DIR string = filepath.Join(os.Getenv("HOME"), ".beast")
	BEAST_DATABASE   string = "beast.db"
)

// Set up the initial bootstrapping for interacting with the local
// SQLite database for beast. The Db variable is the connection variable for the
// database, which is not closed after creating a connection here and can
// be used further after this.
func init() {
	DBMux = &sync.Mutex{}

	beastDb := filepath.Join(BEAST_GLOBAL_DIR, BEAST_DATABASE)
	Db, dberr = gorm.Open("sqlite3", beastDb)

	if dberr != nil {
		log.WithFields(log.Fields{
			"LOCATION": beastDb,
		}).Fatal(dberr)
	}

	Db.AutoMigrate(&Challenge{}, &Transaction{}, &Port{}, &Author{}, &Tag{})
	err := CreateAuthorEntry(&Author{
		Name:  core.DEFAULT_USER_NAME,
		Email: core.DEFAULT_USER_EMAIL,
	})
	if err != nil {
		log.Errorf("Error while creating dummy author entry.")
		os.Exit(1)
	}
}
