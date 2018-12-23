package manager

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/database"
	tools "github.com/sdslabs/beastv4/templates"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

type BeastBareDockerfile struct {
	DockerBaseImage string
	Ports           string
	AptDeps         string
	SetupScripts    []string
	RunCmd          string
	MountVolume     string
	WebRoot         string
}

// This if the function which validates the challenge directory
// which is provided as an arguments. This validtions includes
// * A valid directory pointed by challengeDir
// * Valid config file for the challenge in the challengeDir root named as beast.toml
// * Valid challenge directory name in accordance to the challenge config.
func ValidateChallengeConfig(challengeDir string) error {
	configFile := filepath.Join(challengeDir, core.CHALLENGE_CONFIG_FILE_NAME)

	log.Debug("Checking beast.toml file existance validity")
	err := utils.ValidateFileExists(configFile)
	if err != nil {
		return err
	}

	var config cfg.BeastChallengeConfig
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		return err
	}

	challengeName := filepath.Base(challengeDir)
	if challengeName != config.Challenge.Metadata.Name {
		return fmt.Errorf("Name of the challenge directory(%s) should match the name provided in the config file(%s)", challengeName, config.Challenge.Metadata.Name)
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

func getContextDirPath(dirPath string) (string, error) {
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

func GetCommandForWebLanguage(WebRoot, language string) string {
	switch language {
	case "php7.1":
		return "cd /challenge/" + WebRoot + " && php -S 0.0.0.0"
	default:
		return "ls"
	}
}

// From the provided configFIle path it generates the dockerfile for
// the challenge and returns it as a string. This function again
// assumes that the validation for the configFile is done beforehand
// before calling this function.
func GenerateDockerfile(configFile string) (string, error) {
	log.Debug("Generating dockerfile")
	var config cfg.BeastChallengeConfig
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		return "", err
	}

	relativeStaticContentDir := config.Challenge.Env.StaticContentDir
	if relativeStaticContentDir == "" {
		relativeStaticContentDir = core.PUBLIC
	}

	if config.Challenge.Env.BaseImage == "" {
		config.Challenge.Env.BaseImage = core.DEFAULT_BASE_IMAGE
	}

	RunCmd := config.Challenge.Env.RunCmd
	challengeType := config.Challenge.Metadata.Type
	var webLanguage string
	if strings.HasPrefix(challengeType, "web") {
		webLanguage = strings.Split(challengeType, ":")[1]
		webRoot := config.Challenge.Metadata.WebRoot
		RunCmd = GetCommandForWebLanguage(webRoot, webLanguage)
	}

	data := BeastBareDockerfile{
		DockerBaseImage: config.Challenge.Env.BaseImage,
		Ports:           strings.Trim(strings.Replace(fmt.Sprint(config.Challenge.Env.Ports), " ", " ", -1), "[]"),
		AptDeps:         strings.Join(config.Challenge.Env.AptDeps[:], " "),
		SetupScripts:    config.Challenge.Env.SetupScripts,
		RunCmd:          RunCmd,
		MountVolume:     filepath.Join("/challenge", relativeStaticContentDir),
		WebRoot:         filepath.Join("/challenge", config.Challenge.Env.WebRoot),
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

func updateOrCreateChallengeDbEntry(challEntry *database.Challenge, config cfg.BeastChallengeConfig) error {
	// Challenge is nil, which means the challenge entry does not exist
	// So create a new challenge entry on the basis of the fields provided
	// in the config file for the challenge.
	if challEntry.Name == "" {
		// For creating a new entry for the challenge first create an entry
		// in the challenge table using the config file.
		authorEntry, err := database.QueryFirstAuthorEntry("email", config.Author.Email)
		if err != nil {
			return fmt.Errorf("Error while querying author with email %s", config.Author.Email)
		}

		if authorEntry.Email == "" {

			rMessage := make([]byte, 128)
			rand.Read(rMessage)

			authorEntry = database.Author{
				Name:          config.Author.Name,
				Email:         config.Author.Email,
				SshKey:        config.Author.SSHKey,
				AuthChallenge: rMessage,
			}

			err = database.CreateAuthorEntry(&authorEntry)
			if err != nil {
				return fmt.Errorf("Error while creating author entry : %s", err)
			}

		} else {
			if authorEntry.Email != config.Author.Email &&
				authorEntry.SshKey != config.Author.SSHKey &&
				authorEntry.Name != config.Author.Name {
				return fmt.Errorf("ERROR, author details did not match with the ones in database")
			}
		}

		challEntry = &database.Challenge{
			Name:     config.Challenge.Metadata.Name,
			AuthorID: authorEntry.ID,
			Format:   config.Challenge.Metadata.Type,
			Status:   core.DEPLOY_STATUS["unknown"],
		}

		err = database.CreateChallengeEntry(challEntry)
		if err != nil {
			return fmt.Errorf("Error while creating chall entry with config : %s : %v", err, challEntry)
		}
	}

	allocatedPorts, err := database.GetAllocatedPorts(*challEntry)
	if err != nil {
		return fmt.Errorf("Error while getting allocated ports for : %s : %s", challEntry.Name, err)
	}

	isAllocated := func(port uint32) bool {
		for i, p := range allocatedPorts {
			if port == p.PortNo {
				allocatedPorts[len(allocatedPorts)-1], allocatedPorts[i] = allocatedPorts[i], allocatedPorts[len(allocatedPorts)-1]
				allocatedPorts = allocatedPorts[:len(allocatedPorts)-1]
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
	for _, port := range config.Challenge.Env.Ports {
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
			return err
		}

		// var gotChall database.Challenge
		// database.Db.Model(&gotPort).Related(&gotChall)

		if gotPort.ChallengeID != challEntry.ID {
			return fmt.Errorf("The port %s requested is already in use by another challenge", gotPort.PortNo)
		}
	}

	if len(allocatedPorts) > 0 {
		if err = database.DeleteRelatedPorts(allocatedPorts); err != nil {
			return fmt.Errorf("There was an error while deleting the ports which were already allocated to the challenge : %s : %s", challEntry.Name, err)
		}
	}

	return nil
}

//Provides the Static Content Folder Name from the config
func GetStaticContentDir(configFile, contextDir string) (string, error) {
	var config cfg.BeastChallengeConfig
	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		return "", err
	}
	relativeStaticContentDir := config.Challenge.Env.StaticContentDir
	if relativeStaticContentDir == "" {
		relativeStaticContentDir = core.PUBLIC
	}
	return filepath.Join(contextDir, relativeStaticContentDir), nil
}

// Provides the Web Root Folder Name from the config
func GetWebRootDir(configFile, contextDir string) (string, error) {
	var config cfg.BeastChallengeConfig

	_, err := toml.DecodeFile(configFile, &config)
	if err != nil {
		return "", err
	}

	relativeWebRootDir := config.Challenge.Env.WebRoot
	if relativeWebRootDir == "" {
		relativeWebRootDir = core.PUBLIC
	}

	return filepath.Join(contextDir, relativeWebRootDir), nil
}

//Copies the Static content to the staging/static/folder
func CopyToStaticContent(challengeName, staticContentDir string) error {
	dirPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, core.BEAST_STATIC_FOLDER)
	err := utils.CreateIfNotExistDir(dirPath)
	if err != nil {
		return err
	}
	err = utils.CopyDirectory(staticContentDir, dirPath)
	return err
}
