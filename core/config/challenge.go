package config

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sdslabs/beastv4/core/database/models"
	"github.com/sdslabs/beastv4/utils"
)

type DeploymentMode = models.DeploymentMode
type RuntimeType = models.RuntimeType

const (
	DeploymentModeOffline   = models.DeploymentModeOffline
	DeploymentModeShared    = models.DeploymentModeShared
	DeploymentModeInstanced = models.DeploymentModeInstanced

	RuntimeTypeImage   = models.RuntimeTypeImage
	RuntimeTypeCompose = models.RuntimeTypeCompose
)

type ChallengeHintConfig struct {
	Cost      int        `toml:"cost" json:"cost" yaml:"cost"`
	Content   string     `toml:"content" json:"content" yaml:"content"`
	ReleaseAt *time.Time `toml:"release_at,omitempty" json:"release_at,omitempty" yaml:"release_at,omitempty"`
}

type ChallengePortConfig struct {
	Port     int    `toml:"port" json:"port" yaml:"port"`
	Protocol string `toml:"protocol" json:"protocol" yaml:"protocol"`
	Name     string `toml:"name,omitempty" json:"name,omitempty" yaml:"name,omitempty"`
	Path     string `toml:"path,omitempty" json:"path,omitempty" yaml:"path,omitempty"`
}

type ChallengeRuntimeConfig struct {
	Dockerfile  string                `toml:"dockerfile,omitempty" json:"dockerfile,omitempty" yaml:"dockerfile,omitempty"`
	ComposeFile string                `toml:"compose_file,omitempty" json:"compose_file,omitempty" yaml:"compose_file,omitempty"`
	Ports       []ChallengePortConfig `toml:"ports,omitempty" json:"ports,omitempty" yaml:"ports,omitempty"`
	Environment map[string]string     `toml:"environment,omitempty" json:"environment,omitempty" yaml:"environment,omitempty"`

	ResourceLimits
	models.ChallengeRuntimeConfigBase

	Privileged bool `toml:"privileged,omitempty" json:"privileged,omitempty" yaml:"privileged,omitempty"`
}

type ChallengeInstanceConfig struct {
	TTLMinutes  int `toml:"ttl_minutes,omitempty" json:"ttl_minutes,omitempty" yaml:"ttl_minutes,omitempty"`
	MaxRenewals int `toml:"max_renewals,omitempty" json:"max_renewals,omitempty" yaml:"max_renewals,omitempty"`
}

