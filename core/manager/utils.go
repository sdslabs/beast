package manager

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sdslabs/beastv4/core"
	cfg "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	coreUtils "github.com/sdslabs/beastv4/core/utils"
	tools "github.com/sdslabs/beastv4/templates"
	"github.com/sdslabs/beastv4/utils"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

type BeastBareDockerfile struct {
	DockerBaseImage      string
	Ports                string
	AptDeps              string
	SetupScripts         []string
	Executables          []string
	RunCmd               string
	EnvironmentVariables map[string]string
	MountVolume          string
	XinetdService        bool
	RunRoot              bool
	Entrypoint           string
	SetupCommand         string
}

type BeastXinetdConf struct {
	Port        string
	ServicePath string
	ServiceName string
}

type ChallengePreview struct {
	Name            string
	Category        string
	Tags            []string
	Ports           []database.Port
	Hints           string
	Assets          []string
	AdditionalLinks []string
	Desc            string
	Points          uint
}

// This if the function which validates the challenge directory
// which is provided as an arguments. This validtions includes
// * A valid directory pointed by challengeDir
// * Valid config file for the challenge in the challengeDir root named as beast.toml
// * Valid challenge directory name in accordance to the challenge config.
func ValidateChallengeConfig(challengeDir string) error {
	configFile := filepath.Join(challengeDir, core.CHALLENGE_CONFIG_FILE_NAME)

	log.Debug("Checking beast.toml file existence validity")
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

	// log.Debugf("Parsed config file is : %s", config)
	err = config.ValidateRequiredFields(challengeDir)
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

func emptyFunction(config *BeastBareDockerfile) {

}

func getCommandAndModifierForWebChall(language, framework, webRoot, port string) (string, func(*BeastBareDockerfile)) {
	commands := make([]string, 0)
	commands = append(commands, fmt.Sprintf("cd %s", filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, webRoot)))

	modifier := emptyFunction

	switch language {
	case "php":
		switch framework {
		case "apache":
			modifier = func(config *BeastBareDockerfile) {
				if _, ok := config.EnvironmentVariables["APACHE_DOCUMENT_ROOT"]; !ok {
					config.EnvironmentVariables["APACHE_DOCUMENT_ROOT"] = filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, webRoot)
				}
				setupCommands := make([]string, 0)
				setupCommands = append(setupCommands, fmt.Sprintf("sed -ri -e 's!80!%v!g' /etc/apache2/ports.conf", port))
				setupCommands = append(setupCommands, fmt.Sprintf("sed -ri -e 's!80!%v!g' /etc/apache2/sites-enabled/000-default.conf", port))
				setupCommands = append(setupCommands, "sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-enabled/000-default.conf /etc/apache2/sites-available/*.conf")
				setupCommands = append(setupCommands, "sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf")
				config.SetupCommand = strings.Join(setupCommands, " && ")
				config.RunRoot = true
			}
			commands = append(commands, "docker-php-entrypoint apache2-foreground")
		case "nginx":
			modifier = func(config *BeastBareDockerfile) {
				setupCommands := make([]string, 0)
				setupCommands = append(setupCommands, "cp /usr/local/etc/php/php.ini-production /usr/local/etc/php/php.ini")
				setupCommands = append(setupCommands, "sed -i -e 's!listen = 9000!listen = /var/run/php-fpm.sock!g' /usr/local/etc/php-fpm.d/zz-docker.conf")
				setupCommands = append(setupCommands, "echo 'listen.mode = 0666' >> /usr/local/etc/php-fpm.d/zz-docker.conf")
				setupCommands = append(setupCommands, fmt.Sprintf("sed -i -e 's!listen 80 default_server;!listen %v;!g' /etc/nginx/sites-available/default", port))
				setupCommands = append(setupCommands, fmt.Sprintf("sed -i -e 's!listen [::]:80 default_server;!listen [::]:%v;!g' /etc/nginx/sites-available/default", port))
				setupCommands = append(setupCommands, fmt.Sprintf("sed -i -e 's!root /var/www/html;!root %v;!g' /etc/nginx/sites-available/default", filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, webRoot)))
				setupCommands = append(setupCommands, "sed -i -e 's!index index.html!index index.php index.html!g' /etc/nginx/sites-available/default")
				setupCommands = append(setupCommands, `sed -i -e 's!#location ~ \\\\\\.php$ {!location ~ \\\\\\.php$ {include snippets/fastcgi-php.conf;fastcgi_pass unix:/var/run/php-fpm.sock;}!g' /etc/nginx/sites-available/default`)

				config.SetupCommand = strings.Join(setupCommands, " && ")
				config.AptDeps = config.AptDeps + " nginx"
				config.RunRoot = true
			}
			commands = append(commands, "php-fpm -D")
			commands = append(commands, "/etc/init.d/nginx start")
			commands = append(commands, "exec tail -f /var/log/nginx/*")

		default:
			commands = append(commands, fmt.Sprintf("php -S 0.0.0.0:%v", port))
		}
	case "node":
		commands = append(commands, "npm install")
		commands = append(commands, fmt.Sprintf("node server.js %v", port))
	case "python":
		switch framework {
		case "django":
			commands = append(commands, fmt.Sprintf("python manage.py runserver 0.0.0.0:%v", port))
		case "flask":
			commands = append(commands, fmt.Sprintf("flask run --host=0.0.0.0 --port=%v", port))
		default:
			return "", modifier
		}
	default:
		return "", modifier
	}

	return strings.Join(commands, " && "), modifier
}

