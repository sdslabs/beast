package database

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/auth"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	_ "gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	Db, dberr = gorm.Open(sqlite.Open(beastDb), &gorm.Config{})

	if dberr != nil {
		log.WithFields(log.Fields{
			"LOCATION": beastDb,
		}).Fatal(dberr)
	}

	Db.AutoMigrate(&Challenge{}, &Transaction{}, &Port{}, &User{}, &Tag{}, &Notification{})

	users, err := QueryUserEntries("email", core.DEFAULT_USER_EMAIL)
	if err != nil {
		log.Errorf("Error while checking dummy user entry.")
		os.Exit(1)
	}

	if users == nil || len(users) == 0 {
		log.Info("Creating dummy user entry")

		salt := make([]byte, 16)
		rand.Read(salt)
		randPass := make([]byte, 32)
		rand.Read(randPass)

		err := CreateUserEntry(&User{
			Name:      core.DEFAULT_USER_NAME,
			Email:     core.DEFAULT_USER_EMAIL,
			AuthModel: auth.CreateModel(core.DEFAULT_USER_NAME, string(randPass), core.USER_ROLES["author"]),
		})

		if err != nil {
			log.Errorf("Error while creating dummy user entry.")
			os.Exit(1)
		}
	}
}
