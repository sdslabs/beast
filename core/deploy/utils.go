package deploy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/database"
	tools "github.com/sdslabs/beastv4/templates"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

type BeastBareDockerfile struct {
	Ports       string
	AptDeps     string
	SetupScript string
	RunCmd      string
}

// This if the function which validates the challenge directory
// which is provided as an arguments. This validtions includes
// * A valid directory pointed by challengeDir
// * Valid config file for the challenge in the challengeDir root named as beast.toml
// * Valid challenge directory name in accordance to the challenge config.
func ValidateChallengeConfig(challengeDir string) error {
	configFile := filepath.Join(challengeDir, core.CONFIG_FILE_NAME)

	log.Debug("Checking beast.toml file existance validity")
	err := utils.ValidateFileExists(configFile)
	if err != nil {
		return err
	}

	var config core.BeastConfig
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		return err
	}

	challengeName := filepath.Base(challengeDir)
	if challengeName != config.Challenge.Name {
		return fmt.Errorf("Name of the challenge directory(%s) should match the name provided in the config file(%s)", challengeName, config.Challenge.Name)
	}

	log.Debugf("Parsed config file is : %s", config)
	err = config.ValidateRequiredFields()
	if err != nil {
		return err
	}

	log.Debug("Challenge config file beast.toml is valid")
	return nil
}

// Validates a directory which is considered as a challenge directory
// The function returns an error if the directory is not valid or if it
// does not have valid structure required by beast.
func ValidateChallengeDir(challengeDir string) error {
	log.Debugf("Validating Directory : %s", challengeDir)

	err := utils.ValidateDirExists(challengeDir)
	if err != nil {
		return err
	}

	err = ValidateChallengeConfig(challengeDir)
	if err != nil {
		return err
	}

	log.Infof("Challenge directory validated")
	return nil
}

func GetContextDirPath(dirPath string) (string, error) {
	absContextDir, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf("Unable to get absolute context directory of given context directory %q: %v", dirPath, err)
	}

	err = utils.ValidateDirExists(absContextDir)
	if err != nil {
		return "", err
	}

	return absContextDir, nil
}

// From the provided configFIle path it generates the dockerfile for
// the challenge and returns it as a string. This function again
// assumes that the validation for the configFile is done beforehand
// before calling this function.
func GenerateDockerfile(configFile string) (string, error) {
	log.Debug("Generating dockerfile")
	var config core.BeastConfig
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		return "", err
	}

	data := BeastBareDockerfile{
		Ports:       strings.Trim(strings.Replace(fmt.Sprint(config.Challenge.ChallengeDetails.Ports), " ", " ", -1), "[]"),
		AptDeps:     strings.Join(config.Challenge.ChallengeDetails.AptDeps[:], " "),
		SetupScript: config.Challenge.ChallengeDetails.SetupScript,
		RunCmd:      config.Challenge.ChallengeDetails.RunCmd,
	}

	var dockerfile bytes.Buffer
	log.Debugf("Preparing dockerfile template")
	dockerfileTemplate, err := template.New("dockerfile").Parse(tools.BEAST_BARE_DOCKERFILE_TEMPLATE)
	if err != nil {
		return "", fmt.Errorf("Error while parsing Dockerfile template :: %s", err)
	}

	log.Debugf("Executing dockerfile template with challenge config")
	err = dockerfileTemplate.Execute(&dockerfile, data)
	if err != nil {
		return "", fmt.Errorf("Error while executing Dockerfile template :: %s", err)
	}

	log.Debugf("Dockerfile generated for the challenge")
	return dockerfile.String(), nil
}

