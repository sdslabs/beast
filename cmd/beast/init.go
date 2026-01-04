package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	COLOR_GREEN string = "\u001B[92m"
	RESET       string = "\u001B[0m"
	BLINK_ON    string = "\u001B[5m"
	BLINK_OFF   string = "\u001B[25m"
)

func initDirectories() error {
	log.Infoln("Creating beast directories...")

	directories := []string{
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_LOGO_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_EMAIL_TEMPLATE_DIR),
	}

	for _, dir := range directories {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkDockerDaemon() error {
	log.Infoln("Checking docker daemon...")
	_, err := os.Stat(core.DOCKER_PID)
	return err
}

func installAir() error {
	log.Infoln("Installing air for live reloading...")

	resp, err := http.Get("https://raw.githubusercontent.com/cosmtrek/air/master/install.sh")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	install := "install.sh"
	out, err := os.Create(install)
	if err != nil {
		return err
	}
	defer out.Close()

	defer os.Remove(install)

	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}

	gopath, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return err
	}
	binDir := filepath.Join(strings.TrimSpace(string(gopath)), "bin")

	cmd := exec.Command("sh", install, "-b", binDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func createBeastDbUser(db *sql.DB, configuration *config.PsqlConfig) error {
	if result := utils.PromptBinary("Create default beast postgres user?"); !result {
		return errors.New("failed to create database")
	}

	_, err := db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD $1;", pq.QuoteIdentifier(configuration.User)), configuration.Password)
	return err
}

func createBeastDatabase(db *sql.DB, configuration *config.PsqlConfig) error {
	if result := utils.PromptBinary("Create default beast postgres database?"); !result {
		return errors.New("failed to create database")
	}

	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(configuration.Dbname)))
	return err
}

func dbUserCheck() (bool, error) {
	current, err := user.Current()
	if err != nil {
		return false, err
	}

	return current.Username == core.POSTGRES_SUPER_USER, nil
}

func initDb() error {
	log.Infoln("Initializing database...")

	var configuration config.BeastConfig
	_, err := toml.DecodeFile(BEAST_GLOBAL_CONFIG, &configuration)
	if err != nil {
		return err
	}

	isPostgres, err := dbUserCheck()
	if err != nil {
		return err
	}

	var db *sql.DB
	if isPostgres {
		log.Infoln("Attempting to connect to postgres as postgres super user...")

		dsn := fmt.Sprintf("user=%s dbname=%s sslmode=%s", "postgres", "postgres", "disable")
		db, err = sql.Open("pgx", dsn)

		if err != nil {
			return err
		}
	} else {
		log.Warnln("Current user is not postgres super user...")

		if utils.PromptBinary("Do you use password authentication for the postgres super user?") {
			password := utils.PromptSecret("Enter postgres super user password (leave blank if none):")

			dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", "postgres", password, "postgres", "disable")
			db, err = sql.Open("pgx", dsn)

			if err != nil {
				return err
			}
		} else {
			log.Errorln("Cannot continue with postgres setup... Please run this command as the postgres super user (preferred) or use password authentication.")
			return errors.New("failed to initialize database")
		}
	}

	defer db.Close()

	var exists int
	err = db.QueryRow("SELECT 1 FROM pg_roles WHERE rolname = $1", configuration.PsqlConf.User).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		if err = createBeastDbUser(db, &configuration.PsqlConf); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Infoln(fmt.Sprintf("User %s already exists", configuration.PsqlConf.User))
	}

	err = db.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", configuration.PsqlConf.Dbname).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		if err = createBeastDatabase(db, &configuration.PsqlConf); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Infoln(fmt.Sprintf("Database %s already exists", configuration.PsqlConf.Dbname))
	}

	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", pq.QuoteIdentifier(configuration.PsqlConf.Dbname), pq.QuoteIdentifier(configuration.PsqlConf.User)))
	if err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Granted all privileges on database: %s to user: %s", configuration.PsqlConf.Dbname, configuration.PsqlConf.User))
	return nil
}

func initAdmin() error {
	if result := utils.PromptBinary("Create an administrative user for beast?"); result {
		config.InitConfig()

		name := utils.PromptString("Enter admin name")
		if name == "" {
			return errors.New("admin name is required")
		}

		email := utils.PromptString("Enter admin email")
		if email == "" {
			return errors.New("admin email is required")
		}

		username := utils.PromptString("Enter admin username")
		if username == "" {
			return errors.New("admin username is required")
		}

		password := utils.PromptSecret("Enter admin password")
		if password == "" {
			return errors.New("admin password is required")
		}

		publicKeyPath, err := utils.PromptPublicKeyFile()
		if err != nil {
			return err
		} else if publicKeyPath == "" {
			return errors.New("no public key file selected")
		}

		database.Init()

		createAuthorAdminPrereq()
		coreUtils.CreateAdminOrAuthor(name, username, email, publicKeyPath, password, "admin")
	}

	return nil
}

func runBeastBootsteps() error {
	log.Infoln("Setting up sample environment for beast...")

	if err := initDirectories(); err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Created %s directory", core.BEAST_GLOBAL_DIR))

	if err := initBeastConfig(); err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Beast global config file initiliazed at %s", BEAST_GLOBAL_CONFIG))

	if err := checkDockerDaemon(); err != nil {
		return err
	}

	log.Infoln("Verified Docker Daemon running")

	if err := installAir(); err != nil {
		return err
	}

	log.Infoln("Successfully installed air for live reloading...")

	if err := initDb(); err != nil {
		return err
	}

	log.Infoln("Verified postgres setup for beast")

	if err := initAdmin(); err != nil {
		return err
	}

	log.Infoln("Verified administrative preferences")

	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Run Beast initial setup bootsetps.",
	Long:  "Initializes beast by setting up beast directory, checking for permission. It also configures the logger and local SQLite database to be used by beast",

	Run: func(cmd *cobra.Command, args []string) {
		err := runBeastBootsteps()

		if err != nil {
			log.Errorln(err.Error())
			log.Errorln("Failed to complete beast bootsteps... fix above errors and try again")
			return
		}

		log.Infoln(COLOR_GREEN + "Please run beast server by following command:-" + RESET)
		log.Infoln(COLOR_GREEN + "******************" + RESET)
		log.Infoln(COLOR_GREEN + "*  " + BLINK_ON + "beast run -v" + BLINK_OFF + "  *" + RESET)
		log.Infoln(COLOR_GREEN + "******************" + RESET)
	},
}
