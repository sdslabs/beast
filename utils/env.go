package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sdslabs/beastv4/core"
)

// LoadEnvFile attempts to load .env from the working directory or global beast directory.
func LoadEnvFile() {
	paths := []string{
		".env",
		filepath.Join(core.BEAST_GLOBAL_DIR, ".env"),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, "\"'")
				if os.Getenv(key) == "" {
					_ = os.Setenv(key, val)
				}
			}
		}
	}
}
