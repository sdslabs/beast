package manager

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/pkg/notify"
	"github.com/sdslabs/beastv4/pkg/probes"
	"github.com/sdslabs/beastv4/pkg/remoteManager"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

var HEALTH_CHECKER = false

// Check for static challenegs' assets to be present on staging server.
// At the time of writing, Beast deploys assets to localhost only.
// So it will check only on localhost
func CheckStaticChallenge(chall database.Challenge) error {
	assets := strings.Split(chall.Assets, core.DELIMITER)
	for _, asset := range assets {
		filepath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, chall.Name, core.BEAST_STATIC_FOLDER, asset)
		err := utils.ValidateFileExists(filepath)
		if err != nil {
			err = fmt.Errorf("static chall: %s not staged. Asset: %s Missing!", chall.Name, asset)
			log.Error(err)
			return err
		}
	}
	return nil
}

// Check for container running or not.
func containerProber(chall database.Challenge) error {
	challHost := chall.ServerDeployed
	if challHost == core.LOCALHOST || challHost == "" {
		containers, err := cr.SearchRunningContainerByFilter(map[string]string{"id": chall.ContainerId})
		if err != nil || len(containers) <= 0 {
			err = fmt.Errorf("error while searching for container with id %s on server: %s", chall.ContainerId, chall.ServerDeployed)
			return err
		}
	} else {
		server := config.Cfg.AvailableServers[chall.ServerDeployed]
		containers, err := remoteManager.SearchRunningContainerByFilterRemote(map[string]string{"id": chall.ContainerId}, server)
		if err != nil || len(containers) <= 0 {
			err = fmt.Errorf("error while searching for container with id %s on remote server: %s", chall.ContainerId, chall.ServerDeployed)
			return err
		}
	}
	return nil
}

// Check for challenge running or not
func ChallengesHealthProber(waitTime int) {
	log.Info("Starting Challenge Health Check prober.")
	challs, err := database.QueryChallengeEntriesMap(map[string]interface{}{
		"Status":       core.DEPLOY_STATUS["deployed"],
		"health_check": 1,
	})

	if err != nil {
		log.Errorf("Error while querying challenges : %v", err)
		return
	}

	for _, chall := range challs {
		if chall.Format != core.STATIC_CHALLENGE_TYPE_NAME {
			allocatedPorts, err := database.GetAllocatedPorts(chall)
			if err != nil {
				log.Errorf("Error while accessing database : %v", err)
				continue
			}

			log.Debugf("Doing HealthCheck Probe for %s", chall.Name)

			// Do a better job at health probing mechanism.
			if len(allocatedPorts) > 0 {
				port := int(allocatedPorts[0].PortNo)
				prober := probes.NewTcpProber()
				result, err := prober.Probe(chall.ServerDeployed, port, time.Duration(core.DEFAULT_PROBE_TIMEOUT)*time.Second)
				if err != nil {
					msg := fmt.Sprintf("NETWORK HEALTH CHECK %s: %s : %s", result, chall.Name, err)
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(msg)
					go notify.SendNotification(notify.Error, msg)
				} else {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Info("NETWORK HEALTH CHECK returned success.")
				}
				err = containerProber(chall)
				if err != nil {
					msg := fmt.Sprintf("CONTAINER HEALTH CHECK %s: %s : %s", result, chall.Name, err)
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(msg)
					go notify.SendNotification(notify.Error, msg)
				} else {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Info("CONTAINER HEALTH CHECK returned success.")
				}
			}
		} else {
			err := CheckStaticChallenge(chall)
			if err != nil {
				msg := fmt.Sprintf("HEALTHCHECK Failure: %s : %s", chall.Name, err)
				log.WithFields(log.Fields{
					"ChallName": chall.Name,
				}).Error(msg)
				go notify.SendNotification(notify.Error, msg)
			}
		}
	}
}

// Check for Remote Server running or not
func ServerHealthProber(waitTime int) {
	for _, server := range config.Cfg.AvailableServers {
		if server.Active && server.Host != core.LOCALHOST {
			err := remoteManager.PingServer(server)
			if err != nil {
				msg := fmt.Sprintf("SERVER HEALTH CHECK Faliure: %s : %s", server.Host, err)
				log.WithFields(log.Fields{
					"ChallName": server.Host,
				}).Error(msg)
				go notify.SendNotification(notify.Error, msg)
			} else {
				log.WithFields(log.Fields{
					"ChallName": server.Host,
				}).Info("SERVER HEALTH CHECK returned success.")
			}
		}
	}
}

// Check for beast services running or not
func BeastHeathCheckProber(waitTime int) {
	if !HEALTH_CHECKER {
		log.Info("Starting Health Check prober.")
		HEALTH_CHECKER = true
		for {
			go ChallengesHealthProber(waitTime)
			go ServerHealthProber(waitTime)
			// Wait for some time before next probing.
			time.Sleep(time.Duration(waitTime) * time.Second)
		}
	} else {
		log.Warn("Health Checker Already Running. Not Starting Again")
	}
}
