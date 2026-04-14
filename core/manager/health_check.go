package manager

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/cache"
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
	if chall.Assets == "" {
		return nil
	}

	assets := strings.Split(chall.Assets, core.DELIMITER)
	for _, asset := range assets {
		filepath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, chall.Name, core.BEAST_STATIC_FOLDER, asset)
		err := utils.ValidateFileExists(filepath)
		if err != nil {
			err = fmt.Errorf("static chall: %s not staged. Asset: %s Missing", chall.Name, asset)
			log.Error(err)
			return err
		}
	}
	return nil
}

// Check for container running or not.
func containerProber(chall database.Challenge) error {
	serverDeployed := chall.ServerDeployed
	if config.Cfg.UseLocalDockerDaemon(serverDeployed) {
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
		"status":       core.DEPLOY_STATUS["deployed"],
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
				serverDeployed := chall.ServerDeployed
				if config.Cfg.UseLocalDockerDaemon(serverDeployed) {
					serverDeployed = core.LOCALHOST
				} else if s, ok := config.Cfg.AvailableServers[serverDeployed]; ok {
					serverDeployed = s.Host
				}
				prober := probes.NewTcpProber()
				result, err := prober.Probe(serverDeployed, port, time.Duration(core.DEFAULT_PROBE_TIMEOUT)*time.Second)
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
	for serverDeployed, server := range config.Cfg.AvailableServers {
		if server.Active && !config.Cfg.UseLocalDockerDaemon(serverDeployed) {
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

func BeastHeathCheckProber(waitTime int) {
	if !HEALTH_CHECKER {
		log.Info("Starting Health Check prober.")
		HEALTH_CHECKER = true

		go InstanceCleanupProber()

		for {
			go ChallengesHealthProber(waitTime)
			go ServerHealthProber(waitTime)
			go database.BackupDatabase()
			go cache.BackupCache()
			time.Sleep(time.Duration(waitTime) * time.Second)
		}
	} else {
		log.Warn("Health Checker Already Running. Not Starting Again")
	}
}

func InstanceCleanupProber() {
	log.Info("Starting Instance Cleanup prober with interval: ", core.DEFAULT_HEALTH_CHECK_TIME)

	for {
		QueueExpiredInstances()
		ProcessInstanceDeletionQueue()
		CleanupOrphanedInstanceContainers()
		time.Sleep(core.DEFAULT_HEALTH_CHECK_TIME)
	}
}

func QueueExpiredInstances() {
	log.Debug("Checking for expired instances")

	expired, err := cache.GetExpiredInstances()
	if err != nil {
		log.Warnf("Failed to get expired instances: %v", err)
		return
	}

	for _, instance := range expired {
		log.Infof("Instance %s expired (challenge: %s, user: %s), queueing for deletion",
			instance.InstanceID, instance.ChallengeName, instance.UserID)

		err := cache.QueueInstanceForDeletion(instance.InstanceID)
		if err != nil {
			log.Warnf("Failed to queue instance %s for deletion: %v", instance.InstanceID, err)
		}
	}
}

func ProcessInstanceDeletionQueue() {
	log.Debug("Processing instance deletion queue")

	for i := 0; i < 10; i++ {
		instance, err := cache.PopInstanceForDeletion()
		if err != nil {
			log.Warnf("Error popping from deletion queue: %v", err)
			return
		}

		if instance == nil {
			return
		}

		log.Infof("Processing deletion for instance %s (challenge: %s, container: %s, server: %s)",
			instance.InstanceID, instance.ChallengeName, instance.ContainerID, instance.ServerDeployed)

		err = killInstanceContainer(instance.ContainerID, instance.DeploymentType, instance.InstanceID, instance.ChallengeName, instance.ServerDeployed)
		if err != nil {
			log.Warnf("Failed to kill container for instance %s: %v", instance.InstanceID, err)
		} else {
			log.Infof("Successfully killed container for instance %s", instance.InstanceID)
		}

		cache.FreeContainerPortsOnHost(instance.ServerDeployed, instance.ContainerID)
	}

	queueLen, _ := cache.GetDeletionQueueLength()
	if queueLen > 0 {
		log.Debugf("Deletion queue still has %d items, will process in next cycle", queueLen)
	}
}

func CleanupOrphanedInstanceContainers() {
	log.Debug("Checking for orphaned instance containers")

	cr.CleanupOrphans()
	cr.CleanupOrphanedComposeInstances()

	for serverDeployed, server := range config.Cfg.AvailableServers {
		if server.Active && !config.Cfg.UseLocalDockerDaemon(serverDeployed) {
			remoteManager.CleanupOrphanedOnServer(serverDeployed)
			remoteManager.CleanupOrphanedComposeInstancesOnServer(serverDeployed)
		}
	}
}
