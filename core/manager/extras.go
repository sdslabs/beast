package manager

import (
	"github.com/sdslabs/beastv4/core"
	sc "github.com/sdslabs/beastv4/core/sidecar"
	log "github.com/sirupsen/logrus"
)

func SidecarDeployer() {
	mysqlDeployer, err := sc.GetSidecarDeployer(core.MYSQL_SIDECAR_HOST)
	if err != nil {
		log.Errorf("MySQL sidecar not deployed :%s", err)
	}
	mysqlDeployer.DeploySidecar()
	mongoDeployer, err := sc.GetSidecarDeployer(core.MONGO_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Mongo sidecar not deployed : %s", err)
	}
	mongoDeployer.DeploySidecar()
	staticDeployer, err := sc.GetSidecarDeployer(core.STATIC_SIDECAR_HOST)
	if err != nil {
		log.Errorf("Beast-static car not deployed :%s",err)
	}
	staticDeployer.DeploySidecar()
}
