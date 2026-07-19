package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/cr"
	"github.com/sdslabs/beastv4/utils"

	log "github.com/sirupsen/logrus"
)

var configIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
var hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
var scpGitURLPattern = regexp.MustCompile(`^[^@\s]+@[^:\s]+:[^\s]+$`)
var gitBranchPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)

// This is the global beast configuration structure
//
// # An example of a config file
//
// ```toml
// # Base OS image that beast allows the challenges to use.
// allowed_base_images = ["ubuntu:18.04", "ubuntu:16.04", "debian:jessie"]
//
// # Beast static URL refers to the host used by beast for serving static content
// # of the challenges, whenever required. It follows the URL pattern like
// # [beast_static_url]/static/[chall_dir]/[file_name]
// beast_static_url = "http://hack.sdslabs.co:8034"
//
// # For authentication purposes beast uses JWT based authentication, this is the
// # key used for encrypting the claims of a user. Keep this strong.
// jwt_secret = "beast_jwt_secret_SUPER_STRONG_0x100010000100"
//
// # List of ip addresses of all the servers where challenge could be deployed for
// # balanced load accross servers.
// [[available_servers]]
//
// # IP-address/ Hostname of the server
// host = "192.168.1.1"
//
// # Username to be used for ssh connection
// username = "user1"
//
// # Path to private SSH key for interacting with the server.
// ssh_key_path = "/path/to/your/private/key1"
//
// # Status of remote server to be used
// # If it is set to false then that remote server will not be used
// active = false
//
// # To allow beast to send notification to a notification channel povide this webhook URL
// # We are also working on implmeneting notification using Discord and IRC.
// slack_webhook = ""
//
// #Health Prober, if active starts a health prober on a thread which checks for
// #deployed challenges, containers and servers after every ticker_frequency period
// health_prober = false

// # The frequency for any periodic event in beast, the value is provided in seconds.
// # This is currently only used for health check periodic durations.
// ticker_frequency = 3000
//
// # Container default resource limits for each challenge, this can be
// # Overridden by challenge configuration beast.toml file.
// default_cpu_shares = 1024
// default_memory_limit = 1024
// default_pids_limit = 100
//
// # Configuration corresponding to the remote repository used by beast
// # We use ssh authentication mechanism for interacting with git repository.
// [remote]
//
// # URL of the remote git repository, this should be user@host:<git_repository> format
// url = "git@github.com:sdslabs/hack-test.git"
//
// # Name of the remote
// name = "hack-test"
//
// # Branch we are tracking the remote in beast.
// branch = "master"
//
// # Path to private SSH key for interacting with the git repository.
// ssh_key = "/home/fristonio/.beast/secrets/key.priv"
//
// # Mail config parameters for SMTP configuration
// from = ""
// password = ""
// smtpHost = ""
// smtpPort = ""
//
// # Configuration to connect to Postgresql database
// [psql_config]
// user = "beast"
// password = "12345678"
// dbname = "beast"
// host = "localhost"
// port = "5432"
// sslmode = "prefer"
// ```
type BeastConfig struct {
	AllowedBaseImages    []string                   `toml:"allowed_base_images"`
	AvailableServers     map[string]AvailableServer `toml:"available_servers"`
	GitRemotes           []GitRemote                `toml:"remote"`
	PsqlConf             PsqlConfig                 `toml:"psql_config"`
	RedisConf            RedisConfig                `toml:"redis_config"`
	JWTSecret            string                     `toml:"jwt_secret"`
	NotificationWebhooks []NotificationWebhook      `toml:"notification_webhooks"`
	CompetitionInfo      CompetitionInfo            `toml:"competition_info"`
	BeastStaticUrl       string                     `toml:"beast_static_url"`
	TickerFrequency      int                        `toml:"ticker_frequency"`
	HealthProber         bool                       `toml:"health_prober"`
	RemoteSyncPeriod     time.Duration              `toml:"-"`
	Rsp                  string                     `toml:"remote_sync_period"`
	InstanceConfig       InstanceConfig             `toml:"instance_config"`
	ServerConfig         ServerConfig               `toml:"server"`

	CPUShares int64   `toml:"default_cpu_shares"`
	Memory    int64   `toml:"default_memory_limit"`
	PidsLimit int64   `toml:"default_pids_limit"`
	CPUsLimit float32 `toml:"default_cpus_limit"`

	MailConfig MailConfig `toml:"mail_config"`
}

