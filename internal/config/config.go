package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the pctl configuration structure
type Config struct {
	PortainerURL     string `yaml:"portainer_url"`
	APIToken         string `yaml:"api_token"`
	EnvironmentID    int    `yaml:"environment_id"`
	StackName        string `yaml:"stack_name"`
	ComposeFile      string `yaml:"compose_file"`
	SkipTLSVerify    bool   `yaml:"skip_tls_verify"`
}

const (
	ConfigFileName     = "pctl.yml"
	DefaultComposeFile = "docker-compose.yml"
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
