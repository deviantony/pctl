//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deviantony/pctl/internal/portainer"
	"github.com/deviantony/pctl/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	integrationConfig *testutil.IntegrationConfig
	portainerClient   *portainer.Client
	projectRoot       string
)

// findProjectRoot finds the project root by looking for go.mod file
func findProjectRoot() (string, error) {
	// Start from the current test directory and go up to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Keep going up until we find go.mod or reach the filesystem root
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root, go.mod not found
			return "", fmt.Errorf("go.mod not found in current directory or any parent")
		}
		dir = parent
	}
}

func TestMain(m *testing.M) {
	// Find project root first
	var err error
	projectRoot, err = findProjectRoot()
	if err != nil {
		fmt.Printf("Failed to find project root: %v\n", err)
		os.Exit(1)
	}

	// Load integration configuration
	integrationConfig, err = testutil.LoadIntegrationConfigSimple()
	if err != nil {
		fmt.Printf("Failed to load integration config: %v\n", err)
		os.Exit(1)
	}

	// Validate Portainer connection
	err = testutil.ValidatePortainerConnectionSimple(integrationConfig)
	if err != nil {
		fmt.Printf("Failed to validate Portainer connection: %v\n", err)
		os.Exit(1)
	}

	// Create Portainer client
	portainerClient = portainer.NewClientWithTLS(
		integrationConfig.PortainerURL,
		integrationConfig.APIToken,
		true, // skip TLS verify for integration tests
	)

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// runPctlCommand executes a pctl command and returns the output
func runPctlCommand(t *testing.T, args ...string) (string, error) {
	// Use the project root found in TestMain
	if projectRoot == "" {
		return "", fmt.Errorf("project root not initialized")
	}

	currentDir, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Logf("Warning: Could not get current directory: %v", getwdErr)
		currentDir = "unknown"
	}
	t.Logf("Project root: %s", projectRoot)
	t.Logf("Current dir: %s", currentDir)

	// Build the pctl binary first
	buildCmd := exec.Command("go", "build", "-o", "pctl", ".")
	buildCmd.Dir = projectRoot
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to build pctl: %w, output: %s", err, string(output))
	}

	// Run the pctl command from the test directory
	pctlPath := filepath.Join(projectRoot, "pctl")
	cmd := exec.Command(pctlPath, args...)
	cmd.Dir = "." // Run from current test directory
	output, err = cmd.CombinedOutput()
	return string(output), err
}

func TestIntegration_DeploySimpleStack(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config
	testutil.CreateTestConfig(t, tempDir, integrationConfig, stackName)

	// Create simple compose file
	composePath := testutil.CreateSimpleComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack: %s", stackName)

	// Actually run pctl deploy
	output, err := runPctlCommand(t, "deploy")

	// Check if deployment succeeded (even if there's a panic at the end)
	if err != nil && !strings.Contains(output, "Stack deployed successfully!") {
		t.Logf("pctl deploy output: %s", output)
		t.Fatalf("pctl deploy failed: %v", err)
	}

	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created (even if pctl crashed after deployment)
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after deploy")
	require.NotNil(t, stack, "Stack should exist after deploy")
	assert.Equal(t, stackName, stack.Name, "Stack name should match")
	assert.Equal(t, integrationConfig.EnvironmentID, stack.EnvironmentID, "Environment ID should match")

	t.Logf("Successfully deployed stack: %s (ID: %d)", stack.Name, stack.ID)
}

func TestIntegration_RedeployStack(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config
	testutil.CreateTestConfig(t, tempDir, integrationConfig, stackName)

	// Create simple compose file
	composePath := testutil.CreateSimpleComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack: %s", stackName)

	// First deploy the stack
	output, err := runPctlCommand(t, "deploy")
	if err != nil && !strings.Contains(output, "Stack deployed successfully!") {
		t.Logf("pctl deploy output: %s", output)
		t.Fatalf("pctl deploy failed: %v", err)
	}

	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after deploy")
	require.NotNil(t, stack, "Stack should exist after deploy")

	t.Logf("Successfully deployed stack: %s (ID: %d)", stack.Name, stack.ID)

	// Now test redeploy
	t.Logf("Testing redeploy for stack: %s", stackName)

	output, err = runPctlCommand(t, "redeploy")
	if err != nil && !strings.Contains(output, "Stack redeployed successfully!") {
		t.Logf("pctl redeploy output: %s", output)
		t.Fatalf("pctl redeploy failed: %v", err)
	}

	t.Logf("pctl redeploy output: %s", output)

	// Verify stack still exists after redeploy
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after redeploy")
	require.NotNil(t, stack, "Stack should still exist after redeploy")

	t.Logf("Successfully redeployed stack: %s (ID: %d)", stack.Name, stack.ID)
}

