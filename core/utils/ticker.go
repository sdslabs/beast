package utils

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/database"
	log "github.com/sirupsen/logrus"
)

func HealthTicker(sec int) {
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
				}
				err = HealthChecker(int(allocatedPorts[0].PortNo))
				if err != nil {
					log.WithFields(log.Fields{
						"ChallName": chall.Name,
					}).Error(err)
				}
			}
		}
	}
}

func HealthChecker(port int) error {
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("May be the container stopped : %v", err)
	}
	defer conn.Close()
	return nil
}
