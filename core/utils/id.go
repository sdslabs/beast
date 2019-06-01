package utils

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/sdslabs/beastv4/core"
)

func GetTempImageId(a string) string {
	b := fmt.Sprintf("%s_%s", core.IMAGE_NA, a)
	if len(b) > 30 {
		return b[:30]
	}
	return b
}

func GetTempContainerId(a string) string {
	b := fmt.Sprintf("%s_%s", core.CONTAINER_NA, a)
	if len(b) > 30 {
		return b[:30]
	}
	return b
}

func EncodeID(a string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(a)))[:30]
}

func IsImageIdValid(a string) bool {
	return (!strings.HasPrefix(a, core.IMAGE_NA) && a != "")
}

func IsContainerIdValid(a string) bool {
	return ((!strings.HasPrefix(a, core.CONTAINER_NA)) && a != "")
}
