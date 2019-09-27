package manager

import (
	"github.com/sdslabs/beastv4/core"
	sc "github.com/sdslabs/beastv4/core/sidecar"
	log "github.com/sirupsen/logrus"
)

func SidecarDeployer() {
	mysqlDeployer, err := sc.GetSidecarDeployer(core.MYSQL_SIDECAR_HOST)
	if err != nil {
		log.Errorf("MySQL deployer not deployed")
	}
	mysqlDeployer.DeploySidecar()
	mongoDeployer, err := sc.GetSidecarDeployer(core.MONGO_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Mongo deployer not deployed")
	}
	mongoDeployer.DeploySidecar()
	staticDeployer, err := sc.GetSidecarDeployer(core.STATIC_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Beast-static deployer not deployed")
	}
	staticDeployer.DeploySidecar()
}
