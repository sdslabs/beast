package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sdslabs/beastv4/core/cache"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"github.com/sdslabs/beastv4/pkg/sse"
	"github.com/sdslabs/beastv4/utils"

	"github.com/sdslabs/beastv4/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const controllerLockFile = "controller.lock"

func loadDefaultAuthorPassword(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	expanded, err := utils.ExpandHomePath(path)
	if err != nil {
		return "", err
	}
	if err := utils.ValidateSecretFile(expanded); err != nil {
		return "", fmt.Errorf("validate default author password file: %w", err)
	}
	file, err := os.Open(expanded)
	if err != nil {
		return "", fmt.Errorf("open default author password file: %w", err)
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, 130))
	if err != nil {
		return "", fmt.Errorf("read default author password file: %w", err)
	}
	password := strings.TrimSuffix(strings.TrimSuffix(string(contents), "\n"), "\r")
	if len(password) < 12 || len(password) > 128 || strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("default author password must contain between 12 and 128 non-whitespace bytes")
	}
	return password, nil
}

func acquireControllerLock(directory string) (*os.File, error) {
	path := filepath.Join(directory, controllerLockFile)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open controller lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("another Beast controller is already active: %w", err)
	}
	if err := file.Truncate(0); err != nil {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
		return nil, fmt.Errorf("truncate controller lock: %w", err)
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid()) + "\n"); err != nil {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
		return nil, fmt.Errorf("write controller lock: %w", err)
	}
	return file, nil
}

func releaseControllerLock(file *os.File) {
	if file == nil {
		return
	}
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	_ = file.Close()
}

var (
	BEAST_GRAPH_CACHE       = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CACHE_DIR, core.BEAST_GRAPH_CACHE)
	BEAST_LEADERBOARD_CACHE = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CACHE_DIR, core.BEAST_LEADERBOARD_CACHE)
)

func stopApiScheduler() {
	/* Stop the scheduler if it hasn't stopped already */
	log.Infoln("Stopping the API scheduler...")
	api.BeastScheduler.Stop()
	log.Infoln("API scheduler stopped")
}

func stopWorkerQueue() {
	log.Infoln("Stopping the worker queue...")
	if manager.Q == nil {
		log.Infoln("Worker queue was not started")
		return
	}
	manager.Q.Stop()
	log.Infoln("Worker queue stopped")
}

func stopRemoteManagers() {
	log.Infoln("Stopping the remote manager queue...")
	remoteManager.Stop()
	log.Infoln("Remote Manager queue stopped")
}

func cleanupCacheConnections() {
	log.Infoln("Cleaning up cache connections...")
	log.Infoln("Terminating cache connection...")

	err := cache.Close()
	if err != nil {
		log.Errorln("Unable to terminate cache connections:", err)
	} else {
		log.Infoln("Cache connections terminated successfully")
	}
}

func cleanupDatabaseConnections() {
	log.Infoln("Terminating database connection...")

	if database.Db == nil {
		log.Infoln("Database was not initialized")
		return
	}
	sqlDB, err := database.Db.DB()
	if err != nil {
		log.Errorln("Unable to access database connection:", err)
	} else if err := sqlDB.Close(); err != nil {
		log.Errorln("Unable to close database connection:", err)
	} else {
		log.Infoln("Database connections terminated successfully")
	}
}

func writeJson(data any, location string) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(location, bytes, 0644)
}

func saveLeaderboardCache() {
	var topUsers []uint

	/* get maximum possible number of entires */
	leaderboardFresh, err := database.QueryTopUsersByFrozenScore(math.MaxInt)
	if err == nil {
		for i := 0; i < 10 && i < len(leaderboardFresh); i++ {
			user := leaderboardFresh[i]
			topUsers = append(topUsers, user.ID)
		}
	}

	if err = writeJson(leaderboardFresh, BEAST_LEADERBOARD_CACHE); err != nil {
		log.Errorln(fmt.Sprintf("Failed to write to leaderboard cache: %s", err.Error()))
	}

	graphFresh := database.QueryTimeSeriesForTopUsers(topUsers)
	if err = writeJson(graphFresh, BEAST_GRAPH_CACHE); err != nil {
		log.Errorln(fmt.Sprintf("Failed to write to graph cache: %s", err.Error()))
	}
}

func stopSseNotificationHub() {
	sse.Shutdown()
}

// NOTE: Why this function is not in 'core/utils/cleanup.go'
// 1. 'database', 'manager' and 'remoteManager' depend on utils so moving these inside utils will create cycle dependencies.
// 2. 'core/utils/cleanup.go' contains functions to clean up challenges, while this function does a more broad cleanup.

func cleanup() {
	log.Info("Starting graceful shutdown cleanup...")

	stopApiScheduler()
	stopWorkerQueue()
	stopSseNotificationHub()

	stopRemoteManagers()

	saveLeaderboardCache()

	cleanupCacheConnections()
	cleanupDatabaseConnections()

	// - Clean up temporary files: found no files to be cleared as of now
	// - Close network connections: all ssh connections are already closed and no new network connections as of now

	// - Stop background goroutines

	log.Info("Graceful shutdown cleanup completed")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Beast API server",
	Long:  "Run beast API server using beast/api/server, optionally an argument can be provided to specify the port to run the server on.",

	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(core.BEAST_GLOBAL_DIR); os.IsNotExist(err) {
			log.Infof("%s directory not found... running Beast bootsteps...\n", core.BEAST_GLOBAL_DIR)

			if err := runBeastBootsteps(); err != nil {
				return fmt.Errorf("run Beast bootsteps: %w", err)
			}

			log.Infoln("beast bootsteps complete... starting beast server")
		} else if err != nil {
			return fmt.Errorf("inspect Beast directory: %w", err)
		}
		if Port != "" {
			port, err := strconv.Atoi(Port)
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("invalid API port %q", Port)
			}
		}
		controllerLock, err := acquireControllerLock(core.BEAST_GLOBAL_DIR)
		if err != nil {
			return err
		}
		defer releaseControllerLock(controllerLock)

		ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stopSignals()

		defaultAuthorPassword, err := loadDefaultAuthorPassword(DefaultAuthorPasswordFile)
		if err != nil {
			return err
		}
		err = api.RunBeastApiServer(ctx, Port, defaultAuthorPassword, AutoDeploy, HealthProbe, PeriodicSync, NoCache)
		if ctx.Err() != nil {
			log.Infoln("Shutdown signal received.")
		}
		if manager.Q != nil && database.Db != nil {
			cleanup()
		}
		if err != nil {
			return fmt.Errorf("Beast API stopped: %w", err)
		}
		log.Infoln("Server stopped gracefully.")
		return nil
	},
}
