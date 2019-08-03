package sidecar

import (
	"fmt"

	"github.com/sdslabs/beastv4/core/sidecar/mongo"
	"github.com/sdslabs/beastv4/core/sidecar/mysql"
	"github.com/sdslabs/beastv4/pkg/cr"
)

type SidecarAgent interface {
	Bootstrap(configPath string) error
	Destroy(configPath string) error
}

func GetSidecarAgent(sidecar string) (SidecarAgent, error) {
	switch sidecar {
	case "mysql":
		return &mysql.MySQLAgent{}, nil
	case "mongo":
		return &mongo.MongoAgent{}, nil
	default:
		return nil, fmt.Errorf("Not a valid sidecar name: %s", sidecar)
	}
}

type SidecarDeployer interface {
	DeploySidecar(configPath *cr.CreateContainerConfig) error
	UndeploySidecar(configPath *cr.CreateContainerConfig) error
}

func GetSidecarDeployer(sidecar string) (SidecarDeployer, error) {
	switch sidecar {
	case "mysql":
		return &mysql.MySQLDeployer{}, nil
	case "mongo":
		return &mongo.MongoDeployer{}, nil
	default:
		return nil, fmt.Errorf("Not a valid sidecar name: %s", sidecar)
	}
}
