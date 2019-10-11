package manager

import (
	"fmt"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/notify"
	"github.com/sdslabs/beastv4/pkg/probes"
	log "github.com/sirupsen/logrus"
)

const MAX_RETRIES = 3
const CHALLENGE_HOST = "127.0.0.1"

func ChallengesHealthProber(waitTime int) {
	log.Info("Starting Health Check prober.")

	var retries int = 0
	for {
		if retries > MAX_RETRIES {
			return
		}

		challs, err := database.QueryChallengeEntriesMap(map[string]interface{}{
			"Status":       core.DEPLOY_STATUS["deployed"],
			"health_check": 1,
		})

		if err != nil {
			log.Errorf("Error while querying challenges : %v", err)
			retries += 1
			continue
		}

		for _, chall := range challs {
			if chall.Format != core.STATIC_CHALLENGE_TYPE_NAME {
				allocatedPorts, err := database.GetAllocatedPorts(chall)
				if err != nil {
					log.Errorf("Error while accessing database : %v", err)
					retries += 1
					continue
				}

				log.Debugf("Doing helthcheck probe for %s", chall.Name)
				port := int(allocatedPorts[0].PortNo)
				prober := probes.NewTcpProber()
				result, err := prober.Probe(CHALLENGE_HOST, port, time.Duration(core.DEFAULT_PROBE_TIMEOUT)*time.Second)

				if err != nil {
					msg := fmt.Sprintf("HEALTHCHECK %s: %s : %s", result, chall.Name, err)
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(msg)
					notify.SendNotification(notify.Error, msg)
				} else {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Info("HEALTH CHECK returned success.")
				}
			}
		}

		// Wait for some time before next probing.
		time.Sleep(time.Duration(waitTime) * time.Second)
	}
}
