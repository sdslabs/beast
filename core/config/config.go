package config

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	models "github.com/sdslabs/beastv4/core/database"
)

// Duration is a wrapper around time.Duration supporting TOML parsing.
type Duration time.Duration

func (d Duration) Std() time.Duration { return time.Duration(d) }

func (d Duration) String() string { return time.Duration(d).String() }

func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// Secret is a string wrapper that prevents sensitive credentials (passwords, tokens, keys)
// from leaking through fmt formatting, %#v dumps, JSON serialization, or slog logging.
type Secret string

// String returns "[redacted]" to prevent leaks in %s, %v, %+v fmt calls.
func (s Secret) String() string {
	if s == "" {
		return ""
	}
	return "[redacted]"
}

// GoString returns "[redacted]" to prevent leaks in %#v fmt calls.
func (s Secret) GoString() string {
	if s == "" {
		return `""`
	}
	return `"[redacted]"`
}

// LogValue implements slog.LogValuer so structured loggers automatically redact the secret.
func (s Secret) LogValue() slog.Value {
	if s == "" {
		return slog.StringValue("")
	}
	return slog.StringValue("[redacted]")
}

// MarshalText implements encoding.TextMarshaler to prevent accidental exposure during serialization.
func (s Secret) MarshalText() ([]byte, error) {
	if s == "" {
		return []byte(""), nil
	}
	return []byte("[redacted]"), nil
}

// UnmarshalText implements encoding.TextUnmarshaler so TOML and text decoders decode into Secret.
func (s *Secret) UnmarshalText(text []byte) error {
	*s = Secret(text)
	return nil
}

// Expose returns the underlying plaintext string when the secret is needed (e.g. authenticating).
func (s Secret) Expose() string {
	return string(s)
}

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

func (e Environment) IsProduction() bool {
	return e == EnvProduction
}

// Cfg is the active global Beast configuration instance.
var Cfg *BeastConfig

type BeastConfig struct {
	Environment   Environment         `toml:"environment"`
	Server        ServerConfig        `toml:"server"`
	Auth          AuthConfig          `toml:"auth"`
	OTP           OTPConfig           `toml:"otp"`
	RateLimit     RateLimitConfig     `toml:"rate_limit"`
	Logging       LoggingConfig       `toml:"logging"`
	Audit         AuditConfig         `toml:"audit"`
	Docs          DocsConfig          `toml:"docs"`
	Storage       StorageConfig       `toml:"storage"`
	Runtime       RuntimeConfig       `toml:"runtime"`
	Instance      InstanceConfig      `toml:"instance_config"`
	Generator     GeneratorConfig     `toml:"generator"`
	Submission    SubmissionConfig    `toml:"submission"`
	Notifications NotificationsConfig `toml:"notifications"`

	PsqlConf   PsqlConfig  `toml:"psql_config"`
	RedisConf  RedisConfig `toml:"redis_config"`
	MailConfig MailConfig  `toml:"mail_config"`

	NotificationWebhooks []NotificationWebhook `toml:"notification_webhooks"`

	TickerInterval Duration `toml:"ticker_interval"`
	HealthProber   bool     `toml:"health_prober"`
}

type ServerConfig struct {
	Host               string   `toml:"host"`
	Port               int      `toml:"port"`
	PublicURL          string   `toml:"public_url"`
	CORSAllowedOrigins []string `toml:"cors_allowed_origins"`

	// Proxies whose X-Forwarded-For is trusted. Without this, submissions.ip_address,
	// auth_sessions.ip_address and the rate limiters see the proxy's IP.
	TrustedProxies []string `toml:"trusted_proxies"`

	// Header a reverse proxy sets with the JA3/TLS fingerprint. Stored in
	// submissions.tls_fingerprint. Empty disables the capture.
	TLSFingerprintHeader string `toml:"tls_fingerprint_header"`

	ReadHeaderTimeout Duration `toml:"read_header_timeout"`
	ReadTimeout       Duration `toml:"read_timeout"`
	// WriteTimeout must be 0 (or SSE must reset its own deadline), otherwise
	// /notifications/stream is cut off. Keep 0 unless you handle that.
	WriteTimeout    Duration `toml:"write_timeout"`
	IdleTimeout     Duration `toml:"idle_timeout"`
	ShutdownTimeout Duration `toml:"shutdown_timeout"`

	MaxBodyBytes int64 `toml:"max_body_bytes"` // default cap for JSON bodies
}

