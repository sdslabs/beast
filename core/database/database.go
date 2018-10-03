package database

import (
	"path/filepath"
	"sync"

	"github.com/fristonio/beast/core"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
)

var (
	mutex *sync.Mutex
	Db    *gorm.DB
	dberr error
)

// Set up the initial bootstrapping for interacting with the local
// SQLite database for beast. The Db variable is the connection variable for the
// database, which is not closed after creating a connection here and can
// be used further after this.
func init() {
	mutex = &sync.Mutex{}

	beastDb := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_DATABASE)
	Db, dberr = gorm.Open("sqlite3", beastDb)

	if dberr != nil {
		log.WithFields(log.Fields{
			"LOCATION": beastDb,
		}).Fatal(dberr)
	}

	Db.AutoMigrate(&Challenge{}, &Transaction{}, &Port{})
}