// Generate dockerfile context for the challnge from the challenge config
// file path provided as an argument.
// The challengeConfig provided must be a valid path, which is to be ensured by
// the caller. It does not check if the config even exist or is valid. The validation
// for the dockerfile should be done before calling this function. If the file does
// not exist or there is some error  while parsing the setup file
// this function will simply return the error without logging anything.
func GenerateChallengeDockerfileCtx(challengeConfig string) (string, error) {
	log.Debug("Generating challenge dockerfile context from config")
	file, err := ioutil.TempFile("", "Dockerfile.*")
	if err != nil {
		return "", fmt.Errorf("Error while creating a tempfile for Dockerfile :: %s", err)
	}
	defer file.Close()

	dockerfile, err := GenerateDockerfile(challengeConfig)
	if err != nil {
		return "", err
	}

	_, err = file.WriteString(dockerfile)
	if err != nil {
		return "", fmt.Errorf("Error while writing Dockerfile to file :: %s", err)
	}

	log.Debugf("Generated dockerfile lives in : %s", file.Name())
	return file.Name(), nil
}

func UpdateOrCreateChallengeDbEntry(config core.BeastConfig) (database.Challenge, error) {
	challEntry, err := database.QueryFirstChallengeEntry("challenge_id", config.Challenge.Id)
	if err != nil {
		return challEntry, fmt.Errorf("Error while querying challenge with id %s : %s", config.Challenge.Id, err)
	}

	// Challenge is nil, which means the challenge entry does not exist
	// So create a new challenge entry on the basis of the fields provided
	// in the config file for the challenge.
	if challEntry.ChallengeId == "" {
		// For creating a new entry for the challenge first create an entry
		// in the challenge table using the config file.
		authorEntry, err := database.QueryFirstAuthorEntry("email", config.Author.Email)
		if err != nil {
			return challEntry, fmt.Errorf("Error while querying author with email %s", config.Author.Email)
		}

		if authorEntry.Email == "" {
			authorEntry = database.Author{
				Name:   config.Author.Name,
				Email:  config.Author.Email,
				SshKey: config.Author.SSHKey,
			}

			err = database.CreateAuthorEntry(&authorEntry)
			if err != nil {
				return challEntry, fmt.Errorf("Error while creating author entry : %s", err)
			}
		} else {
			if authorEntry.Email != config.Author.Email &&
				authorEntry.SshKey != config.Author.SSHKey &&
				authorEntry.Name != config.Author.Name {
				return challEntry, fmt.Errorf("ERROR, author details did not match with the ones in database")
			}
		}

		challEntry = database.Challenge{
			ChallengeId: config.Challenge.Id,
			Name:        config.Challenge.Name,
			AuthorID:    authorEntry.ID,
			Format:      config.Challenge.ChallengeType,
			Status:      core.DEPLOY_STATUS["unknown"],
		}

		err = database.CreateChallengeEntry(&challEntry)
		if err != nil {
			return challEntry, fmt.Errorf("Error while creating chall entry with config : %s : %v", err, challEntry)
		}
	}

	allocatedPorts, err := database.GetAllocatedPorts(challEntry)
	if err != nil {
		return challEntry, fmt.Errorf("Error while getting allocated ports for : %s : %s", challEntry.ChallengeId, err)
	}

	isAllocated := func(port uint32) bool {
		for _, p := range allocatedPorts {
			if port == p.PortNo {
				return true
			}
		}

		return false
	}

	// Once the challenge entry has been created, add entries to the ports
	// table in the database with the ports to expose
	// for the challenge.
	// TODO: Do all this under a database transaction so that if any port
	// request is not available
	// TODO: delete previously allocated ports to the challenge if they are not
	// in the current required port list.
	for _, port := range config.Challenge.ChallengeDetails.Ports {
		if isAllocated(port) {
			// The port has already been allocated to the challenge
			// Do nothing for this.
			continue
		}

		portEntry := database.Port{
			ChallengeID: challEntry.ID,
			PortNo:      port,
		}

		gotPort, err := database.PortEntryGetOrCreate(portEntry)
		if err != nil {
			return database.Challenge{}, err
		}

		// var gotChall database.Challenge
		// database.Db.Model(&gotPort).Related(&gotChall)

		if gotPort.ChallengeID != challEntry.ID {
			return database.Challenge{}, fmt.Errorf("The port %s requested is already in use by another challenge", gotPort.PortNo)
		}
	}

	return challEntry, nil
}
