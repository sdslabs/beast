package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sdslabs/beastv4/core"
	coreCache "github.com/sdslabs/beastv4/core/cache"
	"github.com/sdslabs/beastv4/core/config"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		core.BEAST_GLOBAL_DIR,
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CACHE_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SECRETS_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_LOGO_DIR),
		filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_ASSETS_DIR, core.BEAST_EMAIL_TEMPLATE_DIR),
	}

	for _, dir := range directories {
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			return err
		}
		if err := os.Chmod(dir, 0700); err != nil {
			return err
		}
	}

	return nil
}

func checkDockerDaemon() error {
	log.Infoln("Checking docker daemon...")
	output, err := utils.RunCommand(30*time.Second, nil, "docker", "info", "--format", "{{.ServerVersion}}")
	if err != nil {
		return fmt.Errorf("Docker daemon is unavailable: %w; output: %s", err, output)
	}
	return nil
}

func createBeastRedisUser(cache *redis.Client, configuration *config.RedisConfig) error {
	ctx := context.Background()
	log.Warnln("Beast expects Redis ACLs to be enabled. If ACLs are not configured, some features may not function correctly.")

	result, err := cache.ACLUsers(ctx).Result()
	if err != nil {
		return err
	}

	for _, user := range result {
		if user == configuration.User {
			log.Infoln(fmt.Sprintf("Redis user %s already exists", configuration.User))
			break
		}
	}

	_, err = cache.ACLSetUser(ctx, configuration.User, "on", ">"+configuration.Password, "~beast:*", "+@all").Result()
	if err != nil {
		return err
	}
	log.Infoln(fmt.Sprintf("Initialised redis user %s", configuration.User))

	err = cache.Do(ctx, "acl", "save").Err()
	if err != nil {
		return fmt.Errorf("error while trying to save the acl file: %s", err.Error())
	}

	return nil
}

func initCache() error {
	log.Infoln("Initializing cache...")

	redisConfig := config.Cfg.RedisConf
	tlsConfig, err := coreCache.NewTLSConfig(redisConfig.TLS, redisConfig.CAFile, redisConfig.ServerName, redisConfig.Host)
	if err != nil {
		return err
	}
	var cache *redis.Client
	if utils.PromptBinary("Do you use password authentication for the redis default user?") {
		cache = redis.NewClient(&redis.Options{
			Addr:         net.JoinHostPort(redisConfig.Host, redisConfig.Port),
			Username:     core.REDIS_DEFAULT_USER,
			Password:     utils.PromptSecret("Enter default redis user password"),
			TLSConfig:    tlsConfig,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		})
	} else {
		cache = redis.NewClient(&redis.Options{
			Addr:         net.JoinHostPort(redisConfig.Host, redisConfig.Port),
			Username:     core.REDIS_DEFAULT_USER,
			TLSConfig:    tlsConfig,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cache.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connected to redis: %s", err.Error())
	}

	defer cache.Close()
	return createBeastRedisUser(cache, &config.Cfg.RedisConf)
}

func createBeastDbUser(db *sql.DB, configuration *config.PsqlConfig) error {
	if result := utils.PromptBinary("Create default beast postgres user?"); !result {
		return errors.New("failed to create database")
	}

	_, err := db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD %s;", pq.QuoteIdentifier(configuration.User), utils.QuoteLiteral(configuration.Password)))
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

func postgresAdminDSN(configuration config.PsqlConfig, password string) string {
	dsn := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(core.POSTGRES_SUPER_USER, password),
		Host:   net.JoinHostPort(configuration.Host, configuration.Port),
		Path:   "postgres",
	}
	query := dsn.Query()
	query.Set("sslmode", configuration.SslMode)
	if configuration.SSLRootCert != "" {
		query.Set("sslrootcert", configuration.SSLRootCert)
	}
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func initDb() error {
	log.Infoln("Initializing database...")
	configuration := config.Cfg.PsqlConf

	isPostgres, err := dbUserCheck()
	if err != nil {
		return err
	}

	var db *sql.DB
	if isPostgres && isLoopbackHost(configuration.Host) {
		log.Infoln("Attempting to connect to postgres as postgres super user...")

		db, err = sql.Open("pgx", "user=postgres dbname=postgres sslmode=disable")

		if err != nil {
			return err
		}
	} else {
		log.Warnln("A PostgreSQL superuser password is required for this connection...")
		if !utils.PromptBinary("Connect using password authentication for the postgres superuser?") {
			log.Errorln("Cannot continue with postgres setup... Please run this command as the postgres super user (preferred) or use password authentication.")
			return errors.New("failed to initialize database")
		}
		password := utils.PromptSecret("Enter postgres superuser password:")
		db, err = sql.Open("pgx", postgresAdminDSN(configuration, password))
		if err != nil {
			return err
		}
	}

	defer db.Close()

	var exists int
	err = db.QueryRow("SELECT 1 FROM pg_roles WHERE rolname = $1", configuration.User).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		if err = createBeastDbUser(db, &configuration); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Infoln(fmt.Sprintf("User %s already exists", configuration.User))
	}

	log.Infoln(fmt.Sprintf("Changing password for user %s", configuration.User))
	query := fmt.Sprintf("ALTER USER %s WITH PASSWORD %s", pq.QuoteIdentifier(configuration.User), utils.QuoteLiteral(configuration.Password))
	_, err = db.Exec(query)
	if err != nil {
		return err
	}

	err = db.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", configuration.Dbname).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		if err = createBeastDatabase(db, &configuration); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		log.Infoln(fmt.Sprintf("Database %s already exists", configuration.Dbname))
	}

	_, err = db.Exec(fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", pq.QuoteIdentifier(configuration.Dbname), pq.QuoteIdentifier(configuration.User)))
	if err != nil {
		return err
	}

	log.Infoln(fmt.Sprintf("%s set as owner of database %s", configuration.User, configuration.Dbname))
	return nil
}

func initAdmin() error {
	if result := utils.PromptBinary("Create an administrative user for beast?"); result {
		if err := config.InitConfig(); err != nil {
			return err
		}

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
		confirmation := utils.PromptSecret("Confirm admin password")
		if password != confirmation || len(password) < 12 || len(password) > 128 {
			return errors.New("admin password confirmation must match and contain 12 to 128 bytes")
		}

		if err := createAuthorAdminPrereq(); err != nil {
			return err
		}
		if err := coreUtils.CreateAdminOrAuthor(name, username, email, password, "admin"); err != nil {
			return err
		}
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
	if err := ensureLocalTLSCertificate(); err != nil {
		return err
	}

	if err := checkDockerDaemon(); err != nil {
		return err
	}

	log.Infoln("Verified Docker Daemon running")

	if err := config.InitConfig(); err != nil {
		return err
	}

	if err := initCache(); err != nil {
		return err
	}

	log.Infoln("Verified redis setup for beast")

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

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runBeastBootsteps(); err != nil {
			return fmt.Errorf("complete Beast bootsteps: %w", err)
		}

		log.Infoln(COLOR_GREEN + "Please run beast server by following command:-" + RESET)
		log.Infoln(COLOR_GREEN + "******************" + RESET)
		log.Infoln(COLOR_GREEN + "*  " + BLINK_ON + "beast run -v" + BLINK_OFF + "  *" + RESET)
		log.Infoln(COLOR_GREEN + "******************" + RESET)
		return nil
	},
}
