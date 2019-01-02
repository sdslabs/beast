package utils

import (
	"crypto/rand"
	"io/ioutil"

	"github.com/sdslabs/beastv4/database"
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