// AuthConfig covers tokens, sessions and passwords.
type AuthConfig struct {
	JWTSecret Secret `toml:"jwt_secret"` // prefer env BEAST_JWT_SECRET
	JWTIssuer string `toml:"jwt_issuer"`

	AccessTokenTTL  Duration `toml:"access_token_ttl"`  // e.g. "15m"
	RefreshTokenTTL Duration `toml:"refresh_token_ttl"` // e.g. "168h"; stored in auth_sessions.expires_at

	// Oldest session is revoked when exceeded. 0 = unlimited.
	MaxSessionsPerUser int `toml:"max_sessions_per_user"`

	// Bcrypt work factor (4-15). Defaults to 12. Values > 15 cause severe CPU exhaustion.
	BcryptCost        int `toml:"bcrypt_cost"`
	MinPasswordLength int `toml:"min_password_length"`

	// Enables the login_2fa OTP flow (auth_otps.type = 'login_2fa').
	Login2FAEnabled bool `toml:"login_2fa_enabled"`

	// RequireEmailVerification blocks unverified users from logging in or competing.
	RequireEmailVerification bool `toml:"require_email_verification"`

	// PasswordResetEnabled enables self-service password reset via email OTP.
	PasswordResetEnabled bool `toml:"password_reset_enabled"`

	// Bearer token for operatorAuthorize (/readyz). Prefer env BEAST_OPERATOR_TOKEN.
	OperatorToken Secret `toml:"operator_token"`

	// SSH public keys (users.public_ssh_key) are written here.
	AuthorizedKeysFile string `toml:"authorized_keys_file"`
}

// OTPConfig is the shared policy for rows in auth_otps.
type OTPConfig struct {
	CodeLength           int      `toml:"code_length"` // digits, 6 by default
	EmailVerificationTTL Duration `toml:"email_verification_ttl"`
	PasswordResetTTL     Duration `toml:"password_reset_ttl"`
	Login2FATTL          Duration `toml:"login_2fa_ttl"`
	MaxAttempts          int      `toml:"max_attempts"`
	ResendCooldown       Duration `toml:"resend_cooldown"`
}

// RateLimitConfig holds per-IP limits for unauthenticated endpoints.
// Flag submissions are limited by competition_config.submission_rate_limit_per_min.
type RateLimitConfig struct {
	Enabled         bool `toml:"enabled"`
	LoginPerMin     int  `toml:"login_per_min"`
	RegisterPerMin  int  `toml:"register_per_min"`
	OTPRequestPerHr int  `toml:"otp_request_per_hour"` // password reset + e-mail verification requests
	GlobalPerMin    int  `toml:"global_per_min"`       // 0 = no global limit
}

// LoggingConfig replaces the LOG_HTTP_PAYLOADS env var and the hard-coded
// sensitive-path list in middleware/logging.go.
type LoggingConfig struct {
	Level  string `toml:"level"`  // debug | info | warn | error
	Format string `toml:"format"` // text | json

	LogHTTPPayloads bool `toml:"log_http_payloads"`
	MaxPayloadBytes int  `toml:"max_payload_bytes"`

	// Bodies on these path prefixes are never logged, only "[redacted]".
	RedactedPathPrefixes []string `toml:"redacted_path_prefixes"`
	// Paths that are never buffered (streams, binary transfers).
	UnbufferedPathSuffixes []string `toml:"unbuffered_path_suffixes"`
}

// AuditConfig controls the audit_events table.
type AuditConfig struct {
	Enabled bool `toml:"enabled"`
	// Rows older than this are purged by the ticker. 0 = keep forever.
	Retention Duration `toml:"retention"`
}

// DocsConfig protects /docs and /openapi.json (docsAuthorize).
type DocsConfig struct {
	Enabled  bool   `toml:"enabled"`
	Password Secret `toml:"password"` // prefer env BEAST_DOCS_PASSWORD
}

// StorageConfig covers everything written to disk by the API process.
type StorageConfig struct {
	GlobalDir string `toml:"global_dir"` // e.g. ~/.beast

	MaxLogoBytes          int64    `toml:"max_logo_bytes"`
	AllowedLogoTypes      []string `toml:"allowed_logo_types"` // MIME types, e.g. "image/png"
	MaxChallengeFileBytes int64    `toml:"max_challenge_file_bytes"`
}

