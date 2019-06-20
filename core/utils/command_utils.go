package utils

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"

	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/utils"
	log "github.com/sirupsen/logrus"
)

func CreateAuthor(name, email, publicKeyPath string) {
	err := utils.ValidateFileExists(publicKeyPath)
	if err != nil {
		log.Error("Error while checking file existence: %v", err)
		return
	}

	sshKey, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		log.Error("Error while reading file: %v", err)
		return
	}

	rMessage := make([]byte, 128)
	rand.Read(rMessage)

	authorEntry := database.Author{
		Name:          name,
		Email:         email,
		SshKey:        string(sshKey),
		AuthChallenge: rMessage,
	}

	err = database.CreateAuthorEntry(&authorEntry)
	if err != nil {
		log.Error("Error while creating author entry : %v", err)
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
