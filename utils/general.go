package utils

import (
	"encoding/base32"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sdslabs/beastv4/core"
)

// From a list of strings generate a list containing only unique strings
// from the list.
func GetUniqueStrings(list []string) []string {
	var uniq []string
	m := make(map[string]bool)

	for _, str := range list {
		if _, ok := m[str]; !ok {
			m[str] = true
			uniq = append(uniq, str)
		}
	}

	return uniq
}

// Returns a list of directory present in the provided directory
func GetDirsInDir(dirPath string) (error, []string) {
	var dirs []string
	err := ValidateDirExists(dirPath)
	if err != nil {
		return err, dirs
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("Error while reading directory with path %s : %s", dirPath, err), dirs
	}

	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file.Name())
		}
	}

	return nil, dirs
}

// Returns true if in slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func UInt32InList(a uint32, list []uint32) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func GetInvalidImageID(a string) string {
	b := core.IMAGE_NA + "_" + a
	if len(b) > 30 {
		return b[:30]
	}
	return b
}

func GetInvalidContainerID(a string) string {
	b := core.CONTAINER_NA + "_" + a
	if len(b) > 30 {
		return b[:30]
	}
	return b
}

func EncodeID(a string) string {
	return strings.TrimRight(strings.ToLower(base32.StdEncoding.EncodeToString([]byte(a))), "=")
}