// TODO : review of this is required, i dont know much about this
// NetworkPolicyConfig defines container network sandboxing, isolation, and egress rules.
//
// Threat Model & Security Architecture:
//  1. Inter-Challenge Isolation: Every challenge (and per-team dynamic instance) runs on
//     its own dedicated Docker bridge network ("beast-<slug>-<id>"). Containers from
//     different challenges or teams cannot sniff or talk to each other.
//  2. Outbound Egress (RestrictOutbound: true): Creates the Docker network with internal: true.
//     This completely severs external WAN routing, preventing compromised containers from
//     launching DDoS attacks, mining cryptocurrency, or exfiltrating data to external C2.
//  3. DNS Caveat: When RestrictOutbound is true, public resolvers (like 1.1.1.1/8.8.8.8)
//     are unreachable from within the container. DNSResolvers must either be empty/nil
//     (using Docker's internal embedded 127.0.0.11 daemon resolver) or point to an internal IP.
//  4. Host Gateway & Link-Local Defense: internal: true does NOT prevent containers from
//     connecting to their bridge default gateway (the host IP). If services (like Beast API on
//     0.0.0.0:5005, or unauthenticated internal daemons) bind to 0.0.0.0, containers can reach them.
//     BlockHostAccess drops egress traffic targeting the host bridge gateway and cloud
//     link-local metadata endpoints (169.254.169.254 on AWS/GCP/DigitalOcean).
type NetworkPolicyConfig struct {
	// Prefix for per-challenge isolated Docker networks.
	Prefix string `toml:"prefix"`

	// RestrictOutbound creates the challenge network with internal: true.
	// This blocks direct outbound internet access from containers on that network.
	RestrictOutbound bool `toml:"restrict_outbound"`

	// BlockHostAccess drops egress traffic from the container to the host bridge gateway
	// and link-local metadata services (169.254.169.254).
	BlockHostAccess bool `toml:"block_host_access"`

	// EnableICC controls inter-container communication on the bridge.
	// Defaults to true so multi-container challenges (web <-> db) can communicate.
	// Inter-challenge isolation is preserved because each challenge has a dedicated network.
	EnableICC bool `toml:"enable_icc"`

	// BindIP is the host IP address used when publishing a challenge port.
	// Use "127.0.0.1" when traffic must pass through a reverse proxy;
	// use "0.0.0.0" only if direct host-port exposure is intentional.
	BindIP string `toml:"bind_ip"`

	// DNSResolvers overrides container DNS servers.
	// Leave empty/nil to use Docker's embedded 127.0.0.11 resolver.
	// NOTE: When RestrictOutbound is true, public IPs (e.g. 1.1.1.1, 8.8.8.8) are unreachable.
	// Any configured resolvers must be internal/private addresses.
	DNSResolvers []string `toml:"dns_resolvers"`

	// Optional IP supernet pool from which per-challenge subnets are allocated
	// (e.g. "172.30.0.0/16"). Leave empty to let Docker manage subnet pools.
	AddressPool string `toml:"address_pool"`

	// Optional MTU for networks behind VPNs, overlays, or constrained links (0 = Docker default).
	MTU int `toml:"mtu"`

	// Optional network driver (defaults to "bridge").
	Driver string `toml:"driver"`

	// Optional labels attached to created networks, useful for cleanup,
	// observability, and identifying challenge-owned resources.
	Labels map[string]string `toml:"labels"`
}

// RuntimeConfig covers Docker defaults and image policy
// (challenge_runtime_config: image_ref / compose_path).
type RuntimeConfig struct {
	// if empty every thing is allowed
	AllowedBaseImages []string `toml:"allowed_base_images"`

	// Registries image_ref may point to. Empty = any registry.
	AllowedRegistries []string `toml:"allowed_registries"`
	// When true, Beast rejects image tags like :latest or :v1 and strictly requires an immutable
	// SHA256 digest (e.g., ubuntu@sha256:7cc0576...). This guarantees reproducible builds and
	// protects against upstream image tampering
	RequireDigestPinning bool `toml:"require_digest_pinning"`

	CPUShares int64   `toml:"default_cpu_shares"`
	Memory    int64   `toml:"default_memory_limit"`
	PidsLimit int64   `toml:"default_pids_limit"`
	CPUsLimit float64 `toml:"default_cpus_limit"`

	// Network encapsulates all network default sandboxing policies.
	Network NetworkPolicyConfig `toml:"network"`

	PullTimeout  Duration `toml:"pull_timeout"`
	StartTimeout Duration `toml:"start_timeout"` // starting -> running
	StopTimeout  Duration `toml:"stop_timeout"`  // grace before kill
}

// InstanceConfig is the global default for per-user instances and shared
// deployments. Per-challenge overrides live in challenge_instance_config.
type InstanceConfig struct {
	// Defaults for challenge_instance_config.ttl_minutes / max_renewals.
	DefaultTTL         Duration `toml:"default_ttl"`
	DefaultMaxRenewals int      `toml:"default_max_renewals"`

	// challenge_deployments.owner_user_id is a user, so the cap is per user.
	MaxActivePerUser int `toml:"max_active_per_user"`
	MaxActiveTotal   int `toml:"max_active_total"` // 0 = unlimited

	// Host ports handed to instances on a node.
	PortRangeStart int `toml:"port_range_start"`
	PortRangeEnd   int `toml:"port_range_end"`

	// Hostname written into deployment_endpoints.host when a node has none.
	PublicHost string `toml:"public_host"`

	// Worker path under which challenge_deployments.instance_dir is created.
	InstanceRoot string `toml:"instance_root"`

	// How often expired deployments are stopped (status = expired).
	ReaperInterval Duration `toml:"reaper_interval"`
	// How often rows with cleaned_up_at IS NULL are re-checked against Docker.
	ReconcileInterval Duration `toml:"reconcile_interval"`
	// Retry delay for failed cleanup (errored deployments keep their slot).
	CleanupRetryInterval Duration `toml:"cleanup_retry_interval"`
}

// GeneratorConfig covers per-team flag/file generators (generate.py).
type GeneratorConfig struct {
	// OutputDir is relative to StorageConfig.GlobalDir.
	// Defaults to "data/generated_files" (matching ~/.beast/data/generated_files).
	OutputDir string `toml:"output_dir"`

	DefaultTimeout Duration `toml:"default_timeout"` // used when a challenge has no value
	MaxTimeout     Duration `toml:"max_timeout"`     // upper bound for generator_timeout_s

	MaxOutputBytes int64 `toml:"max_output_bytes"`
	MaxConcurrent  int   `toml:"max_concurrent"`
}

