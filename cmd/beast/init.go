package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"os/exec"
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

func crateBeastDbUser(db *sql.DB) error {
	if result := utils.PromptBinary("Create default beast postgres user?"); !result {
		return errors.New("failed to create database")
	}

	_, err := db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", core.BEAST_DEFAULT_DB_USER, core.BEAST_DEFAULT_DB_PASSWORD))
	return err
}

func createBeastDatabase(db *sql.DB) error {
	if result := utils.PromptBinary("Create default beast postgres database?"); !result {
		return errors.New("failed to create database")
	}

	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", core.BEAST_DEFAULT_DB_DATABASE))
	return err
}

func initDb() error {
	log.Infoln("Initializing database...")

	password := utils.PromptSecret("Enter postgres super user password (leave blank if none):")
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", "postgres", password, "postgres", "disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	var exists int
	err = db.QueryRow(fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname = '%s'", core.BEAST_DEFAULT_DB_USER)).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		if err = crateBeastDbUser(db); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Infoln(fmt.Sprintf("User %s already exists", core.BEAST_DEFAULT_DB_USER))
	}

	err = db.QueryRow(fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = '%s'", core.BEAST_DEFAULT_DB_DATABASE)).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		if err = createBeastDatabase(db); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Infoln(fmt.Sprintf("Database %s already exists", core.BEAST_DEFAULT_DB_DATABASE))
	}

	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", core.BEAST_DEFAULT_DB_DATABASE, core.BEAST_DEFAULT_DB_USER))
	if err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Granted all privellages on database: %s to user: %s", core.BEAST_DEFAULT_DB_DATABASE, core.DEFAULT_USER_EMAIL))
	return nil
}

func initAdmin() error {
	if result := utils.PromptBinary("Create an administrative user for beast?"); result {
		return createAdminCmd.Execute()
	}

	return nil
}

func runBeastBootsteps() error {
	log.Infoln("Setting up sample environment for beast...")

	if err := initDirectories(); err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Created %s directory", core.BEAST_GLOBAL_DIR))

	if err := initAuthorizedKeysFile(); err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Created %s", AUTHORIZED_KEYS_FILE))

	if err := initBeastConfig(); err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("Beast global config file initiliased at %s", BEAST_GLOBAL_CONFIG))

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

		log.Infoln(COLOR_GREEN + "Please run beast server by following command:-")
		log.Infoln("******************")
		log.Infoln("*  " + BLINK_ON + "beast run -v" + BLINK_OFF + "  *")
		log.Infoln("******************" + RESET)
	},
}
