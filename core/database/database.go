package database

import (
	"crypto/rand"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/sdslabs/beastv4/core"
	beastConfig "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBMux *sync.Mutex
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
	dsn.RawQuery = query.Encode()
	return dsn.String()
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
	DBMux = &sync.Mutex{}
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
	if err := MigrateChallengeMaintainers(); err != nil {
		return fmt.Errorf("migrate challenge maintainers: %w", err)
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

func BackupAndReset() {
	if err := LoadDbConfig(); err != nil {
		log.Error(err)
		return
	}

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

	backupPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_REMOTES_DIR)
	err = utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return
	}

	backupPath = filepath.Join(backupPath, core.BEAST_REMOTES_DIR+time.Now().Format("20060102150405")+".bak")
	oldPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)
	err = os.Rename(oldPath, backupPath)
	if err != nil {
		log.Errorf("Error while backing up remote dir: %s", err)
		return
	}

	backupPath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_BACKUP_DIR, core.BEAST_STAGING_DIR)

	err = utils.CreateIfNotExistDir(backupPath)
	if err != nil {
		log.Errorf("Error while creating backup directory: %s", err)
		return
	}

	oldPath = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR)
	backupPath = filepath.Join(backupPath, core.BEAST_STAGING_DIR+time.Now().Format("20060102150405")+".bak")
	err = os.Rename(oldPath, backupPath)
	if err != nil {
		log.Errorf("Error while backing up staging dir: %s", err)
		return
	}
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
	cmd := exec.Command("pg_dump", "-U", dbConfig.User, "-h", dbConfig.Host, "-p", dbConfig.Port, "-F", "c", "-f", filepath.Join(backupPath, backupFile), dbConfig.Dbname)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Backup error: %s\n", string(output))
		return err
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
	err := TerminateDatabaseConnections()
	if err != nil {
		log.Errorf("Unable to terminate connections %s", err)
		return err
	}

	dropCmd := exec.Command("dropdb", "-U", dbConfig.User, "-h", dbConfig.Host, "-p", dbConfig.Port, "--force", dbConfig.Dbname)
	dropCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))

	output, err := dropCmd.CombinedOutput()
	if err != nil {
		log.Printf("Drop DB error: %s\n", string(output))
		return err
	}

	createCmd := exec.Command("psql", "-U", dbConfig.User, "-h", dbConfig.Host, "-p", dbConfig.Port, "-d", "postgres", "-c", "CREATE DATABASE "+dbConfig.Dbname+";")
	createCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))

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
	if dbConfig == (beastConfig.PsqlConfig{}) {
		if err := LoadDbConfig(); err != nil {
			return err
		}
	}
	terminateCmd := exec.Command(
		"psql",
		"-U", dbConfig.User,
		"-h", dbConfig.Host,
		"-p", dbConfig.Port,
		"-d", "postgres",
		"-c",
		fmt.Sprintf("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid();", dbConfig.Dbname),
	)
	terminateCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))

	output, err := terminateCmd.CombinedOutput()
	outputStr := string(output)
	if err != nil {
		log.Errorf("Terminate connections error: %s\n", outputStr)
		return err
	}
	log.Debug(outputStr)
	return nil
}

func RestoreDatabase(backupFile string) error {
	if err := LoadDbConfig(); err != nil {
		return err
	}

	err := TerminateDatabaseConnections()
	if err != nil {
		log.Errorf("Unable to terminate connections: %s ", err)
		return err
	}

	err = utils.ValidateFileExists(backupFile)
	if err != nil {
		return fmt.Errorf("backup file does not exist: %s", backupFile)
	}

	restoreCmd := exec.Command(
		"pg_restore",
		"-U", dbConfig.User,
		"-h", dbConfig.Host,
		"-p", dbConfig.Port,
		"-d", dbConfig.Dbname,
		"--no-owner",
		"--clean",
		"--if-exists",
		backupFile,
	)
	restoreCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))

	output, err := restoreCmd.CombinedOutput()
	if err != nil {
		log.Printf("Restore DB error: %s\n", string(output))
		return fmt.Errorf("failed to restore database from %s: %v", backupFile, err)
	}

	log.Println("Database restored successfully from:", backupFile)
	return nil
}