// SubmissionConfig holds limits that are not in competition_config.
type SubmissionConfig struct {
	// Reject longer values before hashing; submissions.submitted_flag stores the raw text.
	MaxFlagLength int `toml:"max_flag_length"`
}

// NotificationsConfig covers /notifications/stream (SSE).
type NotificationsConfig struct {
	HeartbeatInterval Duration `toml:"heartbeat_interval"` // keeps proxies from closing idle streams
	MaxStreamsPerUser int      `toml:"max_streams_per_user"`
	// How far back Last-Event-ID can replay missed notifications.
	ReplayWindow Duration `toml:"replay_window"`
}

// DefaultConfig returns a config with safe defaults. Decode the TOML file into
// it, so keys missing from the file keep these values:
//
//	cfg := config.DefaultConfig()
//	toml.DecodeFile(path, &cfg)
//	cfg.ApplyEnvOverrides()
//	if err := cfg.Validate(); err != nil { ... }
func DefaultConfig() BeastConfig {
	return BeastConfig{
		Environment: EnvProduction,

		Server: ServerConfig{
			Host:              "0.0.0.0",
			Port:              5005,
			ReadHeaderTimeout: Duration(10 * time.Second),
			ReadTimeout:       Duration(30 * time.Second),
			WriteTimeout:      0,
			IdleTimeout:       Duration(120 * time.Second),
			ShutdownTimeout:   Duration(15 * time.Second),
			MaxBodyBytes:      1 << 20,
		},

		Auth: AuthConfig{
			JWTIssuer:          "beast",
			AccessTokenTTL:     Duration(15 * time.Minute),
			RefreshTokenTTL:    Duration(7 * 24 * time.Hour),
			MaxSessionsPerUser: 10,
			BcryptCost:         12,
			MinPasswordLength:  8,
		},

		OTP: OTPConfig{
			CodeLength:           6,
			EmailVerificationTTL: Duration(24 * time.Hour),
			PasswordResetTTL:     Duration(15 * time.Minute),
			Login2FATTL:          Duration(5 * time.Minute),
			MaxAttempts:          5,
			ResendCooldown:       Duration(60 * time.Second),
		},

		RateLimit: RateLimitConfig{
			Enabled:         true,
			LoginPerMin:     10,
			RegisterPerMin:  5,
			OTPRequestPerHr: 10,
			GlobalPerMin:    0,
		},

		Logging: LoggingConfig{
			Level:           "info",
			Format:          "text",
			LogHTTPPayloads: false,
			MaxPayloadBytes: 8 * 1024,
			RedactedPathPrefixes: []string{
				"/api/v1/auth/",
				"/api/v1/me/password",
			},
			UnbufferedPathSuffixes: []string{
				"/notifications/stream",
				"/competition/logo",
			},
		},

		Audit: AuditConfig{
			Enabled:   true,
			Retention: 0,
		},

		Docs: DocsConfig{Enabled: false},

		Storage: StorageConfig{
			MaxLogoBytes:          2 << 20,
			AllowedLogoTypes:      []string{"image/png", "image/jpeg", "image/webp"},
			MaxChallengeFileBytes: 512 << 20,
		},

		Runtime: RuntimeConfig{
			RequireDigestPinning: true,
			CPUShares:            512,
			Memory:               512 << 20,
			PidsLimit:            256,
			CPUsLimit:            1,
			Network: NetworkPolicyConfig{
				Prefix:           "beast-",
				RestrictOutbound: true,
				EnableICC:        true,
				BindIP:           "0.0.0.0",
				DNSResolvers:     nil,
				BlockHostAccess:  true,
				Driver:           "bridge",
			},
			PullTimeout:  Duration(5 * time.Minute),
			StartTimeout: Duration(2 * time.Minute),
			StopTimeout:  Duration(10 * time.Second),
		},

		Instance: InstanceConfig{
			DefaultTTL:           Duration(30 * time.Minute),
			DefaultMaxRenewals:   3,
			MaxActivePerUser:     1,
			MaxActiveTotal:       100,
			PortRangeStart:       20000,
			PortRangeEnd:         30000,
			ReaperInterval:       Duration(30 * time.Second),
			ReconcileInterval:    Duration(5 * time.Minute),
			CleanupRetryInterval: Duration(1 * time.Minute),
		},

		Generator: GeneratorConfig{
			OutputDir:      "data/generated_files",
			DefaultTimeout: Duration(30 * time.Second),
			MaxTimeout:     Duration(5 * time.Minute),
			MaxOutputBytes: 64 << 20,
			MaxConcurrent:  4,
		},

		Submission: SubmissionConfig{MaxFlagLength: 512},

		Notifications: NotificationsConfig{
			HeartbeatInterval: Duration(25 * time.Second),
			MaxStreamsPerUser: 3,
			ReplayWindow:      Duration(1 * time.Hour),
		},

		PsqlConf: PsqlConfig{
			Host:            "localhost",
			Port:            "5432",
			User:            "beast",
			Dbname:          "beast",
			SslMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: Duration(30 * time.Minute),
		},

		RedisConf: RedisConfig{
			Host:         "localhost",
			Port:         "6379",
			User:         "beast",
			KeyPrefix:    "beast:",
			DialTimeout:  Duration(5 * time.Second),
			ReadTimeout:  Duration(3 * time.Second),
			WriteTimeout: Duration(3 * time.Second),
			PoolSize:     20,
		},

		MailConfig: MailConfig{
			Enabled:    false,
			Port:       587,
			Encryption: "starttls",
			Timeout:    Duration(10 * time.Second),
		},

		TickerInterval: Duration(1 * time.Minute),
		HealthProber:   true,
	}
}

