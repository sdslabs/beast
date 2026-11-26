package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/core"
)

func formatTempID(prefix, name string) string {
	b := fmt.Sprintf("%s_%s", prefix, name)
	if len(b) <= 30 {
		return b
	}
	sum := sha256.Sum256([]byte(name))
	hashSuffix := hex.EncodeToString(sum[:])[:8]
	maxPrefixLen := 30 - 1 - len(hashSuffix)
	if maxPrefixLen < len(prefix)+1 {
		maxPrefixLen = len(prefix) + 1
	}
	return fmt.Sprintf("%s_%s", b[:maxPrefixLen], hashSuffix)
}

func GetTempImageId(a string) string {
	return formatTempID(core.IMAGE_NA, a)
}

func GetTempContainerId(a string) string {
	return formatTempID(core.CONTAINER_NA, a)
}

func IsImageIdValid(a string) bool {
	return (!strings.HasPrefix(a, core.IMAGE_NA) && a != "")
}

func IsContainerIdValid(a string) bool {
	return ((!strings.HasPrefix(a, core.CONTAINER_NA)) && a != "")
}
