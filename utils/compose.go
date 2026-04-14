package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"regexp"
	"strings"
)

type Compose struct {
	Services map[string]struct {
		Ports []string `yaml:"ports"`
	} `yaml:"services"`
}

var portRegex = regexp.MustCompile(`\$\{([^}]+)}`)

func ExtractPortsFromCompose(composeFile string) ([]string, error) {
	data, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("error while reading compose file: %w", err)
	}
	var raw Compose
	err = yaml.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("error while parsing compose file: %s", err.Error())
	}

	portVariables := make([]string, 0)
	seen := make(map[string]bool)
	for _, service := range raw.Services {
		for _, port := range service.Ports {
			matches := portRegex.FindAllStringSubmatch(port, -1)

			if len(matches) == 0 {
				return nil, fmt.Errorf("port %s is not mapped using an env variable", port)
			}

			for _, match := range matches {
				varName := match[1]

				/* Only ${PORT} is valid, ${PORT:-DEFAULT} should fail */
				if strings.Contains(varName, ":-") {
					return nil, fmt.Errorf("port variable ${%s} uses default value syntax (:-) which is not supported; use ${%s} instead",
						varName, strings.SplitN(varName, ":-", 2)[0])
				}

				if !seen[varName] {
					seen[varName] = true
					portVariables = append(portVariables, varName)
				}
			}
		}
	}

	return portVariables, nil
}
