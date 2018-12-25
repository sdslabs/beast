package config

import (
	"strings"

	"github.com/sdslabs/beastv4/core"
)

func GetAvailableChallengeTypes() []string {
	types := core.AVAILABLE_CHALLENGE_TYPES

	for k, _ := range core.DockerBaseImageForWebChall {
		for k1, _ := range core.DockerBaseImageForWebChall[k] {
			for k2, _ := range core.DockerBaseImageForWebChall[k][k1] {
				newType := "web:" + k + ":" + k1 + ":" + k2
				newType = strings.TrimRight(strings.Replace(newType, "default", "", -1), ":")
				types = append(types, newType)
			}
		}
	}

	return types
}