type InstanceConfig struct {
	DefaultExpiration   int64 `toml:"default_expiration"`
	MaxExtension        int64 `toml:"max_extension"`
	MaxInstancesPerUser int   `toml:"max_instances_per_user"`
}

type ServerConfig struct {
	TLSCertFile    string   `toml:"tls_cert_file"`
	TLSKeyFile     string   `toml:"tls_key_file"`
	AllowedOrigins []string `toml:"allowed_origins"`
}

func (config *ServerConfig) Validate() error {
	if err := validateAllowedOrigins(config.AllowedOrigins); err != nil {
		return err
	}
	if config.TLSCertFile == "" || config.TLSKeyFile == "" {
		return errors.New("server tls_cert_file and tls_key_file are required")
	}
	var err error
	config.TLSCertFile, err = utils.ExpandHomePath(config.TLSCertFile)
	if err != nil {
		return err
	}
	config.TLSKeyFile, err = utils.ExpandHomePath(config.TLSKeyFile)
	if err != nil {
		return err
	}
	if err := utils.ValidateFileExists(config.TLSCertFile); err != nil {
		return fmt.Errorf("invalid TLS certificate file: %w", err)
	}
	if err := utils.ValidateSecretFile(config.TLSKeyFile); err != nil {
		return fmt.Errorf("invalid TLS private key file: %w", err)
	}
	if _, err := tls.LoadX509KeyPair(config.TLSCertFile, config.TLSKeyFile); err != nil {
		return fmt.Errorf("load TLS certificate and key: %w", err)
	}
	return nil
}

func validateAllowedOrigins(origins []string) error {
	seenOrigins := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		parsed, err := url.ParseRequestURI(origin)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("invalid CORS origin %q", origin)
		}
		hostname := parsed.Hostname()
		loopback := hostname == "localhost" || net.ParseIP(hostname) != nil && net.ParseIP(hostname).IsLoopback()
		if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopback) {
			return fmt.Errorf("CORS origin must use HTTPS unless it is loopback: %q", origin)
		}
		if _, exists := seenOrigins[origin]; exists {
			return fmt.Errorf("duplicate CORS origin %q", origin)
		}
		seenOrigins[origin] = struct{}{}
	}
	return nil
}

func (config *InstanceConfig) Validate() {
	if config.DefaultExpiration <= 0 {
		config.DefaultExpiration = core.DEFAULT_MINIMUM_EXTEND_TIME
	}
	if config.MaxExtension <= config.DefaultExpiration {
		config.DefaultExpiration = core.DEFAULT_MINIMUM_EXTEND_TIME
		config.MaxExtension = core.DEFAULT_MAXIMUM_EXTEND_TIME
	}
	if config.MaxInstancesPerUser <= 0 {
		config.MaxInstancesPerUser = core.DEFAULT_MAXIMUM_INSTANCES_PER_USER
	}
}

func ValidatePortRange(portRange string) error {
	if portRange == "" {
		return fmt.Errorf("port range is empty")
	}

	firstPort, lastPort, err := utils.ParsePortMapping(portRange)
	if err != nil {
		return fmt.Errorf("error while parsing port range in global beast config: %s", err)
	}

	if firstPort > lastPort {
		return fmt.Errorf("invalid port range, %v cannot be greater than %v", firstPort, lastPort)
	}

	if firstPort < core.ALLOWED_MIN_PORT_VALUE {
		return fmt.Errorf("invalid port range, range cannot precede %v", core.ALLOWED_MIN_PORT_VALUE)
	}

	if lastPort > core.ALLOWED_MAX_PORT_VALUE {
		return fmt.Errorf("invalid port range, range cannot exceed %v", core.ALLOWED_MAX_PORT_VALUE)
	}

	return nil
}

