package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create a valid config file
	configContent := `
portainer_url: "https://portainer.example.com"
api_token: "test-token"
environment_id: 1
stack_name: "test-stack"
compose_file: "docker-compose.yml"
skip_tls_verify: true
`
	err := os.WriteFile(ConfigFileName, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test loading the config
	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "https://portainer.example.com", config.PortainerURL)
	assert.Equal(t, "test-token", config.APIToken)
	assert.Equal(t, 1, config.EnvironmentID)
	assert.Equal(t, "test-stack", config.StackName)
	assert.Equal(t, "docker-compose.yml", config.ComposeFile)
	assert.True(t, config.SkipTLSVerify)
}

func TestLoad_FileNotFound(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Don't create the config file
	config, err := Load()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "configuration file 'pctl.yml' not found")
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create an invalid YAML file
	invalidYAML := `
portainer_url: "https://portainer.example.com"
api_token: "test-token"
environment_id: 1
stack_name: "test-stack"
compose_file: "docker-compose.yml"
invalid_yaml: [unclosed array
`
	err := os.WriteFile(ConfigFileName, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Test loading the config
	config, err := Load()
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse configuration file")
}

func TestConfig_Save(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create a config
	config := &Config{
		PortainerURL:  "https://portainer.example.com",
		APIToken:      "test-token",
		EnvironmentID: 1,
		StackName:     "test-stack",
		ComposeFile:   "docker-compose.yml",
		SkipTLSVerify: true,
	}

	// Save the config
	err := config.Save()
	require.NoError(t, err)

	// Verify the file was created and contains the expected content
	content, err := os.ReadFile(ConfigFileName)
	require.NoError(t, err)
	assert.Contains(t, string(content), "portainer_url: https://portainer.example.com")
	assert.Contains(t, string(content), "api_token: test-token")
	assert.Contains(t, string(content), "environment_id: 1")
	assert.Contains(t, string(content), "stack_name: test-stack")
	assert.Contains(t, string(content), "compose_file: docker-compose.yml")
	assert.Contains(t, string(content), "skip_tls_verify: true")
}

func TestConfig_Validate(t *testing.T) {
	config := &Config{
		PortainerURL:  "https://portainer.example.com",
		APIToken:      "test-token",
		EnvironmentID: 1,
		StackName:     "test-stack",
		ComposeFile:   "docker-compose.yml",
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "missing portainer_url",
			config: Config{
				APIToken:      "test-token",
				EnvironmentID: 1,
				StackName:     "test-stack",
				ComposeFile:   "docker-compose.yml",
			},
			expected: "portainer_url is required",
		},
		{
			name: "missing api_token",
			config: Config{
				PortainerURL:  "https://portainer.example.com",
				EnvironmentID: 1,
				StackName:     "test-stack",
				ComposeFile:   "docker-compose.yml",
			},
			expected: "api_token is required",
		},
		{
			name: "missing environment_id",
			config: Config{
				PortainerURL: "https://portainer.example.com",
				APIToken:     "test-token",
				StackName:    "test-stack",
				ComposeFile:  "docker-compose.yml",
			},
			expected: "environment_id is required",
		},
		{
			name: "missing stack_name",
			config: Config{
				PortainerURL:  "https://portainer.example.com",
				APIToken:      "test-token",
				EnvironmentID: 1,
				ComposeFile:   "docker-compose.yml",
			},
			expected: "stack_name is required",
		},
		{
			name: "missing compose_file",
			config: Config{
				PortainerURL:  "https://portainer.example.com",
				APIToken:      "test-token",
				EnvironmentID: 1,
				StackName:     "test-stack",
			},
			expected: "compose_file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestBuildConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   BuildConfig
		expected string
	}{
		{
			name: "valid remote-build mode",
			config: BuildConfig{
				Mode:            BuildModeRemoteBuild,
				Parallel:        BuildParallelAuto,
				WarnThresholdMB: 50,
			},
			expected: "",
		},
		{
			name: "valid load mode",
			config: BuildConfig{
				Mode:            BuildModeLoad,
				Parallel:        BuildParallelAuto,
				WarnThresholdMB: 50,
			},
			expected: "",
		},
		{
			name: "invalid build mode",
			config: BuildConfig{
				Mode:            "invalid-mode",
				Parallel:        BuildParallelAuto,
				WarnThresholdMB: 50,
			},
			expected: "invalid build mode 'invalid-mode'",
		},
		{
			name: "invalid parallel value - empty",
			config: BuildConfig{
				Mode:            BuildModeRemoteBuild,
				Parallel:        "",
				WarnThresholdMB: 50,
			},
			expected: "invalid parallel value ''",
		},
		{
			name: "invalid parallel value - zero",
			config: BuildConfig{
				Mode:            BuildModeRemoteBuild,
				Parallel:        "0",
				WarnThresholdMB: 50,
			},
			expected: "invalid parallel value '0'",
		},
		{
			name: "negative warn threshold",
			config: BuildConfig{
				Mode:            BuildModeRemoteBuild,
				Parallel:        BuildParallelAuto,
				WarnThresholdMB: -1,
			},
			expected: "warn_threshold_mb must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expected == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expected)
			}
		})
	}
}

