package utils

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/pkg/auth"
)

func GetUser(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("No authorization header.")
	}
	values := strings.Split(authHeader, " ")

	if len(values) < 2 {
		return "", fmt.Errorf("Not a valid authorization header")
	}

	jwtToken := values[1]
	userInfoEncr := strings.Split(jwtToken, ".")
	if len(userInfoEncr) != 3 {
		return "", fmt.Errorf("Not a valid JWT token in authorization header: %s", jwtToken)
	}

	sDec, err := b64.StdEncoding.DecodeString(userInfoEncr[1])
	if err != nil {
		return "", fmt.Errorf("Error in decrypting JWT token: %s", err)
	}

	in := []byte(sDec)
	var raw auth.CustomClaims
	json.Unmarshal(in, &raw)

	return raw.User, nil
}
