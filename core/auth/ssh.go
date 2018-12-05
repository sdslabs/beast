package auth

import (
	"fmt"
	"io/ioutil"

	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// This function takes in the path to the authorized keys file and parses it
// producing a map with ssh-public key as key and options corresponding to that
// as the value to corresponding key in the map.
func ParseAuthorizedKeysFile(filePath string) (map[string][]string, error) {
	authorizedKeysMap := map[string][]string{}
	err := utils.ValidateFileExists(filePath)
	if err != nil {
		log.Error("Error while validating authorized_keys file path")
		return authorizedKeysMap, err
	}

	authorizedKeysBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		eMsg := fmt.Errorf("Failed to load authorized_keys, err: %v", err)
		log.Errorf("Error : %s", eMsg)
		return authorizedKeysMap, eMsg
	}

	var eMsg error

	for len(authorizedKeysBytes) > 0 {
		pubKey, _, options, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			eMsg = fmt.Errorf("Error while parsing authorized_keys file : %s", err)
			log.Error(eMsg.Error())
			break
		}

		authorizedKeysMap[string(pubKey.Marshal())] = options
		authorizedKeysBytes = rest
	}

	return authorizedKeysMap, eMsg
}