// This function provides the run command and image for a particular type of web challenge
//  * webRoot:  relative path to web challenge directory
//  * port:     web port
//  * challengeInfo
//
//  It returns the run command for challenge
//  and the docker base image corresponding to language
func GetWebChallSetup(webRoot, port string, challengeInfo []string) (string, string, func(*BeastBareDockerfile)) {
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

	cmd, modifier := getCommandAndModifierForWebChall(language, framework, webRoot, port)
	image := core.DockerBaseImageForWebChall[language][version][framework]

	return cmd, image, modifier
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
	entrypoint := config.Challenge.Env.Entrypoint
	setupScripts := config.Challenge.Env.SetupScripts
	aptDeps := strings.Join(config.Challenge.Env.AptDeps[:], " ")
	var xinetdService bool = false
	var executables []string
	modifier := emptyFunction

	// The challenge type we are looking at is service. This should be deployed
	// using xinetd. The Dockerfile is different for this. Change the runCmd and the
	// apt dependencies to add xinetd.
	if challengeType == core.SERVICE_CHALLENGE_TYPE_NAME {
		xinetdService = true
		runCmd = cfg.SERVICE_CHALL_RUN_CMD
		aptDeps = fmt.Sprintf("%s %s", cfg.SERVICE_CONTAINER_DEPS, aptDeps)
		serviceExecutable := filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, config.Challenge.Env.ServicePath)
		executables = append(executables, serviceExecutable)
	} else if strings.HasPrefix(challengeType, "web") {
		// Challenge type is web here, so set the required variables.
		challengeInfo := strings.Split(challengeType, ":")
		webPort := fmt.Sprint(config.Challenge.Env.DefaultPort)
		var defaultRunCmd string
		defaultRunCmd, baseImage, modifier = GetWebChallSetup(config.Challenge.Env.WebRoot, webPort, challengeInfo)

		// runCmd can only be empty when the challenge has a web prefix.
		if runCmd == "" {
			runCmd = defaultRunCmd
		}
	}

	if baseImage == "" {
		baseImage = core.DEFAULT_BASE_IMAGE
	}

	log.Debugf("Command type inside root[true/false] %v", xinetdService)

	if entrypoint != "" {
		entrypoint = filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, entrypoint)
	}

	data := BeastBareDockerfile{
		DockerBaseImage:      baseImage,
		Ports:                strings.Trim(strings.Replace(fmt.Sprint(config.Challenge.Env.Ports), " ", " ", -1), "[]"),
		AptDeps:              aptDeps,
		SetupScripts:         setupScripts,
		RunCmd:               runCmd,
		MountVolume:          filepath.Join(core.BEAST_DOCKER_CHALLENGE_DIR, relativeStaticContentDir),
		XinetdService:        xinetdService,
		RunRoot:              xinetdService,
		Executables:          executables,
		Entrypoint:           entrypoint,
		EnvironmentVariables: map[string]string{},
	}

	modifier(&data)

	var dockerfile bytes.Buffer
	log.Debugf("Preparing dockerfile template")
	dockerfileTemplate, err := template.New("dockerfile").Parse(tools.BEAST_DOCKERFILE_TEMPLATE)
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

		port := config.Challenge.Env.GetDefaultPort()

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
func UpdateOrCreateChallengeDbEntry(challEntry *database.Challenge, config cfg.BeastChallengeConfig) error {
	// Challenge is nil, which means the challenge entry does not exist
	// So create a new challenge entry on the basis of the fields provided
	// in the config file for the challenge.
	if challEntry.Name == "" {
		// For creating a new entry for the challenge first create an entry
		// in the challenge table using the config file.
		userEntry, err := database.QueryFirstUserEntry("email", config.Author.Email)
		if err != nil {
			return fmt.Errorf("Error while querying user with email %s", config.Author.Email)
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

		users := make([]*database.User, len(config.Maintainers)+1)

		for i, user := range config.Maintainers {
			u, err := database.QueryFirstUserEntry("email", user.Email)
			if err != nil {
				return fmt.Errorf("Error while querying user with email %s", user.Email)
			}
			users[i] = &u
		}

		if userEntry.Email == "" {
			return fmt.Errorf("User with the given email does not exist : %v", config.Author.Email)
		} else {
			if userEntry.Email != config.Author.Email &&
				(userEntry.SshKey != config.Author.SSHKey || config.Author.SSHKey == "") &&
				(userEntry.Name != config.Author.Name || config.Author.Name == "") &&
				userEntry.Role != core.USER_ROLES["author"] {
				return fmt.Errorf("ERROR, author details for %s did not match with the ones in database", userEntry.Email)
			}
		}

		users[len(config.Maintainers)] = &userEntry

		var assetsURL = make([]string, len(config.Challenge.Metadata.Assets))

		for index, asset := range config.Challenge.Metadata.Assets {
			beastStaticAssetUrl, _ := url.Parse(cfg.Cfg.BeastStaticUrl)
			beastStaticAssetUrl.Path = path.Join(beastStaticAssetUrl.Path, config.Challenge.Metadata.Name, core.BEAST_STATIC_FOLDER, asset)
			assetsURL[index] = beastStaticAssetUrl.String()
		}

		*challEntry = database.Challenge{
			Name:        config.Challenge.Metadata.Name,
			AuthorID:    userEntry.ID,
			Format:      config.Challenge.Metadata.Type,
			Status:      core.DEPLOY_STATUS["undeployed"],
			ContainerId: coreUtils.GetTempContainerId(config.Challenge.Metadata.Name),
			ImageId:     coreUtils.GetTempImageId(config.Challenge.Metadata.Name),
			Flag:        config.Challenge.Metadata.Flag,
			Type:        config.Challenge.Metadata.Type,
			Sidecar:     config.Challenge.Metadata.Sidecar,
			Description: config.Challenge.Metadata.Description,
			Hints:       strings.Join(config.Challenge.Metadata.Hints, core.DELIMITER),
			Assets:      strings.Join(assetsURL, core.DELIMITER),
			Points:      config.Challenge.Metadata.Points,
		}

		err = database.CreateChallengeEntry(challEntry)
		if err != nil {
			return fmt.Errorf("Error while creating chall entry with config : %s : %v", err, challEntry)
		}

		database.Db.Model(challEntry).Association("Tags").Append(tags)

		database.Db.Model(challEntry).Association("Users").Append(users)
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

	hostPorts, err := config.Challenge.Env.GetAllHostPorts()
	if err != nil {
		return fmt.Errorf("Error while parsing host port for challenge %s : %s", challEntry.Name, err)
	}
	// Once the challenge entry has been created, add entries to the ports
	// table in the database with the ports to expose
	// for the challenge.
	// TODO: Do all this under a database transaction so that if any port
	// request is not available
	for _, port := range hostPorts {
		if isAllocated(port) {
			// The port has already been allocated to the challenge
			// Do nothing for this.
			continue
		}

		portEntry := database.Port{
			ChallengeID: challEntry.ID,
			PortNo:      port,
		}

		gotPort, err := database.PortEntryGetOrCreate(&portEntry)
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
		return "", fmt.Errorf("Error while decoding file : %s", configFile)
	}
	relativeStaticContentDir := config.Challenge.Env.StaticContentDir
	if relativeStaticContentDir == "" {
		relativeStaticContentDir = core.PUBLIC
	}
	return filepath.Join(contextDir, relativeStaticContentDir), nil
}

//Takes and save the data to transaction table
func LogTransaction(identifier string, action string, authorization string) error {
	log.Debugf("Logging transaction for %s on %s", action, identifier)

	challenge, err := database.QueryFirstChallengeEntry("name", identifier)
	if err != nil {
		return fmt.Errorf("Error while querying challenge: %s", identifier)
	}

	// We are trying to get the username for the request from JWT claims here
	// Since upto this point the request is already authorized, we use a default
	// username if any error occurs while getting the username.
	userName, err := coreUtils.GetUser(authorization)
	if err != nil {
		log.Warnf("Error while getting user from authorization header, using default user(since already authorized)")
		userName = core.DEFAULT_USER_NAME
	}

	user, err := database.QueryFirstUserEntry("username", userName)
	if err != nil {
		return fmt.Errorf("Error while querying user corresponding to request: %s", err)
	}

	TransactionEntry := database.Transaction{
		Action:      action,
		UserID:      user.ID,
		ChallengeID: challenge.ID,
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
		return fmt.Errorf("Error while copying static content : %v", err)
	}

	err = utils.ValidateDirExists(staticContentDir)
	if err != nil {
		log.Warnf("%s : There is no static directory inside challenge, skipping copy.", challengeName)
		return nil
	}

	err = utils.CopyDirectory(staticContentDir, dirPath)
	return err
}

func GetAvailableChallenges() ([]string, error) {
	var challsNameList []string

	for _, gitRemote := range cfg.Cfg.GitRemotes {
		if gitRemote.Active == true {
			challengesDirRoot := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR, gitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR)

			err, challenges := utils.GetDirsInDir(challengesDirRoot)
			if err != nil {
				log.Errorf("Error while getting available challenges : %s", err)
				return nil, err
			}

			for _, chall := range challenges {
				challsNameList = append(challsNameList, chall)
			}
		}
	}
	return challsNameList, nil
}

// ExtractChallengeNamesFromFileNames extracts names of challenges from an array of filenames (git style file paths)
func ExtractChallengeNamesFromFileNames(fileNames []string) []string {
	var challengeNames []string
	set := utils.EmptySet() // A set to help avoid duplicate entries
	for _, fileName := range fileNames {
		filePathArr := strings.Split(fileName, "/")
		if len(filePathArr) > 2 && filePathArr[0] == core.BEAST_REMOTE_CHALLENGE_DIR {
			challengeName := filePathArr[1]
			if !set.Contains(challengeName) {
				challengeNames = append(challengeNames, challengeName)
				set.Add(challengeName)
			}
		}
	}

	return challengeNames
}

//UnTars challenge folder in a destination directory
func UnTarChallengeFolder(tarContextPath, dstPath string) (string, error) {
	baseFileName := filepath.Base(tarContextPath)
	targetDir := filepath.Join(dstPath, strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName)))

	reader, err := os.Open(tarContextPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		path := filepath.Join(filepath.Join(targetDir), header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return "", err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return "", err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return "", err
		}
	}
	return targetDir, nil
}

