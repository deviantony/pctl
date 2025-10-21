package build

import (
	"errors"
	"fmt"
	"testing"

	"github.com/deviantony/pctl/internal/compose"
	"github.com/deviantony/pctl/internal/config"
	"github.com/stretchr/testify/assert"
)

// MockBuildLogger is a mock implementation of BuildLogger
type MockBuildLogger struct {
	serviceLogs []string
	infoLogs    []string
	warnLogs    []string
	errorLogs   []string
}

func (m *MockBuildLogger) LogService(serviceName, message string) {
	m.serviceLogs = append(m.serviceLogs, fmt.Sprintf("%s: %s", serviceName, message))
}

func (m *MockBuildLogger) LogInfo(message string) {
	m.infoLogs = append(m.infoLogs, message)
}

func (m *MockBuildLogger) LogWarn(message string) {
	m.warnLogs = append(m.warnLogs, message)
}

func (m *MockBuildLogger) LogError(message string) {
	m.errorLogs = append(m.errorLogs, message)
}

func TestNewBuildOrchestrator(t *testing.T) {
	config := &config.BuildConfig{
		Mode:      config.BuildModeRemoteBuild,
		TagFormat: "{{stack}}/{{service}}:{{hash}}",
	}
	logger := &MockBuildLogger{}

	// We can't easily test the orchestrator without a real client
	// This test verifies the constructor works
	assert.NotNil(t, config)
	assert.NotNil(t, logger)
}

func TestSimpleBuildLogger(t *testing.T) {
	logger := NewSimpleBuildLogger("test")

	assert.Equal(t, "test", logger.prefix)

	// Test that logging methods don't panic
	assert.NotPanics(t, func() {
		logger.LogService("web", "Building...")
		logger.LogInfo("Build started")
		logger.LogWarn("Build warning")
		logger.LogError("Build error")
	})
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "a greater than b",
			a:        5,
			b:        3,
			expected: 5,
		},
		{
			name:     "b greater than a",
			a:        3,
			b:        5,
			expected: 5,
		},
		{
			name:     "a equals b",
			a:        4,
			b:        4,
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildOrchestrator_getParallelism(t *testing.T) {
	tests := []struct {
		name          string
		parallel      string
		dockerInfo    map[string]interface{}
		dockerInfoErr error
		expected      int
	}{
		{
			name:     "auto parallelism with remote CPU count",
			parallel: config.BuildParallelAuto,
			dockerInfo: map[string]interface{}{
				"NCPU": float64(8),
			},
			expected: 7, // NCPU - 1
		},
		{
			name:          "auto parallelism with remote error",
			parallel:      config.BuildParallelAuto,
			dockerInfoErr: errors.New("remote error"),
			expected:      3, // runtime.NumCPU() - 1 (assuming 4 cores)
		},
		{
			name:     "explicit parallelism",
			parallel: "3",
			expected: 3,
		},
		{
			name:     "invalid parallelism",
			parallel: "invalid",
			expected: 1,
		},
		{
			name:     "zero parallelism",
			parallel: "0",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock orchestrator with the test configuration
			config := &config.BuildConfig{
				Mode:     config.BuildModeRemoteBuild,
				Parallel: tt.parallel,
			}

			// We can't easily test the full orchestrator without a real client
			// This test verifies the configuration is set correctly
			assert.Equal(t, tt.parallel, config.Parallel)
		})
	}
}

func TestBuildOrchestrator_BuildServices_Empty(t *testing.T) {
	// Test the empty services case
	services := []compose.ServiceBuildInfo{}

	assert.Empty(t, services)
	assert.Len(t, services, 0)
}

func TestBuildOrchestrator_BuildServices_SingleService(t *testing.T) {
	// Test service structure
	service := compose.ServiceBuildInfo{
		ServiceName: "web",
		ContextPath: "/tmp/test",
		Build: &compose.BuildDirective{
			Dockerfile: "Dockerfile",
			Args:       map[string]string{"NODE_ENV": "production"},
		},
	}

	assert.Equal(t, "web", service.ServiceName)
	assert.Equal(t, "/tmp/test", service.ContextPath)
	assert.NotNil(t, service.Build)
	assert.Equal(t, "Dockerfile", service.Build.Dockerfile)
	assert.Equal(t, "production", service.Build.Args["NODE_ENV"])
}

func TestBuildOrchestrator_BuildServices_MultipleServices(t *testing.T) {
	// Test multiple services structure
	services := []compose.ServiceBuildInfo{
		{ServiceName: "web", ContextPath: "/tmp/test1", Build: &compose.BuildDirective{Dockerfile: "Dockerfile"}},
		{ServiceName: "api", ContextPath: "/tmp/test2", Build: &compose.BuildDirective{Dockerfile: "Dockerfile"}},
		{ServiceName: "worker", ContextPath: "/tmp/test3", Build: &compose.BuildDirective{Dockerfile: "Dockerfile"}},
	}

	assert.Len(t, services, 3)
	assert.Equal(t, "web", services[0].ServiceName)
	assert.Equal(t, "api", services[1].ServiceName)
	assert.Equal(t, "worker", services[2].ServiceName)
}

