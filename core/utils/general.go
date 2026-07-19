package utils

import (
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/pkg/auth"
)

func GetUser(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("No authorization header.")
	}
	values := strings.Split(authHeader, " ")

	if len(values) != 2 || values[0] != "Bearer" {
		return "", fmt.Errorf("Not a valid authorization header")
	}

	claims, err := auth.AuthorizeClaims(values[1], auth.MANAGER|auth.ADMIN|auth.USER)
	if err != nil {
		return "", fmt.Errorf("invalid authorization token: %w", err)
	}
	return claims.User, nil
}