func (config *BeastConfig) ValidateConfig() error {
	log.Debug("Validating BeastConfig structure")

	err := config.PsqlConf.ValidatePsqlConfig()
	if err != nil {
		return fmt.Errorf("error while validating db config : %s", err)
	}

	err = config.RedisConf.ValidateRedisConfig()
	if err != nil {
		return fmt.Errorf("error while validating redis config : %s", err)
	}
	if err := config.CompetitionInfo.Validate(); err != nil {
		return fmt.Errorf("validate competition info: %w", err)
	}

	if len(config.AvailableServers) == 0 {
		log.Warn("No available servers provided for challenges. Using default localhost")
		config.AvailableServers = map[string]AvailableServer{
			core.LOCALHOST: {
				Name:       core.LOCALHOST,
				Host:       core.LOCALHOST,
				Username:   os.Getenv("USER"),
				SSHKeyPath: "",
				Active:     true,
				PortRange:  fmt.Sprintf("%v%s%v", core.ALLOWED_MIN_PORT_VALUE, core.MappingDelimiter, core.ALLOWED_MAX_PORT_VALUE),
			},
		}
	}

	for name, server := range config.AvailableServers {
		if !configIdentifierPattern.MatchString(name) {
			return fmt.Errorf("server key %q is not a valid identifier", name)
		}

		server.Name = name
		config.AvailableServers[name] = server
		if server.Active {
			err := server.ValidateServerConfig()
			if err != nil {
				return fmt.Errorf("validate server %q: %w", name, err)
			}
		}
	}

	if config.BeastStaticUrl != "" {
		staticURL, err := url.ParseRequestURI(config.BeastStaticUrl)
		if err != nil || (staticURL.Scheme != "http" && staticURL.Scheme != "https") || staticURL.Host == "" {
			return fmt.Errorf("invalid beast static URL provided: %s", config.BeastStaticUrl)
		}
	}

	if !utils.StringInSlice(core.DEFAULT_BASE_IMAGE, config.AllowedBaseImages) {
		config.AllowedBaseImages = append(config.AllowedBaseImages, core.DEFAULT_BASE_IMAGE)
	}

	if len(config.JWTSecret) < 32 {
		return fmt.Errorf("jwt_secret must contain at least 32 bytes")
	}

	for _, gitRemote := range config.GitRemotes {
		if gitRemote.Active {
			err := gitRemote.ValidateGitConfig()
			if err != nil {
				return fmt.Errorf("validate git remote %q: %w", gitRemote.RemoteName, err)
			}
		}
	}
	for index := range config.NotificationWebhooks {
		if err := config.NotificationWebhooks[index].Validate(); err != nil {
			return fmt.Errorf("validate notification webhook %d: %w", index+1, err)
		}
	}

	if config.TickerFrequency <= 0 {
		log.Debug("Time is not provided or is less than equal to zero so default time is taken")
		config.TickerFrequency = core.DEFAULT_TICKER_FREQUENCY
	}

	if config.Rsp == "" {
		log.Debug("Time is not provided or is less than equal to zero so default time is taken")
		config.RemoteSyncPeriod = core.DEFAULT_REMOTE_PERIODIC_SYNC_TIME
	} else {
		duration, err := time.ParseDuration(config.Rsp)
		if err != nil || duration < core.DEFAULT_REMOTE_PERIODIC_SYNC_TIME {
			log.Debug("Invalid format of provided time or the time too less(must be > 120s) for beast remote periodic sync")
			config.RemoteSyncPeriod = core.DEFAULT_REMOTE_PERIODIC_SYNC_TIME
		} else {
			config.RemoteSyncPeriod = duration
		}
	}

	if config.CPUShares <= 0 {
		log.Debug("Per container CPU shares not provided using default value")
		config.CPUShares = core.DEFAULT_CPU_SHARE
	}

	if config.Memory <= 0 {
		log.Debug("Per container Memory Limit not provided using default value")
		config.Memory = core.DEFAULT_MEMORY_LIMIT
	}

	if config.PidsLimit <= 0 {
		log.Debug("Per container Pids Limit not provided using default value")
		config.PidsLimit = core.DEFAULT_PIDS_LIMIT
	}

	if config.CPUsLimit <= 0 {
		log.Debug("Per container CPUsLimit Limit not provided using default value")
		config.CPUsLimit = core.DEFAULT_CPU_LIMIT
	}
	if err := cr.ValidateResourceLimits(config.CPUShares, config.CPUsLimit, config.Memory, config.PidsLimit); err != nil {
		return fmt.Errorf("invalid default container resource limits: %w", err)
	}

	if config.MailConfig.From == "" || config.MailConfig.Password == "" || config.MailConfig.SMTPHost == "" || config.MailConfig.SMTPPort == "" {
		log.Warn("Mail configuration not provided, email notifications will not work")
	}

	config.InstanceConfig.Validate()
	if err := config.ServerConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (config *BeastConfig) UseLocalDockerDaemon(serverName string) bool {
	server, ok := config.AvailableServers[serverName]
	if !ok {
		return false
	}
	return server.Host == core.LOCALHOST || server.Host == core.LOCALHOST_IP
}

type AvailableServer struct {
	Name           string `toml:"-"`
	Host           string `toml:"host"`
	Username       string `toml:"username"`
	SSHKeyPath     string `toml:"ssh_key_path"`
	KnownHostsFile string `toml:"known_hosts_file"`
	Active         bool   `toml:"active"`
	PortRange      string `toml:"port_range"`
}

func (config *AvailableServer) ValidateServerConfig() error {
	if config.Host == "" {
		return fmt.Errorf("host is empty")
	}
	config.Host = strings.TrimSpace(config.Host)
	if net.ParseIP(config.Host) == nil && !hostnamePattern.MatchString(config.Host) {
		return fmt.Errorf("host %q is not a valid IP address or hostname", config.Host)
	}

	err := ValidatePortRange(config.PortRange)
	if err != nil {
		return fmt.Errorf("error while validating port range for server %s: %s", config.Host, err)
	}

	if config.Host == core.LOCALHOST || config.Host == core.LOCALHOST_IP {
		return nil
	}

	if config.Username == "" {
		return fmt.Errorf("username is empty")
	}
	if config.SSHKeyPath == "" {
		return fmt.Errorf("ssh_key_path is empty")
	}
	if config.KnownHostsFile == "" {
		config.KnownHostsFile = filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	}
	config.SSHKeyPath, err = utils.ExpandHomePath(config.SSHKeyPath)
	if err != nil {
		return err
	}
	config.KnownHostsFile, err = utils.ExpandHomePath(config.KnownHostsFile)
	if err != nil {
		return err
	}

	err = utils.ValidateSecretFile(config.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("provided ssh key file(%s) does not exists : %s", config.SSHKeyPath, err)
	}
	if err := utils.ValidateFileExists(config.KnownHostsFile); err != nil {
		return fmt.Errorf("provided known_hosts file(%s) does not exist: %s", config.KnownHostsFile, err)
	}

	return nil
}

type GitRemote struct {
	Url        string `toml:"url"`
	RemoteName string `toml:"name"`
	Branch     string `toml:"branch"`
	Secret     string `toml:"ssh_key"`
	Active     bool   `toml:"active"`
}

func (config *GitRemote) ValidateGitConfig() error {
	if config.Url == "" || config.RemoteName == "" || config.Secret == "" {
		log.Error("One of url, RemoteName or ssh_key is missing in the config")
		return errors.New("git remote config not valid, config parameters missing")
	}
	var err error
	config.Secret, err = utils.ExpandHomePath(config.Secret)
	if err != nil {
		return err
	}

	if !configIdentifierPattern.MatchString(config.RemoteName) {
		return fmt.Errorf("git remote name %q is not a valid identifier", config.RemoteName)
	}
	if config.Branch == "" {
		config.Branch = core.GIT_REMOTE_DEFAULT_BRANCH
	}
	if !gitBranchPattern.MatchString(config.Branch) || strings.Contains(config.Branch, "..") || strings.Contains(config.Branch, "@{") {
		return fmt.Errorf("git branch %q is not valid", config.Branch)
	}
	validURL := scpGitURLPattern.MatchString(config.Url)
	if parsed, err := url.Parse(config.Url); err == nil && parsed.Scheme == "ssh" && parsed.Host != "" && parsed.Path != "" {
		validURL = true
	}
	if !validURL {
		return errors.New("the provided git url is not valid")
	}

	err = utils.ValidateSecretFile(config.Secret)
	log.Debugf("Using git ssh secret : %s", config.Secret)
	if err != nil {
		return fmt.Errorf("provided ssh key file(%s) does not exists : %s", config.Secret, err)
	}

	return nil
}

type PsqlConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Dbname   string `toml:"dbname"`
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	SslMode  string `toml:"sslmode"`
}

