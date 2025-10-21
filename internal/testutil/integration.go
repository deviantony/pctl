package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/portainer"
	"github.com/stretchr/testify/require"
)

// IntegrationConfig represents the configuration for integration tests
type IntegrationConfig struct {
	PortainerURL  string `json:"portainer_url"`
	APIToken      string `json:"api_token"`
	EnvironmentID int    `json:"environment_id"`
}

// LoadIntegrationConfig loads the integration test configuration from integration_test_config.json
func LoadIntegrationConfig(t require.TestingT) *IntegrationConfig {
	configPath := "integration_test_config.json"

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		require.FailNow(t, "integration_test_config.json not found. Copy integration_test_config.json.example to integration_test_config.json and configure it")
	}

	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "Failed to read integration test config")

	var cfg IntegrationConfig
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err, "Failed to parse integration test config")

	// Validate required fields
	require.NotEmpty(t, cfg.PortainerURL, "portainer_url is required")
	require.NotEmpty(t, cfg.APIToken, "api_token is required")
	require.Greater(t, cfg.EnvironmentID, 0, "environment_id must be greater than 0")

	return &cfg
}

// LoadIntegrationConfigSimple loads the integration test configuration without testing.T dependency
func LoadIntegrationConfigSimple() (*IntegrationConfig, error) {
	// Look for config file in project root (go up from tests/integration to project root)
	configPath := "../../integration_test_config.json"

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("integration_test_config.json not found. Copy integration_test_config.json.example to integration_test_config.json and configure it")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read integration test config: %w", err)
	}

	var cfg IntegrationConfig
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse integration test config: %w", err)
	}

	// Validate required fields
	if cfg.PortainerURL == "" {
		return nil, fmt.Errorf("portainer_url is required")
	}
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("api_token is required")
	}
	if cfg.EnvironmentID <= 0 {
		return nil, fmt.Errorf("environment_id must be greater than 0")
	}

	return &cfg, nil
}

// ValidatePortainerConnection validates that we can connect to Portainer and the environment exists
func ValidatePortainerConnection(t require.TestingT, cfg *IntegrationConfig) {
	client := portainer.NewClientWithTLS(cfg.PortainerURL, cfg.APIToken, true)

	// Try to get the specific environment - this validates URL reachability, auth, and environment existence
	environments, err := client.GetEnvironments()
	require.NoError(t, err, "Failed to connect to Portainer or authenticate")

	// Check if our environment ID exists
	found := false
	for _, env := range environments {
		if env.ID == cfg.EnvironmentID {
			found = true
			break
		}
	}
	require.True(t, found, "Environment ID %d not found in Portainer", cfg.EnvironmentID)
}

// ValidatePortainerConnectionSimple validates Portainer connection without testing.T dependency
func ValidatePortainerConnectionSimple(cfg *IntegrationConfig) error {
	client := portainer.NewClientWithTLS(cfg.PortainerURL, cfg.APIToken, true)

	// Try to get the specific environment - this validates URL reachability, auth, and environment existence
	environments, err := client.GetEnvironments()
	if err != nil {
		return fmt.Errorf("failed to connect to Portainer or authenticate: %w", err)
	}

	// Check if our environment ID exists
	found := false
	for _, env := range environments {
		if env.ID == cfg.EnvironmentID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("environment ID %d not found in Portainer", cfg.EnvironmentID)
	}

	return nil
}

// GenerateTestStackName generates a unique stack name for integration tests
func GenerateTestStackName() string {
	return fmt.Sprintf("pctl-integration-test-%d", time.Now().Unix())
}

// CreateTestConfig creates a temporary pctl.yml config file for testing
func CreateTestConfig(t require.TestingT, tempDir string, integrationCfg *IntegrationConfig, stackName string) string {
	config := &config.Config{
		PortainerURL:  integrationCfg.PortainerURL,
		APIToken:      integrationCfg.APIToken,
		EnvironmentID: integrationCfg.EnvironmentID,
		StackName:     stackName,
		ComposeFile:   "docker-compose.yml",
		SkipTLSVerify: true,
		Build: &config.BuildConfig{
			Mode:            config.BuildModeRemoteBuild,
			Parallel:        config.BuildParallelAuto,
			TagFormat:       "pctl-{{stack}}-{{service}}:{{hash}}",
			Platforms:       []string{"linux/amd64"},
			ExtraBuildArgs:  make(map[string]string),
			ForceBuild:      false,
			WarnThresholdMB: 50,
		},
	}

	configPath := filepath.Join(tempDir, "pctl.yml")
	err := config.Save()
	require.NoError(t, err, "Failed to create test config")

	return configPath
}