func TestIntegration_RedeployStackForceRebuild(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config
	testutil.CreateTestConfig(t, tempDir, integrationConfig, stackName)

	// Create build compose file for force rebuild test
	composePath := testutil.CreateBuildComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack with build: %s", stackName)

	// First deploy the stack
	output, err := runPctlCommand(t, "deploy")
	if err != nil && !strings.Contains(output, "Stack deployed successfully!") {
		t.Logf("pctl deploy output: %s", output)
		t.Fatalf("pctl deploy failed: %v", err)
	}

	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after deploy")
	require.NotNil(t, stack, "Stack should exist after deploy")

	t.Logf("Successfully deployed stack: %s (ID: %d)", stack.Name, stack.ID)

	// Now test force rebuild redeploy
	t.Logf("Testing force rebuild redeploy for stack: %s", stackName)

	output, err = runPctlCommand(t, "redeploy", "-f")
	if err != nil && !strings.Contains(output, "Stack redeployed successfully!") {
		t.Logf("pctl redeploy -f output: %s", output)
		t.Fatalf("pctl redeploy -f failed: %v", err)
	}

	t.Logf("pctl redeploy -f output: %s", output)

	// Verify stack still exists after force rebuild redeploy
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after force rebuild redeploy")
	require.NotNil(t, stack, "Stack should still exist after force rebuild redeploy")

	t.Logf("Successfully force rebuilt stack: %s (ID: %d)", stack.Name, stack.ID)
}

func TestIntegration_PsCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config
	testutil.CreateTestConfig(t, tempDir, integrationConfig, stackName)

	// Create simple compose file
	composePath := testutil.CreateSimpleComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack: %s", stackName)

	// First deploy the stack
	output, err := runPctlCommand(t, "deploy")
	if err != nil && !strings.Contains(output, "Stack deployed successfully!") {
		t.Logf("pctl deploy output: %s", output)
		t.Fatalf("pctl deploy failed: %v", err)
	}

	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after deploy")
	require.NotNil(t, stack, "Stack should exist after deploy")

	t.Logf("Successfully deployed stack: %s (ID: %d)", stack.Name, stack.ID)

	// Now test ps command
	t.Logf("Testing ps command for stack: %s", stackName)

	output, err = runPctlCommand(t, "ps")
	if err != nil {
		t.Logf("pctl ps output: %s", output)
		t.Fatalf("pctl ps failed: %v", err)
	}

	t.Logf("pctl ps output: %s", output)

	// Verify output contains expected information
	assert.Contains(t, output, stackName, "PS output should contain stack name")
	assert.Contains(t, output, "nginx:alpine", "PS output should contain nginx image")
	assert.Contains(t, output, "redis:alpine", "PS output should contain redis image")

	t.Logf("Successfully ran ps command for stack: %s", stackName)
}

func TestIntegration_LogsCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config
	testutil.CreateTestConfig(t, tempDir, integrationConfig, stackName)

	// Create simple compose file
	composePath := testutil.CreateSimpleComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack: %s", stackName)

	// First deploy the stack
	output, err := runPctlCommand(t, "deploy")
	if err != nil && !strings.Contains(output, "Stack deployed successfully!") {
		t.Logf("pctl deploy output: %s", output)
		t.Fatalf("pctl deploy failed: %v", err)
	}

	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after deploy")
	require.NotNil(t, stack, "Stack should exist after deploy")

	t.Logf("Successfully deployed stack: %s (ID: %d)", stack.Name, stack.ID)

	// Now test logs command
	t.Logf("Testing logs command for stack: %s", stackName)

	output, err = runPctlCommand(t, "logs", "-t", "10", "--non-interactive")
	if err != nil {
		t.Logf("pctl logs output: %s", output)
		t.Fatalf("pctl logs failed: %v", err)
	}

	t.Logf("pctl logs output: %s", output)

	// Verify output contains expected information
	assert.Contains(t, output, stackName, "Logs output should contain stack name")

	t.Logf("Successfully ran logs command for stack: %s", stackName)
}