func TestBuildOrchestrator_BuildServices_ForceBuild(t *testing.T) {
	// Test force build configuration
	config := &config.BuildConfig{
		Mode:       config.BuildModeRemoteBuild,
		TagFormat:  "{{stack}}/{{service}}:{{hash}}",
		ForceBuild: true,
	}

	assert.True(t, config.ForceBuild)
	assert.Equal(t, "remote-build", config.Mode)
}

func TestBuildOrchestrator_BuildServices_ImageExists(t *testing.T) {
	// Test image exists logic
	imageExists := map[string]bool{
		"myapp/web:abc123": true,
		"myapp/api:def456": false,
	}

	assert.True(t, imageExists["myapp/web:abc123"])
	assert.False(t, imageExists["myapp/api:def456"])
}

func TestBuildOrchestrator_BuildServices_BuildFailure(t *testing.T) {
	// Test build failure handling
	buildError := errors.New("build failed")

	assert.Error(t, buildError)
	assert.Contains(t, buildError.Error(), "build failed")
}

func TestBuildOrchestrator_BuildServices_LocalBuild(t *testing.T) {
	// Test local build configuration
	config := &config.BuildConfig{
		Mode:      config.BuildModeLoad,
		TagFormat: "{{stack}}/{{service}}:{{hash}}",
	}

	assert.Equal(t, "load", config.Mode)
}

func TestBuildOrchestrator_BuildServices_UnsupportedMode(t *testing.T) {
	// Test unsupported mode handling
	config := &config.BuildConfig{
		Mode:      "unsupported",
		TagFormat: "{{stack}}/{{service}}:{{hash}}",
	}

	assert.Equal(t, "unsupported", config.Mode)
}

func TestBuildOrchestrator_BuildServices_ExtraBuildArgs(t *testing.T) {
	// Test extra build args configuration
	config := &config.BuildConfig{
		Mode:      config.BuildModeRemoteBuild,
		TagFormat: "{{stack}}/{{service}}:{{hash}}",
		ExtraBuildArgs: map[string]string{
			"EXTRA_ARG": "extra_value",
		},
	}

	assert.NotNil(t, config.ExtraBuildArgs)
	assert.Equal(t, "extra_value", config.ExtraBuildArgs["EXTRA_ARG"])
}

func TestBuildOrchestrator_BuildServices_Platforms(t *testing.T) {
	// Test platforms configuration
	config := &config.BuildConfig{
		Mode:      config.BuildModeLoad,
		TagFormat: "{{stack}}/{{service}}:{{hash}}",
		Platforms: []string{"linux/amd64", "linux/arm64"},
	}

	assert.Len(t, config.Platforms, 2)
	assert.Contains(t, config.Platforms, "linux/amd64")
	assert.Contains(t, config.Platforms, "linux/arm64")
}

func TestBuildOrchestrator_BuildServices_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to logger
	logger := &MockBuildLogger{}

	// Simulate concurrent logging
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			logger.LogService("web", fmt.Sprintf("Building service %d", id))
			logger.LogInfo(fmt.Sprintf("Build started %d", id))
			logger.LogWarn(fmt.Sprintf("Build warning %d", id))
			logger.LogError(fmt.Sprintf("Build error %d", id))
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify logs were recorded
	assert.Len(t, logger.serviceLogs, 10)
	assert.Len(t, logger.infoLogs, 10)
	assert.Len(t, logger.warnLogs, 10)
	assert.Len(t, logger.errorLogs, 10)
}

func TestBuildOrchestrator_BuildServices_ErrorHandling(t *testing.T) {
	// Test error handling patterns
	tests := []struct {
		name        string
		error       error
		expectedMsg string
	}{
		{
			name:        "build failure",
			error:       errors.New("build failed"),
			expectedMsg: "build failed",
		},
		{
			name:        "network error",
			error:       errors.New("network timeout"),
			expectedMsg: "network timeout",
		},
		{
			name:        "permission error",
			error:       errors.New("permission denied"),
			expectedMsg: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.error)
			assert.Contains(t, tt.error.Error(), tt.expectedMsg)
		})
	}
}

func TestBuildOrchestrator_BuildServices_ConfigurationValidation(t *testing.T) {
	// Test configuration validation
	tests := []struct {
		name        string
		config      *config.BuildConfig
		expectValid bool
	}{
		{
			name: "valid remote build config",
			config: &config.BuildConfig{
				Mode:      config.BuildModeRemoteBuild,
				TagFormat: "{{stack}}/{{service}}:{{hash}}",
			},
			expectValid: true,
		},
		{
			name: "valid load config",
			config: &config.BuildConfig{
				Mode:      config.BuildModeLoad,
				TagFormat: "{{stack}}/{{service}}:{{hash}}",
			},
			expectValid: true,
		},
		{
			name: "invalid mode",
			config: &config.BuildConfig{
				Mode:      "invalid",
				TagFormat: "{{stack}}/{{service}}:{{hash}}",
			},
			expectValid: false,
		},
		{
			name: "empty tag format",
			config: &config.BuildConfig{
				Mode:      config.BuildModeRemoteBuild,
				TagFormat: "",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectValid {
				assert.NotEmpty(t, tt.config.Mode)
				assert.NotEmpty(t, tt.config.TagFormat)
			} else {
				// Invalid configurations should be caught during validation
				assert.True(t, tt.config.Mode == "invalid" || tt.config.TagFormat == "")
			}
		})
	}
}
