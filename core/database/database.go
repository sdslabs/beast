package database

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/sdslabs/beastv4/core"
	beastConfig "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBMux *sync.RWMutex
	Db    *gorm.DB
)

var dbConfig beastConfig.PsqlConfig

func LoadDbConfig() error {
	if beastConfig.Cfg != nil {
		dbConfig = beastConfig.Cfg.PsqlConf
		return nil
	}
	cfg, err := beastConfig.LoadBeastConfig(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME))
	if err != nil {
		return fmt.Errorf("load database config: %w", err)
	}
	dbConfig = cfg.PsqlConf
	return nil
}

func postgresDSN(config beastConfig.PsqlConfig) string {
	dsn := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(config.User, config.Password),
		Host:   net.JoinHostPort(config.Host, config.Port),
		Path:   config.Dbname,
	}
	query := dsn.Query()
	query.Set("sslmode", config.SslMode)
	if config.SSLRootCert != "" {
		query.Set("sslrootcert", config.SSLRootCert)
	}
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func postgresEnvironment(config beastConfig.PsqlConfig) []string {
	environment := append(os.Environ(), "PGPASSWORD="+config.Password, "PGSSLMODE="+config.SslMode)
	if config.SSLRootCert != "" {
		environment = append(environment, "PGSSLROOTCERT="+config.SSLRootCert)
	}
	return environment
}

// Connect psql database
func ConnectDatabase() error {
	if err := LoadDbConfig(); err != nil {
		return err
	}
	db, err := gorm.Open(postgres.Open(postgresDSN(dbConfig)), &gorm.Config{})
	if err != nil {
		return err
	}
	Db = db
	log.Debug("Database initialized")
	return nil
}

// Set up the initial bootstrapping for interacting with the
// Postgresql database for beast. The Db variable is the connection variable for the
// database, which is not closed after creating a connection here and can
// be used further after this.
func Init() error {
	DBMux = &sync.RWMutex{}
	if Db == nil {
		if err := ConnectDatabase(); err != nil {
			return fmt.Errorf("initialize database: %w", err)
		}
	}
	if err := Db.SetupJoinTable(&Challenge{}, "Users", &UserChallenges{}); err != nil {
		return fmt.Errorf("configure user challenge relation: %w", err)
	}
	if err := Db.SetupJoinTable(&User{}, "Challenges", &ChallengeMaintainer{}); err != nil {
		return fmt.Errorf("configure user maintainer relation: %w", err)
	}
	if err := Db.SetupJoinTable(&Challenge{}, "Users", &ChallengeMaintainer{}); err != nil {
		return fmt.Errorf("configure challenge maintainer relation: %w", err)
	}

	if err := Db.SetupJoinTable(&User{}, "Hints", &UserHint{}); err != nil {
		return fmt.Errorf("configure user hint relation: %w", err)
	}

	// UserHint must be explicitly migrated since GORM's AutoMigrate on User only handles
	// the users table, not custom join table structs. Without this, the created_at and
	// challenge_id columns on user_hints won't be added to existing databases.
	err := Db.AutoMigrate(&Challenge{}, &Transaction{}, &Port{}, &User{}, &UserChallenges{}, &ChallengeMaintainer{}, &Tag{}, &Notification{}, &Hint{}, &DynamicFlag{}, &DynamicFlagClaim{}, &DynamicScoreDirty{}, &OTP{}, &UserHint{})
	if err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if err := MigrateSubmissionGuards(); err != nil {
		return fmt.Errorf("migrate submission guards: %w", err)
	}
	if err := MigratePortUniqueness(); err != nil {
		return fmt.Errorf("migrate port uniqueness: %w", err)
	}
	if err := MigrateChallengeIdentifiers(); err != nil {
		return fmt.Errorf("migrate challenge identifiers: %w", err)
	}
	if err := MigrateChallengeMaintainers(); err != nil {
		return fmt.Errorf("migrate challenge maintainers: %w", err)
	}
	if err := ClearLegacyOTPSecrets(); err != nil {
		return fmt.Errorf("clear legacy OTP secrets: %w", err)
	}

	users, err := QueryUserEntries("email", core.DEFAULT_USER_EMAIL)
	if err != nil {
		return fmt.Errorf("check dummy user entry: %w", err)
	}

	if len(users) == 0 {
		log.Info("Creating dummy user entry")

		randPass := make([]byte, 32)
		if _, err := rand.Read(randPass); err != nil {
			return fmt.Errorf("generate dummy user password: %w", err)
		}
		authModel, err := auth.CreateModel(core.DEFAULT_USER_NAME, string(randPass), core.USER_ROLES["author"])
		if err != nil {
			return fmt.Errorf("hash dummy user password: %w", err)
		}

		err = CreateUserEntry(&User{
			Name:      core.DEFAULT_USER_NAME,
			Email:     core.DEFAULT_USER_EMAIL,
			AuthModel: authModel,
		})

		if err != nil {
			return fmt.Errorf("create dummy user entry: %w", err)
		}
	}
	return nil
}