type RedisConfig struct {
	User       string `toml:"user"`
	Password   string `toml:"password"`
	Host       string `toml:"host"`
	Port       string `toml:"port"`
	Db         uint32 `toml:"db"`
	TLS        bool   `toml:"tls"`
	CAFile     string `toml:"ca_file"`
	ServerName string `toml:"server_name"`
}

func (config *PsqlConfig) ValidatePsqlConfig() error {
	if config.User == "" || config.Password == "" || config.Dbname == "" || config.Host == "" || config.Port == "" {
		return errors.New("psql config not valid, config parameters missing")
	}
	if !configIdentifierPattern.MatchString(config.User) || !configIdentifierPattern.MatchString(config.Dbname) {
		return errors.New("psql user and database names must be safe identifiers")
	}
	if err := validateServiceAddress(config.Host, config.Port); err != nil {
		return fmt.Errorf("invalid psql address: %w", err)
	}
	if config.SslMode == "" {
		config.SslMode = "prefer"
	}
	validSSLModes := map[string]struct{}{
		"disable": {}, "allow": {}, "prefer": {}, "require": {}, "verify-ca": {}, "verify-full": {},
	}
	if _, ok := validSSLModes[config.SslMode]; !ok {
		return fmt.Errorf("unsupported psql sslmode %q", config.SslMode)
	}
	return nil
}

