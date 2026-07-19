package utils

import (
	"fmt"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
)

func CreateAdminOrAuthor(name string, username string, email string, password string, role string) error {
	authModel, err := auth.CreateModel(username, password, core.USER_ROLES[role])
	if err != nil {
		return err
	}
	userEntry := database.User{
		Name:      name,
		AuthModel: authModel,
		Email:     email,
	}
	err = database.CreateUserEntry(&userEntry)
	if err != nil {
		return fmt.Errorf("create author entry: %w", err)
	}
	return nil
}

func DeleteChallengeEntryWithPorts(challname string) error {
	chall, found, err := database.FindFirstChallengeEntry("name", challname)
	if err != nil {
		return fmt.Errorf("Error while querying database : %v", err)
	}
	if !found {
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
