package utils

import (
	"fmt"
	"io/ioutil"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

func CreateAuthor(name, username, email, publicKeyPath, password string) {
	var sshKey []byte
	if publicKeyPath != "" {
		err := utils.ValidateFileExists(publicKeyPath)
		if err != nil {
			log.Errorf("Error while checking validity of file(%v): %v : ", publicKeyPath, err)
			return
		}

		sshKey, err = ioutil.ReadFile(publicKeyPath)
		if err != nil {
			log.Errorf("Error while reading file: %v", err)
			return
		}

	} else {
		log.Warn("SSH Key for author is not provided")
	}

	userEntry := database.User{
		Name:      name,
		AuthModel: auth.CreateModel(username, password, core.USER_ROLES["author"]),
		Email:     email,
		SshKey:    string(sshKey),
	}
	err := database.CreateUserEntry(&userEntry)
	if err != nil {
		log.Errorf("Error while creating author entry : %v", err)
	}
}

func DeleteChallengeEntryWithPorts(challname string) error {
	chall, err := database.QueryFirstChallengeEntry("name", challname)
	if err != nil {
		return fmt.Errorf("Error while querying database : %v", err)
	}
	if chall.Name == "" {
		return nil
	}
	ports, err := database.GetAllocatedPorts(chall)
	if err != nil {
		return fmt.Errorf("Error while querying from database : %v", err)
	}
	if err = database.DeleteRelatedPorts(ports); err != nil {
		return fmt.Errorf("Error while deleting ports from database : %v", err)
	}
	if err = database.DeleteChallengeEntry(&chall); err != nil {
		return fmt.Errorf("Error while deleting challentry from database : %v", err)
	}
	return nil
}
