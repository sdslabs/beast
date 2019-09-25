package manager

import (
	sc "github.com/sdslabs/beastv4/core/sidecar"
	log "github.com/sirupsen/logrus"
)

func SidecarDeployer() {
	mysqlDeployer, err := sc.GetSidecarDeployer("mysql")
	if err != nil {
		log.Errorf("MySQL deployer not deployed")
	}
	mysqlDeployer.DeploySidecar()
	mongoDeployer, err := sc.GetSidecarDeployer("mongo")
	if err != nil {
		log.Errorf("Mongo deployer not deployed")
	}
	mongoDeployer.DeploySidecar()
	staticDeployer, err := sc.GetSidecarDeployer("beast-static")
	if err != nil {
		log.Errorf("Beast-static deployer not deployed")
	}
	staticDeployer.DeploySidecar()
}
