package sidecar

import (
	"fmt"

	"github.com/sdslabs/beastv4/core/sidecar/mysql"
)

type SidecarAgent interface {
	Bootstrap(configPath string) error
	Destroy(configPath string) error
}

func GetSidecarAgent(sidecar string) (SidecarAgent, error) {
	switch sidecar {
	case "mysql":
		return &mysql.MySQLAgent{}, nil
	default:
		return nil, fmt.Errorf("Not a valid sidecar name: %s", sidecar)
	}
}