// ApplyEnvOverrides lets secrets come from the environment instead of the file.
func (c *BeastConfig) ApplyEnvOverrides() error {
	setSecret := func(dst *Secret, key string) {
		if v := os.Getenv(key); v != "" {
			*dst = Secret(v)
		}
	}

	setSecret(&c.Auth.JWTSecret, "BEAST_JWT_SECRET")
	setSecret(&c.Auth.OperatorToken, "BEAST_OPERATOR_TOKEN")
	setSecret(&c.Docs.Password, "BEAST_DOCS_PASSWORD")
	setSecret(&c.PsqlConf.Password, "BEAST_DB_PASSWORD")
	setSecret(&c.RedisConf.Password, "BEAST_REDIS_PASSWORD")
	setSecret(&c.RedisConf.DefaultPassword, "BEAST_REDIS_DEFAULT_PASSWORD")
	setSecret(&c.MailConfig.Password, "BEAST_SMTP_PASSWORD")

	payloadEnv := os.Getenv("BEAST_LOG_HTTP_PAYLOADS")
	if payloadEnv == "" {
		payloadEnv = os.Getenv("LOG_HTTP_PAYLOADS")
	}
	if payloadEnv != "" {
		b, err := strconv.ParseBool(payloadEnv)
		if err != nil {
			return fmt.Errorf("invalid BEAST_LOG_HTTP_PAYLOADS %q: %w", payloadEnv, err)
		}
		c.Logging.LogHTTPPayloads = b
	}

	if v := os.Getenv("BEAST_ENV"); v != "" {
		c.Environment = Environment(v)
	}

	return nil
}