type ChallengeConfig struct {
	models.ChallengeBase
	Flag                string   `toml:"flag,omitempty" json:"flag,omitempty" yaml:"flag,omitempty"`
	IsFlagCaseSensitive bool     `toml:"is_flag_case_sensitive" json:"is_flag_case_sensitive" yaml:"is_flag_case_sensitive"`
	Tags                []string `toml:"tags,omitempty" json:"tags,omitempty" yaml:"tags,omitempty"`

	// Prerequisites is a list of challenge slugs that must be solved before unlocking this challenge.
	Prerequisites []string              `toml:"prerequisites,omitempty" json:"prerequisites,omitempty" yaml:"prerequisites,omitempty"`
	Hints         []ChallengeHintConfig `toml:"hints,omitempty" json:"hints,omitempty" yaml:"hints,omitempty"`

	StaticFiles    []string `toml:"static_files,omitempty" json:"static_files,omitempty" yaml:"static_files,omitempty"`
	GeneratedFiles []string `toml:"generated_files,omitempty" json:"generated_files,omitempty" yaml:"generated_files,omitempty"`

	Runtime  *ChallengeRuntimeConfig  `toml:"runtime,omitempty" json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Ports    []ChallengePortConfig    `toml:"ports,omitempty" json:"ports,omitempty" yaml:"ports,omitempty"`
	Instance *ChallengeInstanceConfig `toml:"instance,omitempty" json:"instance,omitempty" yaml:"instance,omitempty"`
}

func DefaultChallengeConfig() ChallengeConfig {
	return ChallengeConfig{
		ChallengeBase: models.ChallengeBase{
			MaxPoints:         500,
			MinPoints:         100,
			DecaySolves:       50,
			DeploymentMode:    DeploymentModeOffline,
			GeneratorTimeoutS: 30,
		},
		IsFlagCaseSensitive: true,
	}
}

// GetGeneratorTimeout returns the effective generator timeout duration.
func (c *ChallengeConfig) GetGeneratorTimeout() time.Duration {
	if c.GeneratorTimeoutS > 0 {
		return time.Duration(c.GeneratorTimeoutS) * time.Second
	}
	return 30 * time.Second
}

// GetAllPorts returns all exposed ports configured either under Root Ports or Runtime.Ports.
func (c *ChallengeConfig) GetAllPorts() []ChallengePortConfig {
	var ports []ChallengePortConfig
	if c.Runtime != nil && len(c.Runtime.Ports) > 0 {
		ports = append(ports, c.Runtime.Ports...)
	}
	if len(c.Ports) > 0 {
		ports = append(ports, c.Ports...)
	}
	return ports
}

func (c *ChallengeConfig) Validate(challengeDir string) error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	challengeDir = strings.TrimSpace(challengeDir)
	if challengeDir == "" {
		challengeDir = "."
	}

	if err := utils.ValidateDirExists(challengeDir); err != nil {
		return fmt.Errorf("invalid challenge directory %q: %w", challengeDir, err)
	}

	if strings.TrimSpace(c.Name) == "" {
		add("challenge name is required")
	}
	if strings.TrimSpace(c.Slug) == "" {
		add("challenge slug is required")
	} else if !utils.IsValidSlug(c.Slug) {
		add("challenge slug %q must be lowercase alphanumeric with hyphens (e.g. 'baby-overflow')", c.Slug)
	}

	if c.MinPoints < 0 {
		add("min_points must be >= 0")
	}
	if c.MaxPoints < c.MinPoints {
		add("max_points (%d) must be greater than or equal to min_points (%d)", c.MaxPoints, c.MinPoints)
	}
	if c.DecaySolves <= 0 {
		add("decay_solves must be > 0")
	}
	if c.MaxAttempts != nil && *c.MaxAttempts <= 0 {
		add("max_attempts must be > 0 when specified")
	}

	// Validate Flag & Generator Invariants
	hasStaticFlag := strings.TrimSpace(c.Flag) != ""
	hasGenerator := c.GeneratorPath != nil && strings.TrimSpace(*c.GeneratorPath) != ""
	hasFlagTemplate := c.FlagTemplate != nil && strings.TrimSpace(*c.FlagTemplate) != ""

	if hasStaticFlag && hasGenerator {
		add("challenge cannot configure both static 'flag' and dynamic 'generator_path'")
	}
	if hasGenerator && !hasFlagTemplate {
		add("generator_path requires flag_template to be configured")
	}
	if !hasGenerator && len(c.GeneratedFiles) > 0 {
		add("generated_files cannot be configured without a generator_path")
	}

	// Validate Prerequisites
	seenPrereqs := make(map[string]bool)
	for i, prereq := range c.Prerequisites {
		prereqSlug := strings.TrimSpace(prereq)
		if prereqSlug == "" {
			add("prerequisite[%d]: slug cannot be empty", i)
			continue
		}
		if !utils.IsValidSlug(prereqSlug) {
			add("prerequisite[%d]: slug %q must be lowercase alphanumeric with hyphens", i, prereqSlug)
		}
		if prereqSlug == strings.TrimSpace(c.Slug) {
			add("prerequisite[%d]: challenge cannot have itself (%q) as a prerequisite", i, prereqSlug)
		}
		if seenPrereqs[prereqSlug] {
			add("prerequisite[%d]: duplicate prerequisite slug %q", i, prereqSlug)
		}
		seenPrereqs[prereqSlug] = true
	}

	switch c.DeploymentMode {
	case DeploymentModeOffline, DeploymentModeShared, DeploymentModeInstanced:
	default:
		add("invalid deployment_mode %q: must be one of 'offline', 'shared', 'instanced'", c.DeploymentMode)
	}

	allPorts := c.GetAllPorts()
	switch c.DeploymentMode {
	case DeploymentModeOffline:
		if c.Runtime != nil && (c.Runtime.Dockerfile != "" || (c.Runtime.ImageRef != nil && *c.Runtime.ImageRef != "") || c.Runtime.ComposeFile != "") {
			add("offline challenge must not configure container runtime (dockerfile, image, or compose_file)")
		}
	case DeploymentModeShared, DeploymentModeInstanced:
		if c.Runtime == nil {
			add("%s challenge requires runtime configuration", c.DeploymentMode)
		} else {
			hasDockerfile := strings.TrimSpace(c.Runtime.Dockerfile) != ""
			hasImage := c.Runtime.ImageRef != nil && strings.TrimSpace(*c.Runtime.ImageRef) != ""
			hasCompose := strings.TrimSpace(c.Runtime.ComposeFile) != ""

			count := 0
			if hasDockerfile {
				count++
			}
			if hasImage {
				count++
			}
			if hasCompose {
				count++
			}

			if count != 1 {
				add("%s challenge runtime must specify exactly one of dockerfile, image, or compose_file", c.DeploymentMode)
			}
			if hasDockerfile {
				// TODO : check if it complies with the rules of Cfg.Runtime
				dfPath, err := utils.SafeJoin(challengeDir, c.Runtime.Dockerfile)
				if err != nil {
					add("runtime dockerfile %q path error: %w", c.Runtime.Dockerfile, err)
				} else if err := utils.ValidateFileReadable(dfPath); err != nil {
					add("runtime dockerfile %q is not accessible in challenge directory: %w", c.Runtime.Dockerfile, err)
				}
			}
			if hasCompose {
				// TODO : check if it complies with the rules of Cfg.Runtime
				cfPath, err := utils.SafeJoin(challengeDir, c.Runtime.ComposeFile)
				if err != nil {
					add("runtime compose_file %q path error: %w", c.Runtime.ComposeFile, err)
				} else if err := utils.ValidateFileReadable(cfPath); err != nil {
					add("runtime compose_file %q is not accessible in challenge directory: %w", c.Runtime.ComposeFile, err)
				}
			}
		}

		if len(allPorts) == 0 {
			add("%s challenge must expose at least one port", c.DeploymentMode)
		}
	}

	validProtocols := map[string]bool{
		"tcp":   true,
		"udp":   true,
		"http":  true,
		"https": true,
		"ssh":   true,
	}
	seenPorts := make(map[int]bool)
	for i, p := range allPorts {
		if p.Port < 1 || p.Port > 65535 {
			add("port[%d]: port number %d must be between 1 and 65535", i, p.Port)
		}
		proto := strings.ToLower(p.Protocol)
		if !validProtocols[proto] {
			add("port[%d]: invalid protocol %q (must be tcp, udp, http, https, or ssh)", i, p.Protocol)
		}
		if p.Path != "" {
			if proto != "http" && proto != "https" {
				add("port[%d]: path is only allowed for http/https protocols", i)
			} else {
				if !strings.HasPrefix(p.Path, "/") {
					add("port[%d]: path %q must start with '/'", i, p.Path)
				}
				if strings.Contains(p.Path, "//") {
					add("port[%d]: path %q must not contain consecutive slashes '//'", i, p.Path)
				}
				if strings.ContainsAny(p.Path, " \t\r\n") {
					add("port[%d]: path %q must not contain whitespace", i, p.Path)
				}
				if _, err := url.Parse(p.Path); err != nil {
					add("port[%d]: invalid URL path %q: %w", i, p.Path, err)
				}
			}
		}

		portKey := fmt.Sprintf("%s:%d", proto, p.Port)
		if seenPorts[p.Port] {
			add("port[%d]: duplicate port exposure %q across configured ports", i, portKey)
		}
		seenPorts[p.Port] = true
	}

	for i, h := range c.Hints {
		if strings.TrimSpace(h.Content) == "" {
			add("hint[%d]: content cannot be empty", i)
		}
		if h.Cost < 0 {
			add("hint[%d]: cost must be >= 0", i)
		}
	}

	for i, sf := range c.StaticFiles {
		trimmed := strings.TrimSpace(sf)
		if trimmed == "" {
			add("static_files[%d]: path cannot be empty", i)
			continue
		}

		fullPath, err := utils.SafeJoin(challengeDir, trimmed)
		if err != nil {
			add("static file %q path error: %w", trimmed, err)
		} else if err := utils.ValidateFileReadable(fullPath); err != nil {
			add("static file %q is not accessible in challenge directory: %w", trimmed, err)
		}
	}

	for i, gf := range c.GeneratedFiles {
		trimmed := strings.TrimSpace(gf)
		if trimmed == "" {
			add("generated_files[%d]: path cannot be empty", i)
			continue
		}
	}

	var staticNames []string

	for _, sf := range c.StaticFiles {
		name := filepath.Base(strings.TrimSpace(sf))
		staticNames = append(staticNames, name)
	}

	for _, gf := range c.GeneratedFiles {
		name := filepath.Base(strings.TrimSpace(gf))
		staticNames = append(staticNames, name)
	}

	slices.Sort(staticNames)
	for i := 1; i < len(staticNames); i++ {
		if staticNames[i] == staticNames[i-1] {
			add("same name (%s) multiple file found :", staticNames[i])
		}
	}

	if hasGenerator && hasFlagTemplate {
		genPath, err := utils.SafeJoin(challengeDir, *c.GeneratorPath)
		if err != nil {
			add("generator script %q path error: %w", *c.GeneratorPath, err)
		} else if err := utils.ValidateFileExecutable(genPath); err != nil {
			add("generator script %q is not executable: %w", *c.GeneratorPath, err)
		} else {
			tempDir, err := CreateTempDir("chal-validate-generator-")
			if err != nil {
				add("failed to create temporary directory for generator: %w", err)
			} else {
				defer utils.DeleteDir(tempDir)

				effectiveFlag, err := utils.GetRandomFlagFromTemplate(*c.FlagTemplate)
				if err != nil || effectiveFlag == "" {
					add("failed to parse flag from template: %w", err)
				} else {
					timeout := c.GetGeneratorTimeout()
					if err := utils.RunGeneratorWithTimeout(genPath, effectiveFlag, tempDir, timeout); err != nil {
						add("generator script execution failed: %w", err)
					} else {
						for _, gf := range c.GeneratedFiles {
							trimmed := strings.TrimSpace(gf)
							if trimmed == "" {
								continue
							}
							genFilePath, err := utils.SafeJoin(tempDir, trimmed)
							if err != nil {
								add("generated file %q path error: %w", trimmed, err)
							} else if err := utils.ValidateFileReadable(genFilePath); err != nil {
								add("expected generated file %q was not created by generator: %w", trimmed, err)
							}
						}
					}
				}
			}
		}
	}

	if c.Instance != nil {
		if c.Instance.TTLMinutes <= 0 {
			add("instance.ttl_minutes must be > 0")
		}
		if c.Instance.MaxRenewals < 0 {
			add("instance.max_renewals must be >= 0")
		}
	}

	if len(errs) > 0 {
		var b strings.Builder
		b.WriteString("challenge config validation failed:\n")
		for _, e := range errs {
			b.WriteString("  - ")
			b.WriteString(e.Error())
			b.WriteString("\n")
		}
		return errors.New(strings.TrimRight(b.String(), "\n"))
	}

	return nil
}
