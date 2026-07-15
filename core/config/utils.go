package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sdslabs/beastv4/core"
)

func decodeTOMLFileStrict(path string, target interface{}) error {
	metadata, err := toml.DecodeFile(path, target)
	if err != nil {
		return err
	}

	undecoded := metadata.Undecoded()
	if len(undecoded) == 0 {
		return nil
	}

	keys := make([]string, 0, len(undecoded))
	for _, key := range undecoded {
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	return fmt.Errorf("unknown configuration keys: %s", strings.Join(keys, ", "))
}

func LoadChallengeConfig(path string) (BeastChallengeConfig, error) {
	var config BeastChallengeConfig
	if err := decodeTOMLFileStrict(path, &config); err != nil {
		return BeastChallengeConfig{}, err
	}
	return config, nil
}

func GetAvailableChallengeTypes() []string {
	types := core.AVAILABLE_CHALLENGE_TYPES

	// Extract all the web challenges type.
	for k := range core.DockerBaseImageForWebChall {
		for k1 := range core.DockerBaseImageForWebChall[k] {
			for k2 := range core.DockerBaseImageForWebChall[k][k1] {
				newType := "web:" + k + ":" + k1 + ":" + k2
				newType = strings.TrimRight(strings.Replace(newType, "default", "", -1), ":")
				types = append(types, newType)
			}
		}
	}

	return types
}