// Validate reports every problem at once, so the operator can fix them in one pass.
func (c *BeastConfig) Validate() error {

	for i := range c.NotificationWebhooks {
		if c.NotificationWebhooks[i].Timeout == 0 {
			c.NotificationWebhooks[i].Timeout = Duration(5 * time.Second)
		}
		if c.NotificationWebhooks[i].RetryBackoff == 0 {
			c.NotificationWebhooks[i].RetryBackoff = Duration(2 * time.Second)
		}
	}

	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	prod := c.Environment.IsProduction()
	if c.Environment != EnvDevelopment && c.Environment != EnvProduction {
		add("environment must be %q or %q, got %q", EnvDevelopment, EnvProduction, c.Environment)
	}

	// Server
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		add("server.port must be between 1 and 65535")
	}
	if c.Server.MaxBodyBytes <= 0 {
		add("server.max_body_bytes must be > 0")
	}
	if prod {
		if c.Server.PublicURL == "" {
			add("server.public_url is required in production (used in e-mail links)")
		}
		if len(c.Server.CORSAllowedOrigins) == 0 {
			add("server.cors_allowed_origins must be set in production (empty allows every origin)")
		}
		for _, o := range c.Server.CORSAllowedOrigins {
			if o == "*" {
				add("server.cors_allowed_origins must not contain \"*\" in production")
			}
		}
	}

	// Auth
	if c.Auth.JWTSecret == "" {
		add("auth.jwt_secret is required (set BEAST_JWT_SECRET)")
	} else if prod && len(c.Auth.JWTSecret) < 32 {
		add("auth.jwt_secret must be at least 32 characters in production")
	}
	if c.Auth.AccessTokenTTL <= 0 {
		add("auth.access_token_ttl must be > 0")
	}
	if c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		add("auth.refresh_token_ttl must be longer than auth.access_token_ttl")
	}
	if c.Auth.BcryptCost < 4 || c.Auth.BcryptCost > 15 {
		add("auth.bcrypt_cost must be between 4 and 15 (values > 15 cause severe CPU DoS)")
	} else if prod && c.Auth.BcryptCost < 10 {
		add("auth.bcrypt_cost must be at least 10 in production")
	}
	if c.Auth.MinPasswordLength < 1 {
		add("auth.min_password_length must be >= 1")
	}
	if prod && len(c.Auth.OperatorToken) < 16 {
		add("auth.operator_token must be at least 16 characters in production (protects /readyz)")
	}
	if c.Auth.Login2FAEnabled && !c.MailConfig.Enabled {
		add("auth.login_2fa_enabled requires mail_config.enabled to be true (otherwise users cannot receive 2FA codes)")
	}
	if c.Auth.RequireEmailVerification && !c.MailConfig.Enabled {
		add("auth.require_email_verification requires mail_config.enabled to be true (otherwise users cannot verify their accounts)")
	}
	if c.Auth.PasswordResetEnabled && !c.MailConfig.Enabled {
		add("auth.password_reset_enabled requires mail_config.enabled to be true (otherwise users cannot receive password reset codes)")
	}

	// OTP
	if c.OTP.CodeLength < 4 || c.OTP.CodeLength > 10 {
		add("otp.code_length must be between 4 and 10")
	}
	if c.OTP.MaxAttempts < 1 {
		add("otp.max_attempts must be >= 1")
	}
	for name, ttl := range map[string]Duration{
		"otp.email_verification_ttl": c.OTP.EmailVerificationTTL,
		"otp.password_reset_ttl":     c.OTP.PasswordResetTTL,
		"otp.login_2fa_ttl":          c.OTP.Login2FATTL,
	} {
		if ttl <= 0 {
			add("%s must be > 0", name)
		}
	}

	// Logging
	switch strings.ToLower(c.Logging.Level) {
	case "debug", "info", "warn", "error":
	default:
		add("logging.level must be debug, info, warn or error")
	}
	switch strings.ToLower(c.Logging.Format) {
	case "text", "json":
	default:
		add("logging.format must be text or json")
	}
	if prod && c.Logging.LogHTTPPayloads {
		add("logging.log_http_payloads must be false in production (logs personal data)")
	}

	// Docs
	if prod && c.Docs.Enabled && c.Docs.Password == "" {
		add("docs.password is required when docs are enabled in production")
	}

	// Storage
	if c.Storage.MaxLogoBytes <= 0 || c.Storage.MaxChallengeFileBytes <= 0 {
		add("storage.max_logo_bytes and storage.max_challenge_file_bytes must be > 0")
	}
	for _, t := range c.Storage.AllowedLogoTypes {
		if t == "image/svg+xml" {
			add("storage.allowed_logo_types: image/svg+xml is not allowed (XSS vulnerability risk)")
		}
	}

	// Runtime
	if c.Runtime.Memory <= 0 || c.Runtime.PidsLimit <= 0 || c.Runtime.CPUsLimit <= 0 {
		add("runtime default memory, pids and cpus limits must be > 0")
	}
	if prod && len(c.Runtime.AllowedBaseImages) == 0 && len(c.Runtime.AllowedRegistries) == 0 && !c.Runtime.RequireDigestPinning {
		add("runtime image policy: in production, at least one of allowed_base_images or allowed_registries must be specified, or require_digest_pinning must be true")
	}
	// Network Policy Validation
	if c.Runtime.Network.RestrictOutbound && len(c.Runtime.Network.DNSResolvers) > 0 {
		for _, r := range c.Runtime.Network.DNSResolvers {
			ip := net.ParseIP(r)
			if ip == nil || (!ip.IsPrivate() && !ip.IsLoopback()) {
				add("runtime.network.dns_resolvers: %q is not reachable when restrict_outbound is enabled (must be an internal/private IP or empty/nil to use embedded DNS)", r)
			}
		}
	}

	// Instances
	if c.Instance.DefaultTTL <= 0 {
		add("instance_config.default_ttl must be > 0")
	}
	if c.Instance.DefaultMaxRenewals < 0 {
		add("instance_config.default_max_renewals must be >= 0")
	}
	if c.Instance.MaxActivePerUser < 1 {
		add("instance_config.max_active_per_user must be >= 1")
	}
	if c.Instance.MaxActiveTotal < 0 {
		add("instance_config.max_active_total must be >= 0")
	}
	if prod && c.Instance.MaxActiveTotal <= 0 {
		add("instance_config.max_active_total must be > 0 in production (0 allows unlimited instances, risking resource exhaustion)")
	}
	if c.Instance.PortRangeStart < 1024 || c.Instance.PortRangeEnd > 65535 ||
		c.Instance.PortRangeStart >= c.Instance.PortRangeEnd {
		add("instance_config port range must satisfy 1024 <= start < end <= 65535")
	}
	if c.Instance.ReaperInterval <= 0 || c.Instance.ReconcileInterval <= 0 || c.Instance.CleanupRetryInterval <= 0 {
		add("instance_config reaper, reconcile and cleanup intervals must be > 0")
	}

	// Generators
	if c.Generator.DefaultTimeout <= 0 || c.Generator.DefaultTimeout > c.Generator.MaxTimeout {
		add("generator.default_timeout must be > 0 and <= generator.max_timeout")
	}
	if c.Generator.MaxConcurrent < 1 {
		add("generator.max_concurrent must be >= 1")
	}

	// Submissions and notifications
	if c.Submission.MaxFlagLength < 1 {
		add("submission.max_flag_length must be >= 1")
	}
	if c.Notifications.HeartbeatInterval <= 0 {
		add("notifications.heartbeat_interval must be > 0")
	}

	if c.TickerInterval <= 0 {
		add("ticker_interval must be > 0")
	}

	// Notification webhooks
	seenHooks := map[string]bool{}
	for i, w := range c.NotificationWebhooks {
		label := fmt.Sprintf("notification_webhooks[%d]", i)
		if w.Name != "" {
			label = fmt.Sprintf("notification_webhooks %q", w.Name)
		}
		if w.Name == "" {
			add("%s: name is required", label)
		} else if seenHooks[w.Name] {
			add("%s: duplicate name", label)
		}
		seenHooks[w.Name] = true

		switch w.Format {
		case "", "generic", "discord", "slack":
		default:
			add("%s: format must be generic, discord or slack", label)
		}

		raw := w.ResolvedURL()
		u, err := url.Parse(raw)
		switch {
		case raw == "":
			add("%s: url or url_env (set and non-empty) is required", label)
		case err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http"):
			add("%s: url must be a valid http(s) URL", label)
		case prod && u.Scheme != "https":
			add("%s: url must use https in production", label)
		}

		if len(w.Events) == 0 {
			add("%s: events must not be empty (specify at least one event type)", label)
		}
		for _, ev := range w.Events {
			if !models.NotificationType(ev).IsValid() {
				add("%s: unknown event %q (valid: announcement, challenge_update, first_blood, team)", label, ev)
			}
		}

		if w.Timeout <= 0 {
			add("%s: timeout must be > 0", label)
		}
		if w.RetryBackoff <= 0 {
			add("%s: retry_backoff must be > 0", label)
		}
		if w.MaxRetries < 0 {
			add("%s: max_retries must be >= 0", label)
		} else if w.MaxRetries >= 63 || int64(w.RetryBackoff) > (math.MaxInt64>>w.MaxRetries) {
			add("%s: max_retries %d with retry_backoff %s overflows maximum duration (int64)", label, w.MaxRetries, w.RetryBackoff)
		}
	}

	// Postgres
	if c.PsqlConf.Host == "" || c.PsqlConf.User == "" || c.PsqlConf.Dbname == "" {
		add("psql_config host, user and dbname are required")
	}
	if n, err := strconv.Atoi(c.PsqlConf.Port); err != nil || n < 1 || n > 65535 {
		add("psql_config.port must be a number between 1 and 65535")
	}
	switch c.PsqlConf.SslMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		add("psql_config.sslmode must be disable, allow, prefer, require, verify-ca or verify-full")
	}
	if prod && c.PsqlConf.SslMode == "disable" && !isLoopback(c.PsqlConf.Host) {
		add("psql_config.sslmode must not be \"disable\" for a non-local database in production")
	}
	if c.PsqlConf.MaxOpenConns < 1 || c.PsqlConf.MaxIdleConns < 0 || c.PsqlConf.MaxIdleConns > c.PsqlConf.MaxOpenConns {
		add("psql_config pool: max_open_conns must be >= 1 and max_idle_conns between 0 and max_open_conns")
	}

	// Redis
	if c.RedisConf.Host == "" || c.RedisConf.User == "" {
		add("redis_config host and user are required")
	}
	if n, err := strconv.Atoi(c.RedisConf.Port); err != nil || n < 1 || n > 65535 {
		add("redis_config.port must be a number between 1 and 65535")
	}
	// beast init creates the Redis ACL user with ~beast:* only.
	if !strings.HasPrefix(c.RedisConf.KeyPrefix, "beast:") {
		add("redis_config.key_prefix must start with \"beast:\" (the ACL user can only touch ~beast:*)")
	}
	if c.RedisConf.PoolSize < 1 {
		add("redis_config.pool_size must be >= 1")
	}

	// Mail
	if c.MailConfig.Enabled {
		if c.MailConfig.Host == "" || c.MailConfig.From == "" {
			add("mail_config host and from are required when mail is enabled")
		}
		if c.MailConfig.Port < 1 || c.MailConfig.Port > 65535 {
			add("mail_config.port must be between 1 and 65535")
		}
		switch c.MailConfig.Encryption {
		case "starttls", "tls", "none":
		default:
			add("mail_config.encryption must be starttls, tls or none")
		}
		if prod && c.MailConfig.Encryption == "none" {
			add("mail_config.encryption must not be \"none\" in production")
		}
	}

	return errors.Join(errs...)
}