// CreateTestConfigForLoadMode creates a test config with load mode
func CreateTestConfigForLoadMode(t require.TestingT, tempDir string, integrationCfg *IntegrationConfig, stackName string) string {
	config := &config.Config{
		PortainerURL:  integrationCfg.PortainerURL,
		APIToken:      integrationCfg.APIToken,
		EnvironmentID: integrationCfg.EnvironmentID,
		StackName:     stackName,
		ComposeFile:   "docker-compose.yml",
		SkipTLSVerify: true,
		Build: &config.BuildConfig{
			Mode:            config.BuildModeLoad,
			Parallel:        config.BuildParallelAuto,
			TagFormat:       "pctl-{{stack}}-{{service}}:{{hash}}",
			Platforms:       []string{"linux/amd64"},
			ExtraBuildArgs:  make(map[string]string),
			ForceBuild:      false,
			WarnThresholdMB: 50,
		},
	}

	configPath := filepath.Join(tempDir, "pctl.yml")
	err := config.Save()
	require.NoError(t, err, "Failed to create test config for load mode")

	return configPath
}

// CleanupStack ensures a stack is deleted from Portainer
func CleanupStack(t require.TestingT, client *portainer.Client, stackName string, environmentID int) {
	stack, err := client.GetStack(stackName, environmentID)
	if err != nil {
		// Stack might not exist, which is fine
		return
	}

	if stack != nil {
		// Actually delete the stack
		err = client.DeleteStack(stack.ID, environmentID)
		if err != nil {
			// Just log the error, don't fail the test
			fmt.Printf("Warning: Failed to delete stack %s (ID: %d): %v\n", stackName, stack.ID, err)
		} else {
			fmt.Printf("Successfully deleted stack %s (ID: %d)\n", stackName, stack.ID)
		}
	}
}

// CreateSimpleComposeFile creates a simple docker-compose.yml without build directives
func CreateSimpleComposeFile(t require.TestingT, tempDir string) string {
	composeContent := `version: '3.8'

services:
  web:
    image: nginx:alpine
    ports:
      - "18080:80"
    environment:
      - NGINX_HOST=localhost
      - NGINX_PORT=80
    restart: unless-stopped

  redis:
    image: redis:alpine
    ports:
      - "16379:6379"
    restart: unless-stopped
`

	composePath := filepath.Join(tempDir, "docker-compose.yml")
	err := os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err, "Failed to create simple compose file")

	return composePath
}

// CreateBuildComposeFile creates a docker-compose.yml with build directives
func CreateBuildComposeFile(t require.TestingT, tempDir string) string {
	// Create test app directory
	appDir := filepath.Join(tempDir, "test-app")
	err := os.MkdirAll(appDir, 0755)
	require.NoError(t, err, "Failed to create test app directory")

	// Create simple Dockerfile
	dockerfileContent := `FROM nginx:alpine
COPY index.html /usr/share/nginx/html/index.html
EXPOSE 80
`
	dockerfilePath := filepath.Join(appDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644)
	require.NoError(t, err, "Failed to create test Dockerfile")

	// Create simple index.html
	indexContent := `<!DOCTYPE html>
<html>
<head>
    <title>pctl Integration Test</title>
</head>
<body>
    <h1>pctl Integration Test App</h1>
    <p>This is a test application for pctl integration tests.</p>
</body>
</html>
`
	indexPath := filepath.Join(appDir, "index.html")
	err = os.WriteFile(indexPath, []byte(indexContent), 0644)
	require.NoError(t, err, "Failed to create test index.html")

	// Create compose file with build directive
	composeContent := `version: '3.8'

services:
  app:
    build:
      context: ./test-app
      dockerfile: Dockerfile
    ports:
      - "13000:80"
    restart: unless-stopped

  redis:
    image: redis:alpine
    ports:
      - "16379:6379"
    restart: unless-stopped
`

	composePath := filepath.Join(tempDir, "docker-compose.yml")
	err = os.WriteFile(composePath, []byte(composeContent), 0644)
	require.NoError(t, err, "Failed to create build compose file")

	return composePath
}
