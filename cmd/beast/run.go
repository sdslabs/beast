package main

import (
	"encoding/json"
	"fmt"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/core/manager"
	"github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sdslabs/beastv4/api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	BEAST_GRAPH_CACHE       = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CACHE_DIR, core.BEAST_GRAPH_CACHE)
	BEAST_LEADERBOARD_CACHE = filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CACHE_DIR, core.BEAST_LEADERBOARD_CACHE)
)

func stopApiScheduler() {
	log.Infoln("Stopping the API scheduler...")
	api.BeastScheduler.Stop()
	log.Infoln("API scheduler stoped")
}

func stopWorkerQueue() {
	log.Infoln("Stopping the worker queue...")
	manager.Q.Stop()
	log.Infoln("Worker queue stopped")
}

func stopRemoteManagers() {
	log.Infoln("Stopping the remote manager queue...")
	remoteManager.Stop()
	log.Infoln("Remote Manager queue stopped")
}

func cleanUpRunningContainers() {
	log.Infoln("Cleaning up running challenges...")

	challenges, err := database.QueryAllChallenges()
	if err != nil {
		return
	}

	for _, challenge := range challenges {
		if challenge.Status == core.DEPLOY_STATUS["deployed"] {
			err = manager.UndeployChallenge(challenge.Name)
			if err != nil {
				log.Errorln(fmt.Sprintf("Failed to undeploy challenge [Id: %v] %s", challenge.ID, challenge.Name))
				log.Errorln(err.Error())
			} else {
				log.Infoln(fmt.Sprintf("Successfully undeploy challenge [Id: %v] %s", challenge.ID, challenge.Name))
			}
		}
	}
}

func cleanUpDatabaseConnections() {
	log.Infoln("Backing up database...")

	err := database.BackupDatabase()
	if err != nil {
		log.Errorln("Error while backing up database:", err)
	} else {
		log.Infoln("Database backup completed successfully")
	}

	log.Infoln("Terminating  database connection...")

	err = database.TerminateDatabaseConnections()
	if err != nil {
		log.Errorln("Unable to terminate database connections:", err)
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

	leaderboardFresh, err := database.QueryTopUsersByFrozenScore(core.LEADERBOARD_SIZE)
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
		log.Errorln(fmt.Sprintf("Failed to write to leaderboard cache: %s", err.Error()))
	}
}

func storeLoaderBoardCache() {
	err, i := utils.CheckTime()

	if err != nil {
		log.Errorln("Error checking time for competition... saving Graph Data")
	} else if i == 2 {
		// save graph data
	}
}

func cleanup() {
	log.Info("Starting graceful shutdown cleanup...")

	stopApiScheduler()

	stopWorkerQueue()
	stopRemoteManagers()

	saveLeaderboardCache()

	cleanUpRunningContainers()
	cleanUpDatabaseConnections()

	// - Clean up temporary files: found no files to be cleared as of now
	// - Close network connections: all ssh connections are already closed and no new network connections as of now

	// - Stop background goroutines

	log.Info("Graceful shutdown cleanup completed")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run Beast API server",
	Long:  "Run beast API server using beast/api/server, optionally an argument can be provided to specify the port to run the server on.",

	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(core.BEAST_GLOBAL_DIR); os.IsNotExist(err) {
			log.Infof("%s directory not found... running Beast bootsteps...\n", core.BEAST_GLOBAL_DIR)

			if err := runBeastBootsteps(); err != nil {
				log.Error("Error while running Beast bootsteps.")
				os.Exit(1)
			}

			log.Infoln("beast bootsteps complete... starting beast server")
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go api.RunBeastApiServer(Port, DefaultAuthorPassword, AutoDeploy, HealthProbe, PeriodicSync, NoCache)
		<-sigChan

		log.Infoln("\nShutdown signal received.")
		cleanup()
		log.Infoln("Server stopped gracefully.")
	},
}