func (config *RedisConfig) ValidateRedisConfig() error {
	if config.Password == "" || config.Host == "" || config.Port == "" {
		return errors.New("redis config not valid, config parameters missing")
	}
	if config.User != "" && !configIdentifierPattern.MatchString(config.User) {
		return errors.New("redis user must be a safe identifier")
	}
	if err := validateServiceAddress(config.Host, config.Port); err != nil {
		return fmt.Errorf("invalid redis address: %w", err)
	}
	if !config.TLS {
		ip := net.ParseIP(config.Host)
		if config.Host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return errors.New("TLS is required for non-loopback Redis connections")
		}
		if config.CAFile != "" || config.ServerName != "" {
			return errors.New("Redis CA file and server name require TLS")
		}
		return nil
	}
	if config.ServerName != "" && net.ParseIP(config.ServerName) == nil && !hostnamePattern.MatchString(config.ServerName) {
		return fmt.Errorf("invalid Redis TLS server name %q", config.ServerName)
	}
	if config.CAFile != "" {
		caFile, err := utils.ExpandHomePath(config.CAFile)
		if err != nil {
			return fmt.Errorf("expand Redis CA file: %w", err)
		}
		config.CAFile = caFile
		if err := utils.ValidateFileExists(config.CAFile); err != nil {
			return fmt.Errorf("validate Redis CA file: %w", err)
		}
	}
	return nil
}

func validateServiceAddress(host, port string) error {
	if net.ParseIP(host) == nil && !hostnamePattern.MatchString(host) {
		return fmt.Errorf("invalid host %q", host)
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return fmt.Errorf("invalid port %q", port)
	}
	return nil
}

type NotificationWebhook struct {
	URL         string `toml:"url"`
	ServiceName string `toml:"service_name"`
	Active      bool   `toml:"active"`
}

func (webhook NotificationWebhook) Validate() error {
	if !webhook.Active {
		return nil
	}
	if len(webhook.URL) > 2048 {
		return errors.New("webhook URL is too long")
	}
	parsed, err := url.Parse(webhook.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("active webhook must use a valid HTTPS URL without credentials or fragments")
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return errors.New("webhook URL may only use the default HTTPS port")
	}
	switch webhook.ServiceName {
	case "slack":
		if parsed.Hostname() != "hooks.slack.com" || !strings.HasPrefix(parsed.EscapedPath(), "/services/") {
			return errors.New("Slack webhook must use hooks.slack.com/services")
		}
	case "discord":
		if (parsed.Hostname() != "discord.com" && parsed.Hostname() != "discordapp.com") || !strings.HasPrefix(parsed.EscapedPath(), "/api/webhooks/") {
			return errors.New("Discord webhook must use the official API webhook endpoint")
		}
	default:
		return fmt.Errorf("unsupported notification service %q", webhook.ServiceName)
	}
	return nil
}