func BackupAndReset() error {
	if err := LoadDbConfig(); err != nil {
		return err
	}
	if err := BackupDatabase(); err != nil {
		return fmt.Errorf("back up database: %w", err)
	}
	if err := ResetDatabase(); err != nil {
		return fmt.Errorf("reset database: %w", err)
	}
	return nil
}

func BackupDatabase() error {
	if dbConfig == (beastConfig.PsqlConfig{}) {
		if err := LoadDbConfig(); err != nil {
			return err
		}
	}

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.DB_BACKUP_DIR)
	err := utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return err
	}

	backupFile := fmt.Sprintf("%s_%s.bak", dbConfig.Dbname, time.Now().Format("20060102150405"))
	environment := postgresEnvironment(dbConfig)
	output, err := utils.RunCommand(30*time.Minute, environment, "pg_dump", "-U", dbConfig.User, "-h", dbConfig.Host, "-p", dbConfig.Port, "-F", "c", "-f", filepath.Join(backupPath, backupFile), dbConfig.Dbname)
	if err != nil {
		return fmt.Errorf("pg_dump failed: %w; output: %s", err, output)
	}
	log.Debug("Backup successful.")
	return nil
}

func ResetDatabase() error {
	if dbConfig == (beastConfig.PsqlConfig{}) {
		if err := LoadDbConfig(); err != nil {
			return err
		}
	}
	environment := postgresEnvironment(dbConfig)
	output, err := utils.RunCommand(2*time.Minute, environment, "dropdb", "-U", dbConfig.User, "-h", dbConfig.Host, "-p", dbConfig.Port, "--force", dbConfig.Dbname)
	if err != nil {
		return fmt.Errorf("drop database: %w; output: %s", err, output)
	}

	output, err = utils.RunCommand(2*time.Minute, environment, "psql", "-U", dbConfig.User, "-h", dbConfig.Host, "-p", dbConfig.Port, "-d", "postgres", "-c", "CREATE DATABASE "+pq.QuoteIdentifier(dbConfig.Dbname)+";")
	if err != nil {
		return fmt.Errorf("create database: %w; output: %s", err, output)
	}
	log.Debug("Reset successful.")
	return nil
}

func RestoreDatabase(backupFile string) error {
	if err := LoadDbConfig(); err != nil {
		return err
	}

	err := utils.ValidateFileExists(backupFile)
	if err != nil {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	environment := postgresEnvironment(dbConfig)
	output, err := utils.RunCommand(30*time.Minute, environment, "pg_restore",
		"-U", dbConfig.User,
		"-h", dbConfig.Host,
		"-p", dbConfig.Port,
		"-d", dbConfig.Dbname,
		"--no-owner",
		"--clean",
		"--if-exists",
		backupFile,
	)
	if err != nil {
		return fmt.Errorf("restore database from %s: %w; output: %s", backupFile, err, output)
	}

	log.Println("Database restored successfully from:", backupFile)
	return nil
}