func TestConfig_GetBuildConfig(t *testing.T) {
	config := &Config{
		PortainerURL:  "https://portainer.example.com",
		APIToken:      "test-token",
		EnvironmentID: 1,
		StackName:     "test-stack",
		ComposeFile:   "docker-compose.yml",
	}

	buildConfig := config.GetBuildConfig()
	require.NotNil(t, buildConfig)

	assert.Equal(t, DefaultBuildMode, buildConfig.Mode)
	assert.Equal(t, DefaultBuildParallel, buildConfig.Parallel)
	assert.Equal(t, DefaultBuildTagFormat, buildConfig.TagFormat)
	assert.Equal(t, []string{"linux/amd64"}, buildConfig.Platforms)
	assert.NotNil(t, buildConfig.ExtraBuildArgs)
	assert.False(t, buildConfig.ForceBuild)
	assert.Equal(t, DefaultBuildWarnThresholdMB, buildConfig.WarnThresholdMB)
}

func TestConfig_GetBuildConfig_NilBuild(t *testing.T) {
	config := &Config{
		PortainerURL:  "https://portainer.example.com",
		APIToken:      "test-token",
		EnvironmentID: 1,
		StackName:     "test-stack",
		ComposeFile:   "docker-compose.yml",
		Build:         nil,
	}

	buildConfig := config.GetBuildConfig()
	require.NotNil(t, buildConfig)

	assert.Equal(t, DefaultBuildMode, buildConfig.Mode)
	assert.Equal(t, DefaultBuildParallel, buildConfig.Parallel)
	assert.Equal(t, DefaultBuildTagFormat, buildConfig.TagFormat)
	assert.Equal(t, []string{"linux/amd64"}, buildConfig.Platforms)
	assert.NotNil(t, buildConfig.ExtraBuildArgs)
	assert.False(t, buildConfig.ForceBuild)
	assert.Equal(t, DefaultBuildWarnThresholdMB, buildConfig.WarnThresholdMB)
}

func TestGetDefaultStackName(t *testing.T) {
	// Create a temporary directory with a specific name
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create a subdirectory with a test name
	testDir := filepath.Join(tempDir, "my-test-project")
	err := os.Mkdir(testDir, 0755)
	require.NoError(t, err)

	os.Chdir(testDir)

	stackName := GetDefaultStackName()
	assert.Equal(t, "pctl_my_test_project", stackName)
}

func TestGetDefaultComposeFile(t *testing.T) {
	composeFile := GetDefaultComposeFile()
	assert.Equal(t, DefaultComposeFile, composeFile)
}

func TestConfig_Validate_WithBuildConfig(t *testing.T) {
	config := &Config{
		PortainerURL:  "https://portainer.example.com",
		APIToken:      "test-token",
		EnvironmentID: 1,
		StackName:     "test-stack",
		ComposeFile:   "docker-compose.yml",
		Build: &BuildConfig{
			Mode:            "invalid-mode",
			Parallel:        BuildParallelAuto,
			WarnThresholdMB: 50,
		},
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid build configuration")
}

func TestConfig_GetBuildConfig_WithPartialBuildConfig(t *testing.T) {
	config := &Config{
		PortainerURL:  "https://portainer.example.com",
		APIToken:      "test-token",
		EnvironmentID: 1,
		StackName:     "test-stack",
		ComposeFile:   "docker-compose.yml",
		Build: &BuildConfig{
			Mode: BuildModeLoad,
			// Other fields are empty/zero values
		},
	}

	buildConfig := config.GetBuildConfig()
	require.NotNil(t, buildConfig)

	assert.Equal(t, BuildModeLoad, buildConfig.Mode)                          // Should preserve the set value
	assert.Equal(t, DefaultBuildParallel, buildConfig.Parallel)               // Should apply default
	assert.Equal(t, DefaultBuildTagFormat, buildConfig.TagFormat)             // Should apply default
	assert.Equal(t, []string{"linux/amd64"}, buildConfig.Platforms)           // Should apply default
	assert.NotNil(t, buildConfig.ExtraBuildArgs)                              // Should initialize empty map
	assert.False(t, buildConfig.ForceBuild)                                   // Should preserve zero value
	assert.Equal(t, DefaultBuildWarnThresholdMB, buildConfig.WarnThresholdMB) // Should apply default
}