type CompetitionInfo struct {
	Name         string `toml:"name"`
	About        string `toml:"about"`
	Prizes       string `toml:"prizes"`
	StartingTime string `toml:"starting_time"`
	EndingTime   string `toml:"ending_time"`
	TimeZone     string `toml:"timezone"`
	LogoURL      string `toml:"logo_url"`
	DynamicScore bool   `toml:"dynamic_score"`
}

func (config CompetitionInfo) Validate() error {
	windowConfigured := config.StartingTime != "" || config.EndingTime != "" || config.TimeZone != ""
	if !windowConfigured {
		if config.DynamicScore {
			return errors.New("dynamic scoring requires a competition time window")
		}
		return nil
	}
	_, _, err := config.ParseWindow()
	return err
}

func (config CompetitionInfo) ParseWindow() (time.Time, time.Time, error) {
	if config.StartingTime == "" || config.EndingTime == "" || config.TimeZone == "" {
		return time.Time{}, time.Time{}, errors.New("starting_time, ending_time, and timezone must be configured together")
	}
	locationName := strings.TrimSpace(strings.SplitN(config.TimeZone, ":", 2)[0])
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("load competition timezone %q: %w", locationName, err)
	}
	start, err := parseCompetitionTimestamp(config.StartingTime, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse starting_time: %w", err)
	}
	end, err := parseCompetitionTimestamp(config.EndingTime, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse ending_time: %w", err)
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, errors.New("ending_time must be after starting_time")
	}
	return start, end, nil
}

func parseCompetitionTimestamp(value string, location *time.Location) (time.Time, error) {
	parts := strings.SplitN(value, ",", 3)
	timeFields := strings.Fields(parts[0])
	if len(parts) < 2 || len(timeFields) == 0 {
		return time.Time{}, fmt.Errorf("invalid timestamp %q", value)
	}
	dateAndTime := strings.TrimSpace(parts[1]) + " " + timeFields[0]
	parsed, err := time.ParseInLocation("2 January 2006 15:04:05", dateAndTime, location)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

type MailConfig struct {
	From     string `toml:"from"`
	Password string `toml:"password"`
	SMTPHost string `toml:"smtpHost"`
	SMTPPort string `toml:"smtpPort"`
}

func GetCompetitionInfo() (CompetitionInfo, error) {
	if Cfg == nil {
		return CompetitionInfo{}, errors.New("beast config is not initialized")
	}
	return Cfg.CompetitionInfo, nil
}

// From the path of the config file provided as an arguement this function
// loads the parse the config file and load it into the BeastConfig
// structure. After parsing it validates the data in the config file and returns
// error if the validation fails.
func LoadBeastConfig(configPath string) (BeastConfig, error) {
	var config BeastConfig

	info, err := os.Lstat(configPath)
	if err != nil {
		return config, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return config, fmt.Errorf("global config must not be a symbolic link: %s", configPath)
	}
	if !info.Mode().IsRegular() {
		return config, fmt.Errorf("global config is not a regular file: %s", configPath)
	}
	if info.Mode().Perm() != 0600 {
		return config, fmt.Errorf("global config permissions must be 0600, got %04o", info.Mode().Perm())
	}

	err = utils.ValidateFileExists(configPath)
	if err != nil {
		return config, err
	}

	if err = decodeTOMLFileStrict(configPath, &config); err != nil {
		return config, err
	}

	err = config.ValidateConfig()
	if err != nil {
		return config, err
	}

	log.Debug("Global beast config file config.toml has been verified")
	return config, nil
}

var Cfg *BeastConfig
var NoCache bool

// InitConfig loads the config from the global config file and populates Cfg.
func InitConfig() error {
	log.Info("Loading up beast configuration.")
	if Cfg != nil {
		log.Warn("Config is already initialized; restart Beast to load configuration changes")
		return nil
	}
	configPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_CONFIG_FILE_NAME)
	cfg, err := LoadBeastConfig(configPath)
	if err != nil {
		return fmt.Errorf("load Beast global config: %w", err)
	}

	Cfg = &cfg
	return nil
}
