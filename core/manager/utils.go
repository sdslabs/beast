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
	"github.com/sdslabs/beastv4/core/auth"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
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
	Executables     []string
	RunCmd          string
	MountVolume     string
	RunAsRoot       bool
}

type BeastXinetdConf struct {
	Port        string
	ServicePath string
	ServiceName string
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

func getCommandForWebChall(language, framework, webRoot, port string) string {
	dir := "cd " + filepath.Join("/challenge", webRoot) + " && "
	var cmd string

	switch language {
	case "php":
		switch framework {
		case "apache":
			cmd = "<some command>"
		case "cli":
			cmd = "php -S 0.0.0.0:"
		default:
			cmd = "php -S 0.0.0.0:"
		}
	case "node":
		cmd = "npm install && node server.js "
	case "python":
		switch framework {
		case "django":
			cmd = "python manage.py runserver 0.0.0.0:"
		case "flask":
			cmd = "flask run --host=0.0.0.0 --port="
		default:
			return ""
		}
	default:
		return ""
	}

	return dir + cmd + port
}

// This function provides the run command and image for a particular type of web challenge
//  * webRoot:  relative path to web challenge directory
//  * port:     web port
//  * challengeInfo
//
//  It returns the run command for challenge
//  and the docker base image corresponding to language
func GetCommandAndImageForWebChall(webRoot, port string, challengeInfo []string) (string, string) {
	length := len(challengeInfo)
	reqLength := 4

	if length < reqLength {
		for i := length; i < reqLength; i++ {
			challengeInfo = append(challengeInfo, "default")
		}
	}

	language := challengeInfo[1]
	version := challengeInfo[2]
	framework := challengeInfo[3]

	cmd := getCommandForWebChall(language, framework, webRoot, port)
	image := core.DockerBaseImageForWebChall[language][version][framework]

	return cmd, image
}

// From the provided configFIle path it generates the dockerfile for
// the challenge and returns it as a string. This function again
// assumes that the validation for the configFile is done beforehand
// before calling this function.
func GenerateDockerfile(config *cfg.BeastChallengeConfig) (string, error) {
	log.Debug("Generating dockerfile")

	relativeStaticContentDir := config.Challenge.Env.StaticContentDir
	if relativeStaticContentDir == "" {
		relativeStaticContentDir = core.PUBLIC
	}

	baseImage := config.Challenge.Env.BaseImage
	runCmd := config.Challenge.Env.RunCmd
	challengeType := config.Challenge.Metadata.Type
	aptDeps := strings.Join(config.Challenge.Env.AptDeps[:], " ")
	var runAsRoot bool = false
	var executables []string

	// The challenge type we are looking at is service. This should be deployed
	// using xinetd. The Dockerfile is different for this. Change the runCmd and the
	// apt dependencies to add xinetd.
	if challengeType == core.SERVICE_CHALLENGE_TYPE_NAME {
		runAsRoot = true
		runCmd = cfg.SERVICE_CHALL_RUN_CMD
		aptDeps = fmt.Sprintf("%s %s", cfg.SERVICE_CONTAINER_DEPS, aptDeps)
		serviceExecutable := filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, config.Challenge.Env.ServicePath)
		executables = append(executables, serviceExecutable)
	} else if strings.HasPrefix(challengeType, "web") {
		// Challenge type is web here, so set the required variables.
		challengeInfo := strings.Split(challengeType, ":")
		webPort := fmt.Sprint(config.Challenge.Env.DefaultPort)
		defaultRunCmd, webBaseImage := GetCommandAndImageForWebChall(config.Challenge.Env.WebRoot, webPort, challengeInfo)

		// runCmd can only be empty when the challenge has a web prefix.
		if runCmd == "" {
			runCmd = defaultRunCmd
		}
		baseImage = webBaseImage
	}

	if baseImage == "" {
		baseImage = core.DEFAULT_BASE_IMAGE
	}

	log.Debugf("Command type inside root[true/false] %s", runAsRoot)

	data := BeastBareDockerfile{
		DockerBaseImage: baseImage,
		Ports:           strings.Trim(strings.Replace(fmt.Sprint(config.Challenge.Env.Ports), " ", " ", -1), "[]"),
		AptDeps:         aptDeps,
		SetupScripts:    config.Challenge.Env.SetupScripts,
		RunCmd:          runCmd,
		MountVolume:     filepath.Join("/challenge", relativeStaticContentDir),
		RunAsRoot:       runAsRoot,
		Executables:     executables,
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
// The config provided must be a valid beast configuration, which is to be ensured by
// the caller. It does not check if the config even exist or is valid. The validation
// for the dockerfile should be done before calling this function. If the file does
// not exist or there is some error  while parsing the setup file
// this function will simply return the error without logging anything.
func GenerateChallengeDockerfileCtx(config *cfg.BeastChallengeConfig) (string, error) {
	log.Debug("Generating challenge dockerfile context from config")
	file, err := ioutil.TempFile("", "Dockerfile.*")
	if err != nil {
		return "", fmt.Errorf("Error while creating a tempfile for Dockerfile :: %s", err)
	}
	defer file.Close()

	dockerfile, err := GenerateDockerfile(config)
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

// Add any additional file contexts to the map additionalCtx which needs to be inside docker /challenge
// directory.
func appendAdditionalFileContexts(additionalCtx map[string]string, config *cfg.BeastChallengeConfig) error {
	log.Debug("Adding additional required file context to docker context.")
	// If the challenge type is service, we need to add xinetd configuration to the
	// docker directory context.
	if config.Challenge.Metadata.Type == core.SERVICE_CHALLENGE_TYPE_NAME {
		log.Debug("Challenge type is service, trying to embed xinetd configuration.")
		file, err := ioutil.TempFile("", "xinetd.conf.*")
		if err != nil {
			return fmt.Errorf("Error while creating a tempfile for xinetdconf :: %s", err)
		}
		defer file.Close()

		var xinetd bytes.Buffer
		xinetdTemplate, err := template.New("xinetd").Parse(tools.XINETD_CONFIGURATION_TEMPLATE)
		if err != nil {
			return fmt.Errorf("Error while parsing Xinetd config template :: %s", err)
		}

		port := config.Challenge.Env.DefaultPort
		if port == 0 {
			port = config.Challenge.Env.Ports[0]
		}

		data := BeastXinetdConf{
			Port:        fmt.Sprintf("%d", port),
			ServiceName: config.Challenge.Metadata.Name,
			ServicePath: filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, config.Challenge.Env.ServicePath),
		}
		err = xinetdTemplate.Execute(&xinetd, data)
		if err != nil {
			return fmt.Errorf("Error while executing Xinetd Config template :: %s", err)
		}

		_, err = file.WriteString(xinetd.String())
		if err != nil {
			return fmt.Errorf("Error while writing xinetd config to file :: %s", err)
		}

		log.Debugf("Successfully added xinetd config context in docker context.")
		additionalCtx[core.DEFAULT_XINETD_CONF_FILE] = file.Name()
	}

	return nil
}

// This function assumes that the staging directory exist and thus does not check for the same
// it is the responsibility of the caller to check for that. If stagingDir does not exist
// then this function will not return error, it will simply skip copying all the files.
func copyAdditionalContextToStaging(fileCtx map[string]string, stagingDir string) {
	for key, val := range fileCtx {
		filePath := filepath.Join(stagingDir, key)
		err := utils.RemoveFileIfExists(filePath)
		if err != nil {
			log.Errorf("%s", err)
			log.Errorf("Skipping copying file %s", key)
			continue
		}

		err = utils.CopyFile(val, filePath)
		if err != nil {
			log.Errorf("Error while copying %s SKIPPING : %s", key, err)
		}

	}
	log.Debugf("Copied additional context to staging directory")
}

// TODO: Refactor this.
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

		tags := make([]*database.Tag, len(config.Challenge.Metadata.Tags))

		for i, tag := range config.Challenge.Metadata.Tags {
			tags[i] = &database.Tag{
				TagName: tag,
			}
			if err = database.QueryOrCreateTagEntry(tags[i]); err != nil {
				return fmt.Errorf("Error while querying the tags for challenge(%s) : %v", config.Challenge.Metadata.Name, err)
			}
		}

		if authorEntry.Email == "" {
			// Create a new authentication challenge message for the user.
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

		*challEntry = database.Challenge{
			Name:        config.Challenge.Metadata.Name,
			AuthorID:    authorEntry.ID,
			Format:      config.Challenge.Metadata.Type,
			Status:      core.DEPLOY_STATUS["unknown"],
			ContainerId: coreUtils.GetTempContainerId(config.Challenge.Metadata.Name),
			ImageId:     coreUtils.GetTempImageId(config.Challenge.Metadata.Name),
			Flag:        config.Challenge.Metadata.Flag,
			Type:        config.Challenge.Metadata.Type,
			Sidecar:     config.Challenge.Metadata.Sidecar,
			Description: config.Challenge.Metadata.Description,
			Hint:        config.Challenge.Metadata.Hint,
		}

		err = database.CreateChallengeEntry(challEntry)
		if err != nil {
			return fmt.Errorf("Error while creating chall entry with config : %s : %v", err, challEntry)
		}

		database.Db.Model(challEntry).Association("Tags").Append(tags)
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

//Takes and save the data to transaction table
func SaveTransactionFunc(identifier string, action string, authorization string) error {
	challengeId, err := database.QueryFirstChallengeEntry("name", identifier)
	if err != nil {
		log.Infof("Error while getting challenge ID")
	}

	TransactionEntry := database.Transaction{
		Action:      action,
		UserId:      auth.GetUser(authorization),
		ChallengeID: challengeId.ID,
	}

	log.Infof("Trying %s for challenge with identifier : %s", action, identifier)
	err = database.SaveTransaction(&TransactionEntry)
	return err
}

//Copies the Static content to the staging/static/folder
func CopyToStaticContent(challengeName, staticContentDir string) error {
	dirPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_STAGING_DIR, challengeName, core.BEAST_STATIC_FOLDER)
	err := utils.CreateIfNotExistDir(dirPath)
	if err != nil {
		return err
	}

	err = utils.ValidateDirExists(staticContentDir)
	if err != nil {
		log.Warnf("%s : There is no static directory inside challenge, skipping copy.", challengeName)
		return nil
	}

	err = utils.CopyDirectory(staticContentDir, dirPath)
	return err
}
