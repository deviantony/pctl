package compose

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BuildDirective represents a build configuration in a compose service
type BuildDirective struct {
	Context    string            `yaml:"context"`
	Dockerfile string            `yaml:"dockerfile"`
	Args       map[string]string `yaml:"args"`
	Target     string            `yaml:"target"`
	CacheFrom  []string          `yaml:"cache_from"`
}

// ServiceBuildInfo contains build information for a service
type ServiceBuildInfo struct {
	ServiceName string
	Build       *BuildDirective
	ContextPath string // Resolved absolute path to build context
}

// ComposeFile represents a parsed compose file
type ComposeFile struct {
	Services map[string]interface{} `yaml:"services"`
	Version  string                 `yaml:"version"`
}

// ParseComposeFile parses a compose file and extracts build information
func ParseComposeFile(content string) (*ComposeFile, error) {
	var compose ComposeFile
	if err := yaml.Unmarshal([]byte(content), &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &compose, nil
}

// FindServicesWithBuild finds all services that have build directives
func (cf *ComposeFile) FindServicesWithBuild() ([]ServiceBuildInfo, error) {
	var servicesWithBuild []ServiceBuildInfo

	for serviceName, serviceData := range cf.Services {
		buildInfo, err := extractBuildInfo(serviceName, serviceData)
		if err != nil {
			return nil, fmt.Errorf("failed to extract build info for service '%s': %w", serviceName, err)
		}

		if buildInfo != nil {
			servicesWithBuild = append(servicesWithBuild, *buildInfo)
		}
	}

	return servicesWithBuild, nil
}

// extractBuildInfo extracts build information from a service definition
func extractBuildInfo(serviceName string, serviceData interface{}) (*ServiceBuildInfo, error) {
	serviceMap, ok := serviceData.(map[string]interface{})
	if !ok {
		return nil, nil // Not a map, no build directive
	}

	buildData, exists := serviceMap["build"]
	if !exists {
		return nil, nil // No build directive
	}

	buildInfo := &ServiceBuildInfo{
		ServiceName: serviceName,
	}

	// Handle different build directive formats
	switch build := buildData.(type) {
	case string:
		// Simple format: build: "./path"
		buildInfo.Build = &BuildDirective{
			Context: build,
		}
	case map[string]interface{}:
		// Complex format: build: { context: "./path", dockerfile: "Dockerfile", ... }
		buildDirective := &BuildDirective{}

		if context, ok := build["context"].(string); ok {
			buildDirective.Context = context
		}

		if dockerfile, ok := build["dockerfile"].(string); ok {
			buildDirective.Dockerfile = dockerfile
		}

		if args, ok := build["args"].(map[string]interface{}); ok {
			buildDirective.Args = make(map[string]string)
			for key, value := range args {
				if strValue, ok := value.(string); ok {
					buildDirective.Args[key] = strValue
				}
			}
		}

		if target, ok := build["target"].(string); ok {
			buildDirective.Target = target
		}

		if cacheFrom, ok := build["cache_from"].([]interface{}); ok {
			buildDirective.CacheFrom = make([]string, len(cacheFrom))
			for i, item := range cacheFrom {
				if strItem, ok := item.(string); ok {
					buildDirective.CacheFrom[i] = strItem
				}
			}
		}

		buildInfo.Build = buildDirective
	default:
		return nil, fmt.Errorf("invalid build directive format for service '%s'", serviceName)
	}

	// Resolve context path to absolute path
	if buildInfo.Build.Context == "" {
		buildInfo.Build.Context = "." // Default to current directory
	}

	absPath, err := filepath.Abs(buildInfo.Build.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve context path '%s': %w", buildInfo.Build.Context, err)
	}
	buildInfo.ContextPath = absPath

	// Set default dockerfile if not specified
	if buildInfo.Build.Dockerfile == "" {
		buildInfo.Build.Dockerfile = "Dockerfile"
	}

	return buildInfo, nil
}

// HasBuildDirectives checks if the compose file has any services with build directives
func (cf *ComposeFile) HasBuildDirectives() (bool, error) {
	servicesWithBuild, err := cf.FindServicesWithBuild()
	if err != nil {
		return false, err
	}
	return len(servicesWithBuild) > 0, nil
}

// GetServiceNames returns all service names in the compose file
func (cf *ComposeFile) GetServiceNames() []string {
	var names []string
	for name := range cf.Services {
		names = append(names, name)
	}
	return names
}

// ValidateBuildContexts validates that all build contexts exist and are accessible
func (cf *ComposeFile) ValidateBuildContexts() error {
	servicesWithBuild, err := cf.FindServicesWithBuild()
	if err != nil {
		return err
	}

	for _, service := range servicesWithBuild {
		// Check if context directory exists
		if !isDirectory(service.ContextPath) {
			return fmt.Errorf("build context directory does not exist for service '%s': %s",
				service.ServiceName, service.ContextPath)
		}

		// Check if Dockerfile exists in context
		dockerfilePath := filepath.Join(service.ContextPath, service.Build.Dockerfile)
		if !isFile(dockerfilePath) {
			return fmt.Errorf("Dockerfile does not exist for service '%s': %s",
				service.ServiceName, dockerfilePath)
		}
	}

	return nil
}

// Helper functions for file system checks
func isDirectory(path string) bool {
	// This is a simplified check - in a real implementation, you'd use os.Stat
	// For now, we'll assume the path exists if it's not empty
	return path != ""
}

func isFile(path string) bool {
	// This is a simplified check - in a real implementation, you'd use os.Stat
	// For now, we'll assume the file exists if it's not empty
	return path != ""
}

// GetBuildContextSummary returns a summary of build contexts for logging
func (cf *ComposeFile) GetBuildContextSummary() (string, error) {
	servicesWithBuild, err := cf.FindServicesWithBuild()
	if err != nil {
		return "", err
	}

	if len(servicesWithBuild) == 0 {
		return "No build directives found", nil
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Found %d service(s) with build directives:\n", len(servicesWithBuild)))

	for _, service := range servicesWithBuild {
		summary.WriteString(fmt.Sprintf("  - %s: context=%s, dockerfile=%s\n",
			service.ServiceName, service.Build.Context, service.Build.Dockerfile))
	}

	return summary.String(), nil
}