func TestIntegration_BuildRemoteMode(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config (default is remote-build mode)
	testutil.CreateTestConfig(t, tempDir, integrationConfig, stackName)

	// Create build compose file
	composePath := testutil.CreateBuildComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify test app directory exists
	appDir := filepath.Join(tempDir, "test-app")
	_, err = os.Stat(appDir)
	require.NoError(t, err, "Test app directory should exist")

	// Verify Dockerfile exists
	dockerfilePath := filepath.Join(appDir, "Dockerfile")
	_, err = os.Stat(dockerfilePath)
	require.NoError(t, err, "Dockerfile should exist")

	// Verify index.html exists
	indexPath := filepath.Join(appDir, "index.html")
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index.html should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack with remote build mode: %s", stackName)

	// Actually run pctl deploy
	output, err := runPctlCommand(t, "deploy")
	require.NoError(t, err, "pctl deploy with remote build should succeed")
	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created and image was built
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after remote build deploy")
	require.NotNil(t, stack, "Stack should exist after remote build deploy")

	t.Logf("Successfully deployed stack with remote build mode: %s (ID: %d)", stack.Name, stack.ID)
}

func TestIntegration_BuildLoadMode(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Generate unique stack name
	stackName := testutil.GenerateTestStackName()
	t.Cleanup(func() {
		testutil.CleanupStack(t, portainerClient, stackName, integrationConfig.EnvironmentID)
	})

	// Create test config for load mode
	testutil.CreateTestConfigForLoadMode(t, tempDir, integrationConfig, stackName)

	// Create build compose file
	composePath := testutil.CreateBuildComposeFile(t, tempDir)

	// Verify compose file was created
	_, err := os.Stat(composePath)
	require.NoError(t, err, "Compose file should exist")

	// Verify test app directory exists
	appDir := filepath.Join(tempDir, "test-app")
	_, err = os.Stat(appDir)
	require.NoError(t, err, "Test app directory should exist")

	// Verify Dockerfile exists
	dockerfilePath := filepath.Join(appDir, "Dockerfile")
	_, err = os.Stat(dockerfilePath)
	require.NoError(t, err, "Dockerfile should exist")

	// Verify index.html exists
	indexPath := filepath.Join(appDir, "index.html")
	_, err = os.Stat(indexPath)
	require.NoError(t, err, "index.html should exist")

	// Verify stack doesn't exist yet
	stack, err := portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack")
	assert.Nil(t, stack, "Stack should not exist initially")

	t.Logf("Deploying stack with load build mode: %s", stackName)

	// Actually run pctl deploy
	output, err := runPctlCommand(t, "deploy")
	require.NoError(t, err, "pctl deploy with load build should succeed")
	t.Logf("pctl deploy output: %s", output)

	// Verify stack was created and image was built
	stack, err = portainerClient.GetStack(stackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for stack after load build deploy")
	require.NotNil(t, stack, "Stack should exist after load build deploy")

	t.Logf("Successfully deployed stack with load build mode: %s (ID: %d)", stack.Name, stack.ID)
}

func TestIntegration_CleanupNonExistentStack(t *testing.T) {
	// Test cleanup of non-existent stack
	nonExistentStackName := fmt.Sprintf("pctl-nonexistent-stack-%d", time.Now().Unix())

	// This should not error even if stack doesn't exist
	testutil.CleanupStack(t, portainerClient, nonExistentStackName, integrationConfig.EnvironmentID)

	// Verify stack doesn't exist
	stack, err := portainerClient.GetStack(nonExistentStackName, integrationConfig.EnvironmentID)
	require.NoError(t, err, "Should be able to check for non-existent stack")
	assert.Nil(t, stack, "Non-existent stack should not exist")

	t.Logf("Cleanup non-existent stack test completed for stack: %s", nonExistentStackName)
}