func isLoopback(host string) bool {
	h := strings.TrimSpace(host)
	if h == "" {
		return false
	}

	// Unix socket directory (libpq/pgx convention).
	if strings.HasPrefix(h, "/") {
		return true
	}

	h = strings.TrimSuffix(strings.ToLower(h), ".") // "LocalHost." -> "localhost"
	if h == "localhost" {
		return true
	}

	h = strings.TrimSuffix(strings.TrimPrefix(h, "["), "]") // "[::1]" -> "::1"

	addr, err := netip.ParseAddr(h)
	if err != nil || addr.Zone() != "" {
		return false // not an IP: any other hostname is NOT assumed local
	}
	return addr.Unmap().IsLoopback() // covers all of 127.0.0.0/8, ::1 and ::ffff:127.x.x.x
}

// PsqlConfig is the PostgreSQL connection for the beast user.
// Port stays a string because beast init builds DSNs with %s.
type PsqlConfig struct {
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	User     string `toml:"user"`
	Password Secret `toml:"password"` // prefer env BEAST_DB_PASSWORD
	Dbname   string `toml:"dbname"`
	SslMode  string `toml:"sslmode"` // disable | allow | prefer | require | verify-ca | verify-full

	MaxOpenConns    int      `toml:"max_open_conns"`
	MaxIdleConns    int      `toml:"max_idle_conns"`
	ConnMaxLifetime Duration `toml:"conn_max_lifetime"`
}

