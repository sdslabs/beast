package main

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/utils"
	"github.com/spf13/cobra"
)

var managerUsernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,31}$`)

func createAuthorAdminPrereq() error {
	if err := config.InitConfig(); err != nil {
		return err
	}

	auth.Init(core.ITERATIONS, core.HASH_LENGTH, core.TIMEPERIOD, core.ISSUER, config.Cfg.JWTSecret, []string{core.USER_ROLES["author"], core.USER_ROLES["maintainer"]}, []string{core.USER_ROLES["admin"]}, []string{core.USER_ROLES["contestant"]})
	return database.Init()
}

func validateManagerArguments(role string) error {
	if strings.TrimSpace(Name) == "" {
		return fmt.Errorf("name of %s is required", role)
	}
	if len(Name) > 128 {
		return fmt.Errorf("name must not exceed 128 bytes")
	}
	if !managerUsernamePattern.MatchString(Username) {
		return fmt.Errorf("username must match %s", managerUsernamePattern.String())
	}
	mailbox, err := mail.ParseAddress(Email)
	if err != nil || mailbox.Address != Email {
		return fmt.Errorf("email must be a canonical mailbox address")
	}
	return nil
}

func promptNewPassword() (string, error) {
	password := utils.PromptSecret("Enter password")
	confirmation := utils.PromptSecret("Confirm password")
	if password != confirmation {
		return "", fmt.Errorf("password confirmation does not match")
	}
	if len(password) < 12 || len(password) > 128 || strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password must contain between 12 and 128 non-whitespace bytes")
	}
	return password, nil
}

func createManager(role string) error {
	if err := validateManagerArguments(role); err != nil {
		return err
	}
	password, err := promptNewPassword()
	if err != nil {
		return err
	}
	if err := createAuthorAdminPrereq(); err != nil {
		return fmt.Errorf("initialize configuration: %w", err)
	}
	sqlDB, err := database.Db.DB()
	if err != nil {
		return fmt.Errorf("access database connection: %w", err)
	}
	defer sqlDB.Close()
	if err := coreUtils.CreateAdminOrAuthor(Name, Username, Email, password, role); err != nil {
		return fmt.Errorf("create %s: %w", role, err)
	}
	return nil
}

var createAuthorCmd = &cobra.Command{
	Use:   "create-author",
	Short: "Creates a new author",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createManager("author")
	},
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Creates a new admin",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createManager("admin")
	},
}
