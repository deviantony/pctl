package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BuildConfig represents the build configuration
type BuildConfig struct {
	Mode            string            `yaml:"mode"`              // remote-build | load
	Parallel        string            `yaml:"parallel"`          // auto | number
	TagFormat       string            `yaml:"tag_format"`        // template with {{stack}}, {{service}}, {{hash}}, {{timestamp}}
	Platforms       []string          `yaml:"platforms"`         // used for load mode local builds
	ExtraBuildArgs  map[string]string `yaml:"extra_build_args"`  // optional global overrides
	ForceBuild      bool              `yaml:"force_build"`       // force rebuild even if unchanged
	WarnThresholdMB int               `yaml:"warn_threshold_mb"` // WARN if tar/image stream exceeds this size
}

// Config represents the pctl configuration structure
type Config struct {
	PortainerURL  string       `yaml:"portainer_url"`
	APIToken      string       `yaml:"api_token"`
	EnvironmentID int          `yaml:"environment_id"`
	StackName     string       `yaml:"stack_name"`
	ComposeFile   string       `yaml:"compose_file"`
	SkipTLSVerify bool         `yaml:"skip_tls_verify"`
	Build         *BuildConfig `yaml:"build,omitempty"`
}

const (
	ConfigFileName     = "pctl.yml"
	DefaultComposeFile = "docker-compose.yml"

	// Build mode constants
	BuildModeRemoteBuild = "remote-build"
	BuildModeLoad        = "load"

	// Build parallel constants
	BuildParallelAuto = "auto"

	// Default build configuration values
	DefaultBuildMode            = BuildModeRemoteBuild
	DefaultBuildParallel        = BuildParallelAuto
	DefaultBuildTagFormat       = "pctl-{{stack}}-{{service}}:{{hash}}"
	DefaultBuildWarnThresholdMB = 50
)

// Load reads and parses the pctl.yml configuration file
func Load() (*Config, error) {
	configPath := ConfigFileName

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file '%s' not found. Run 'pctl init' to create it", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to pctl.yml
func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(ConfigFileName, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// Validate checks if all required configuration fields are present
func (c *Config) Validate() error {
	if c.PortainerURL == "" {
		return fmt.Errorf("portainer_url is required")
	}

	if c.APIToken == "" {
		return fmt.Errorf("api_token is required")
	}

	if c.EnvironmentID == 0 {
		return fmt.Errorf("environment_id is required")
	}

	if c.StackName == "" {
		return fmt.Errorf("stack_name is required")
	}

	if c.ComposeFile == "" {
		return fmt.Errorf("compose_file is required")
	}

	// Validate build configuration if present
	if c.Build != nil {
		if err := c.Build.Validate(); err != nil {
			return fmt.Errorf("invalid build configuration: %w", err)
		}
	}

	return nil
}

// GetDefaultSkipTLSVerify returns the default value for skip_tls_verify
func GetDefaultSkipTLSVerify() bool {
	return true // Default to true for self-hosted environments
}

// GetDefaultStackName generates a default stack name based on the current directory
func GetDefaultStackName() string {
	wd, err := os.Getwd()
	if err != nil {
		return "pctl_project"
	}

	dirName := filepath.Base(wd)
	// Clean up directory name for use as stack name
	dirName = strings.ToLower(dirName)
	dirName = strings.ReplaceAll(dirName, " ", "_")
	dirName = strings.ReplaceAll(dirName, "-", "_")

	return fmt.Sprintf("pctl_%s", dirName)
}

// GetDefaultComposeFile returns the default compose file path
func GetDefaultComposeFile() string {
	return DefaultComposeFile
}

// GetBuildConfig returns the build configuration with defaults applied
func (c *Config) GetBuildConfig() *BuildConfig {
	if c.Build == nil {
		return &BuildConfig{
			Mode:            DefaultBuildMode,
			Parallel:        DefaultBuildParallel,
			TagFormat:       DefaultBuildTagFormat,
			Platforms:       []string{"linux/amd64"},
			ExtraBuildArgs:  make(map[string]string),
			ForceBuild:      false,
			WarnThresholdMB: DefaultBuildWarnThresholdMB,
		}
	}

	// Apply defaults for missing fields
	build := *c.Build // Copy the struct

	if build.Mode == "" {
		build.Mode = DefaultBuildMode
	}
	if build.Parallel == "" {
		build.Parallel = DefaultBuildParallel
	}
	if build.TagFormat == "" {
		build.TagFormat = DefaultBuildTagFormat
	}
	if len(build.Platforms) == 0 {
		build.Platforms = []string{"linux/amd64"}
	}
	if build.ExtraBuildArgs == nil {
		build.ExtraBuildArgs = make(map[string]string)
	}
	if build.WarnThresholdMB == 0 {
		build.WarnThresholdMB = DefaultBuildWarnThresholdMB
	}

	return &build
}

// ValidateBuildConfig validates the build configuration
func (bc *BuildConfig) Validate() error {
	if bc.Mode != BuildModeRemoteBuild && bc.Mode != BuildModeLoad {
		return fmt.Errorf("invalid build mode '%s', must be '%s' or '%s'", bc.Mode, BuildModeRemoteBuild, BuildModeLoad)
	}

	if bc.Parallel != BuildParallelAuto {
		// If not "auto", it should be a positive integer
		if bc.Parallel == "" || bc.Parallel == "0" {
			return fmt.Errorf("invalid parallel value '%s', must be 'auto' or a positive integer", bc.Parallel)
		}
	}

	if bc.WarnThresholdMB < 0 {
		return fmt.Errorf("warn_threshold_mb must be non-negative, got %d", bc.WarnThresholdMB)
	}

	return nil
}