// File copies a single file from src to dst
func CopyFile(src, dst string) error {
	var err error
	var srcfd *os.File
	var dstfd *os.File
	var srcinfo os.FileInfo

	if srcfd, err = os.Open(src); err != nil {
		return err
	}
	defer srcfd.Close()

	if dstfd, err = os.Create(dst); err != nil {
		return err
	}
	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		return err
	}
	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}
	return os.Chmod(dst, srcinfo.Mode())
}

// Dir copies a whole directory recursively
func CopyDir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := path.Join(src, fd.Name())
		dstfp := path.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = CopyDir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = CopyFile(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

func UpdateChallenges() {
	beastRemoteDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_REMOTES_DIR)

	for _, gitRemote := range cfg.Cfg.GitRemotes {
		if !gitRemote.Active {
			continue
		}

		challengesDir := filepath.Join(beastRemoteDir, gitRemote.RemoteName, core.BEAST_REMOTE_CHALLENGE_DIR)
		depthChall := strings.Count(challengesDir,string(os.PathSeparator))+1;
		dirs := utils.GetAllDirectoriesNameTillDepth(challengesDir, depthChall)
		
		uploadsDir := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_UPLOADS_DIR)
		depthUploads := strings.Count(uploadsDir,string(os.PathSeparator))+1
		uploadedChalls := utils.GetAllDirectoriesNameTillDepth(uploadsDir, depthUploads)
		
		dirs = append(dirs, uploadedChalls...)

		for _, dir := range dirs {

			configFile := filepath.Join(dir, core.CHALLENGE_CONFIG_FILE_NAME)
			var config cfg.BeastChallengeConfig
			_, err := toml.DecodeFile(configFile, &config)
			if err != nil {
				log.Errorf("Error while decoding challenge config file: %s", err.Error())
				continue
			}
			challengeName := config.Challenge.Metadata.Name

			err = config.ValidateRequiredFields(dir)
			if err != nil {
				log.Errorf("Error while validating required fields in the challenge directory %s : %s", challengeName, err)
				continue
			}

			// Validate challenge directory name with the name of the challenge
			// provided in the config file for the beast. There should be no
			// conflict in the name.
			if challengeName != config.Challenge.Metadata.Name {
				log.Errorf("Name of the challenge directory(%s) should match the name provided in the config file(%s)",
					challengeName,
					config.Challenge.Metadata.Name)
				continue
			}

			challenge, err := database.QueryFirstChallengeEntry("name", config.Challenge.Metadata.Name)
			if err != nil {
				log.Errorf("Error while querying challenge %s : %s", config.Challenge.Metadata.Name, err)
				continue
			}

			// Using the challenge dir we got, update the database entries for the challenge.
			err = UpdateOrCreateChallengeDbEntry(&challenge, config)
			if err != nil {
				log.Errorf("An error occured while creating db entry for challenge :: %s", challengeName)
				log.Errorf("Db error : %s", err)
				continue
			}

		}
	}
	log.Debugf("Challenges updated in Db")
}
