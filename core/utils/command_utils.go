package utils

import (
	"fmt"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	log "github.com/sirupsen/logrus"
)

func CreateAdminOrAuthor(name string, username string, email string, password string, role string) {
	userEntry := database.User{
		Name:      name,
		AuthModel: auth.CreateModel(username, password, core.USER_ROLES[role]),
		Email:     email,
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
