package database

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBMux *sync.Mutex
	Db    *gorm.DB
	dberr error
)

var (
	BEAST_GLOBAL_DIR string = filepath.Join(os.Getenv("HOME"), ".beast")
	dbConfig         Config
)

type Config struct {
	PsqlConf PSQLConfig `toml:"psql_config"`
}
type PSQLConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Dbname   string `toml:"dbname"`
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	SslMode  string `toml:"sslmode"`
}

// Db config is loaded separately here for temp use because init() function is
// called during initialization of package.
// It is also loaded during db backup/reset
func LoadDbConfig() {
	if _, err := toml.DecodeFile(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME), &dbConfig); err != nil {
		log.Fatalf("Error loading TOML file: %v", err)
	}
}

// Connect psql database
func ConnectDatabase() error {
	LoadDbConfig()
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s", dbConfig.PsqlConf.User, dbConfig.PsqlConf.Password, dbConfig.PsqlConf.Dbname, dbConfig.PsqlConf.Host, dbConfig.PsqlConf.Port, dbConfig.PsqlConf.SslMode)
	Db, dberr = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if dberr != nil {
		log.Error("Error while initializing the database.", dberr)
		return dberr
	}
	log.Debug("Databse initialized")
	return nil
}

// Set up the initial bootstrapping for interacting with the
// Postgresql database for beast. The Db variable is the connection variable for the
// database, which is not closed after creating a connection here and can
// be used further after this.
func init() {
	DBMux = &sync.Mutex{}
	if Db == nil {
		dberr = ConnectDatabase()
		if dberr != nil {
			log.Error("Error while initializing the database.", dberr)
		}
	}

	if err := Db.SetupJoinTable(&Challenge{}, "Users", &UserChallenges{}); err != nil {
		log.Fatalf("Cannot create related models: %s", err)
	}
	if err := Db.SetupJoinTable(&User{}, "Challenges", &UserChallenges{}); err != nil {
		log.Fatalf("Cannot create related models: %s", err)
	}

	Db.AutoMigrate(&Challenge{}, &Transaction{}, &Port{}, &User{}, &Tag{}, &Notification{}, &DynamicFlag{})

	users, err := QueryUserEntries("email", core.DEFAULT_USER_EMAIL)
	if err != nil {
		log.Errorf("Error while checking dummy user entry.")
		os.Exit(1)
	}

	if len(users) == 0 {
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

func BackupAndReset() {
	beastRemoteDir := filepath.Join(BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	beastStagingDir := filepath.Join(BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	LoadDbConfig()
	err := BackupDatabase()
	if err != nil {
		log.Errorf("Error while backing up database: %s", err)
		return
	}
	err = ResetDatabase()
	if err != nil {
		log.Errorf("Error while resetting up database: %s", err)
		return
	}
	err = os.Rename(beastRemoteDir, beastRemoteDir+time.Now().Format("20060102150405")+".bak")
	if err != nil {
		log.Errorf("Error while backing up remote dir: %s", err)
		return
	}
	err = os.Rename(beastStagingDir, beastStagingDir+time.Now().Format("20060102150405")+".bak")
	if err != nil {
		log.Errorf("Error while backing up staging dir: %s", err)
		return
	}
}

func BackupDatabase() error {
	if dbConfig == (Config{}) {
		LoadDbConfig()
	}
	backupFile := fmt.Sprintf("%s_%s.bak", dbConfig.PsqlConf.Dbname, time.Now().Format("20060102150405"))
	cmd := exec.Command("pg_dump", "-U", dbConfig.PsqlConf.User, "-h", dbConfig.PsqlConf.Host, "-p", dbConfig.PsqlConf.Port, "-F", "c", "-f", filepath.Join(core.BEAST_GLOBAL_DIR, backupFile), dbConfig.PsqlConf.Dbname)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.PsqlConf.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Backup error: %s\n", string(output))
		return err
	}
	log.Debug("Backup successful.")
	return nil
}

func ResetDatabase() error {
	if dbConfig == (Config{}) {
		LoadDbConfig()
	}
	err := TerminateDatabaseConnections()
	if err != nil {
		log.Errorf("Unable to terminate connections ", err)
		return err
	}

	dropCmd := exec.Command("psql", "-U", dbConfig.PsqlConf.User, "-h", dbConfig.PsqlConf.Host, "-p", dbConfig.PsqlConf.Port, "-c", "DROP DATABASE IF EXISTS "+dbConfig.PsqlConf.Dbname)
	dropCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.PsqlConf.Password))

	output, err := dropCmd.CombinedOutput()
	if err != nil {
		log.Printf("Drop DB error: %s\n", string(output))
		return err
	}

	createCmd := exec.Command("psql", "-U", dbConfig.PsqlConf.User, "-h", dbConfig.PsqlConf.Host, "-p", dbConfig.PsqlConf.Port, "-c", "CREATE DATABASE "+dbConfig.PsqlConf.Dbname)
	createCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.PsqlConf.Password))

	output, err = createCmd.CombinedOutput()
	if err != nil {
		log.Printf("Create DB error: %s\n", string(output))
		return err
	}
	log.Debug("Reset successful.")
	return nil
}

// Terminate all active connections before dropping
func TerminateDatabaseConnections() error {
	if dbConfig == (Config{}) {
		LoadDbConfig()
	}
	terminateCmd := exec.Command(
		"psql",
		"-U", dbConfig.PsqlConf.User,
		"-h", dbConfig.PsqlConf.Host,
		"-p", dbConfig.PsqlConf.Port,
		"-d", "postgres",
		"-c",
		fmt.Sprintf("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid();", dbConfig.PsqlConf.Dbname),
	)
	terminateCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.PsqlConf.Password))

	output, err := terminateCmd.CombinedOutput()
	if err != nil {
		log.Errorf("Terminate connections error: %s\n", string(output))
		return err
	}
	log.Debug(output)
	return nil
}

func RestoreDatabase(backupFile string) error {
	LoadDbConfig()

	err := TerminateDatabaseConnections()
	if err != nil {
		log.Error("Unable to terminate connections ", err)
		return err
	}
	
	err = utils.ValidateFileExists(backupFile)
	if err != nil {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	restoreCmd := exec.Command(
		"pg_restore",
		"-U", dbConfig.PsqlConf.User,
		"-h", dbConfig.PsqlConf.Host,
		"-p", dbConfig.PsqlConf.Port,
		"-d", dbConfig.PsqlConf.Dbname,
		"--no-owner",
		"--clean",
		"--if-exists",
		backupFile,
	)
	restoreCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.PsqlConf.Password))

	output, err := restoreCmd.CombinedOutput()
	if err != nil {
		log.Printf("Restore DB error: %s\n", string(output))
		return fmt.Errorf("failed to restore database from %s: %v", backupFile, err)
	}

	log.Println("Database restored successfully from:", backupFile)
	return nil
}
