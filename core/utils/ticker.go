package utils

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/database"
	"github.com/sdslabs/beastv4/notify"
	log "github.com/sirupsen/logrus"
)

func ChallengesHealthTicker(sec int) {
	for {
		time.Sleep(time.Duration(sec) * time.Second)

		challs, err := database.QueryChallengeEntriesMap(map[string]interface{}{
			"Status": core.DEPLOY_STATUS["deployed"],
		})

		if err != nil {
			log.Errorf("Error while querying challenges : %v", err)
			continue
		}

		for _, chall := range challs {
			if chall.Format != core.STATIC_CHALLENGE_TYPE_NAME {
				allocatedPorts, err := database.GetAllocatedPorts(chall)
				if err != nil {
					log.Errorf("Error while accessing database : %v", err)
					continue
				}
				err = HealthChecker(int(allocatedPorts[0].PortNo))
				if err != nil {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(err)
					msg := fmt.Sprintf("HEALTHCHECK ERROR: %s : %s", chall.Name, err)
					log.Error(msg)
					notify.SendNotificationToSlack(notify.Error, msg)
				}
			}
		}
	}
}

func HealthChecker(port int) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("May be the container stopped : %v", err)
	}
	defer conn.Close()
	return nil
}
