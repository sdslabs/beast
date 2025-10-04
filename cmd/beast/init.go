package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/manifoldco/promptui"
	"github.com/nmrshll/go-cp"
	"github.com/sdslabs/beastv4/core"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func initDirectories() error {
	log.Infoln("Creating beast directories")

	directories := []string{
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_LOGO_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_EMAIL_TEMPLATE_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR),
	}

	for _, dir := range directories {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	log.Infof("Created %s directory\n", core.BEAST_GLOBAL_DIR)
	return nil
}

func initAuthorizedKeysFile() error {
	authorizedKeyFile := filepath.Join(core.BEAST_GLOBAL_DIR, core.DEFAULT_AUTH_KEYS_FILE)
	log.Infoln("Defaulting Authorized keys file:", authorizedKeyFile, "... can be changed later")

	// skipping generating secret.key since found no uses for it
	return os.WriteFile(authorizedKeyFile, []byte("auth_keys"), 0666)
}

func initExampleConfig() error {
	response, err := http.Get("https://raw.githubusercontent.com/sdslabs/beast/master/_examples/example.config.toml")
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New("error while downloading: " + response.Status)
	}

	globalConfig, err := os.Create(filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME))
	if err != nil {
		return err
	}
	defer globalConfig.Close()

	_, err = io.Copy(globalConfig, response.Body)
	return err
}

func initBeastConfig() error {
	globalConfig := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
	if _, err := os.Stat(globalConfig); os.IsNotExist(err) {
		exampleConfig := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_EXAMPLE_DIR, core.BEAST_EX_CONFIG_FILE_NAME)
		if _, err = os.Stat(exampleConfig); os.IsNotExist(err) {
			return initExampleConfig()
		}

		log.Infoln("Using example config file:", exampleConfig)
		return cp.CopyFile(exampleConfig, globalConfig)
	}

	return nil
}

func checkDockerDaemon() error {
	_, err := os.Stat(core.DOCKER_PID)
	return err
}

func installAir() error {
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

func promptYesNo(promptLabel string) bool {
	log.Println("hello there")
	log.Println(promptLabel)

	prompt := promptui.Select{
		Label: fmt.Sprintf("%s (y/n)", promptLabel),
		Items: []string{"y", "n"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		log.Errorln("prompt failed to execute, defaulting to no...")
		return false
	}

	return result == "y"
}

func initDb() error {
	if result := promptYesNo("Create default beast postgres user and database?"); !result {
		log.Infoln("not setting up beast postgres user and database... please do so manually or beast will not run... continuing...")
		return nil
	}

	log.Println("Enter postgres super user password (leave blank if none):")
	passwordBytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return err
	}

	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s", "postgres", string(passwordBytes), "postgres", "disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", core.BEAST_DEFAULT_DB_USER, core.BEAST_DEFAULT_DB_PASSWORD))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", core.BEAST_DEFAULT_DB_DATABASE))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", core.BEAST_DEFAULT_DB_DATABASE, core.BEAST_DEFAULT_DB_USER))
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"user":     core.BEAST_DEFAULT_DB_USER,
		"database": core.BEAST_DEFAULT_DB_DATABASE,
	}).Infoln("Created default beast postgres user")

	return nil
}

func runBeastBootsteps() error {
	log.Infoln("Setting up sample environment for beast...")

	if err := initDirectories(); err != nil {
		return err
	}

	if err := initAuthorizedKeysFile(); err != nil {
		return err
	}

	if err := initBeastConfig(); err != nil {
		return err
	}

	if err := checkDockerDaemon(); err != nil {
		log.Errorln(err.Error())
		return errors.New("docker daemon not running... Please start docker daemon and try again... Aborting")
	}

	if err := installAir(); err != nil {
		return err
	}

	if err := initDb(); err != nil {
		return err
	}

	if result := promptYesNo("prompt creation of an administrative user?"); result {
		return createAdminCmd.Execute()
	}

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
			log.Errorln("beast init failed... fix above errors and try again")
			return
		}

		log.Infoln("\u001B[92mPlease run beast server by following command:-\u001B[0m")
		log.Infoln("******************")
		log.Infoln("*  \u001B[5mbeast run -v\u001B[25m  *")
		log.Infoln("******************")
	},
}