// DSN returns a postgres:// URL. Unlike a key=value string, passwords with
// spaces, quotes or "@" are escaped correctly. Works with sql.Open("pgx", ...).
func (p PsqlConfig) DSN() string {
	q := url.Values{}
	if p.SslMode != "" {
		q.Set("sslmode", p.SslMode)
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.User, p.Password.Expose()),
		Path:   "/" + p.Dbname,
	}

	if strings.HasPrefix(p.Host, "/") {
		q.Set("host", p.Host)
		if p.Port != "" {
			q.Set("port", p.Port)
		}
	} else {
		u.Host = net.JoinHostPort(p.Host, p.Port)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// RedisConfig is the Redis connection for the beast ACL user.
type RedisConfig struct {
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	User     string `toml:"user"`
	Password Secret `toml:"password"` // prefer env BEAST_REDIS_PASSWORD
	DB       int    `toml:"db"`

	// Only used by `beast init` to log in as the "default" user and create the
	// beast ACL user. Never needed at runtime; leave empty after setup.
	DefaultPassword Secret `toml:"default_password"`

	// Every key must start with this. The ACL user only has access to ~beast:*.
	KeyPrefix string `toml:"key_prefix"`

	UseTLS       bool     `toml:"use_tls"`
	DialTimeout  Duration `toml:"dial_timeout"`
	ReadTimeout  Duration `toml:"read_timeout"`
	WriteTimeout Duration `toml:"write_timeout"`
	PoolSize     int      `toml:"pool_size"`
}

// Addr returns host:port for redis.Options.Addr.
func (r RedisConfig) Addr() string { return net.JoinHostPort(r.Host, r.Port) }

// MailConfig is the SMTP setup used for e-mail verification, password reset
// and login 2FA codes (auth_otps).
type MailConfig struct {
	Enabled bool `toml:"enabled"` // false = OTP e-mails are not sent

	Host     string `toml:"host"`
	Port     int    `toml:"port"` // 587 starttls, 465 tls, 25 none
	Username string `toml:"username"`
	Password Secret `toml:"password"` // prefer env BEAST_SMTP_PASSWORD

	// starttls | tls | none
	Encryption         string `toml:"encryption"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"` // dev only: accept self-signed certs

	From     string `toml:"from"`      // address, e.g. "ctf@example.com"
	FromName string `toml:"from_name"` // display name, e.g. "Beast CTF"
	ReplyTo  string `toml:"reply_to"`

	Timeout Duration `toml:"timeout"`

	// TemplatesDir is relative to StorageConfig.GlobalDir (e.g. "assets/mailTemplates").
	// Overrides the default e-mail templates in <global_dir>/assets/mailTemplates.
	TemplatesDir string `toml:"templates_dir"`
}

// Addr returns host:port for the SMTP dialer.
func (m MailConfig) Addr() string { return net.JoinHostPort(m.Host, strconv.Itoa(m.Port)) }

type NotificationWebhook struct {
	Name     string `toml:"name"`
	Disabled bool   `toml:"disabled"` // omitted = enabled

	// URLEnv wins over URL when both are set, falling back to URL. URLEnv
	// names an environment variable holding the URL so webhook tokens never sit in the config file.
	URL    string `toml:"url"` // for mock/dev purposes
	URLEnv string `toml:"url_env"`

	// generic (default): JSON body, signed when a secret is set.
	// discord / slack:   payload shaped for that service.
	Format string `toml:"format"`

	// Generic format only: the body is signed with HMAC-SHA256 and sent in the
	// X-Beast-Signature header so the receiver can verify it. SecretEnv wins
	// over Secret when both are set.
	Secret    Secret `toml:"secret"` // for mock purposes
	SecretEnv string `toml:"secret_env"`

	// Notification types to forward (see models.NotificationType).
	Events []string `toml:"events"` // this event will trigger this webhook

	// notifications.team_id / user_id set means the notification targets one
	// team or user. Those are NOT forwarded unless this is true, so private
	// messages don't leak to a public channel.
	IncludeTargeted bool `toml:"include_targeted"`

	Timeout      Duration `toml:"timeout"`
	MaxRetries   int      `toml:"max_retries"`   // after the first attempt
	RetryBackoff Duration `toml:"retry_backoff"` // doubled each retry
}

// ResolvedURL returns the URL from URLEnv if set and non-empty, otherwise URL.
func (w NotificationWebhook) ResolvedURL() string {
	if w.URLEnv != "" {
		if v := os.Getenv(w.URLEnv); v != "" {
			return v
		}
	}
	return w.URL
}

// ResolvedSecret returns the signing secret, preferring SecretEnv.
func (w NotificationWebhook) ResolvedSecret() string {
	if w.SecretEnv != "" {
		if v := os.Getenv(w.SecretEnv); v != "" {
			return v
		}
	}
	return w.Secret.Expose()
}

// Wants reports whether this webhook should receive a notification of the given
// type. targeted is true when the notification has a team_id or user_id.
func (w NotificationWebhook) Wants(eventType string, targeted bool) bool {
	if w.Disabled || (targeted && !w.IncludeTargeted) {
		return false
	}
	return slices.Contains(w.Events, eventType)
}
