package deploy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/fristonio/beast/core"
	tools "github.com/fristonio/beast/templates"
	"github.com/fristonio/beast/utils"
	log "github.com/sirupsen/logrus"
)

type BeastBareDockerfile struct {
	AptDeps     string
	SetupScript string
	RunCmd      string
}

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
