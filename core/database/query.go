package database

import (
	"fmt"
	"strings"
)

func validatedQueryColumn(key string, allowed ...string) (string, error) {
	column := strings.ToLower(strings.TrimSpace(key))
	for _, candidate := range allowed {
		if column == candidate {
			return column, nil
		}
	}
	return "", fmt.Errorf("unsupported query column %q", key)
}
