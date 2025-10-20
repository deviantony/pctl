package build

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strconv"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/deviantony/pctl/internal/compose"
	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/portainer"
)

// BuildOrchestrator coordinates the build process for multiple services
type BuildOrchestrator struct {
	client    *portainer.Client
	config    *config.BuildConfig
	envID     int
	stackName string
	logger    BuildLogger
}

// BuildLogger interface for logging build output
type BuildLogger interface {
	LogService(serviceName, message string)
	LogInfo(message string)
	LogWarn(message string)
	LogError(message string)
}

// BuildResult represents the result of building a service
type BuildResult struct {
	ServiceName string
	ImageTag    string
	Success     bool
	Error       error
}

// NewBuildOrchestrator creates a new build orchestrator
func NewBuildOrchestrator(client *portainer.Client, buildConfig *config.BuildConfig, envID int, stackName string, logger BuildLogger) *BuildOrchestrator {
	return &BuildOrchestrator{
		client:    client,
		config:    buildConfig,
		envID:     envID,
		stackName: stackName,
		logger:    logger,
	}
}

// BuildServices builds all services with build directives
func (bo *BuildOrchestrator) BuildServices(servicesWithBuild []compose.ServiceBuildInfo) (map[string]string, error) {
	if len(servicesWithBuild) == 0 {
		return make(map[string]string), nil
	}

	bo.logger.LogInfo(fmt.Sprintf("Building %d service(s) with build directives", len(servicesWithBuild)))

	// Determine parallelism
	parallel := bo.getParallelism()
	bo.logger.LogInfo(fmt.Sprintf("Using parallelism: %d", parallel))

	// Create semaphore for controlling parallelism
	semaphore := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	results := make(chan BuildResult, len(servicesWithBuild))

	// Build services in parallel
	for _, service := range servicesWithBuild {
		wg.Add(1)
		go func(serviceInfo compose.ServiceBuildInfo) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result := bo.buildService(serviceInfo)
			results <- result
		}(service)
	}

	// Wait for all builds to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	imageTags := make(map[string]string)
	var buildErrors []error

	for result := range results {
		if result.Success {
			imageTags[result.ServiceName] = result.ImageTag
			bo.logger.LogInfo(fmt.Sprintf("✓ Built %s -> %s", result.ServiceName, result.ImageTag))
		} else {
			buildErrors = append(buildErrors, fmt.Errorf("failed to build %s: %w", result.ServiceName, result.Error))
			bo.logger.LogError(fmt.Sprintf("✗ Failed to build %s: %v", result.ServiceName, result.Error))
		}
	}

	// Check for build failures
	if len(buildErrors) > 0 {
		return nil, fmt.Errorf("build failed for %d service(s): %v", len(buildErrors), buildErrors[0])
	}

	bo.logger.LogInfo(fmt.Sprintf("Successfully built %d service(s)", len(imageTags)))
	return imageTags, nil
}

// buildService builds a single service
func (bo *BuildOrchestrator) buildService(serviceInfo compose.ServiceBuildInfo) BuildResult {
	serviceName := serviceInfo.ServiceName
	bo.logger.LogService(serviceName, "Starting build...")

	// Generate content hash
	hasher := NewContentHasher()
	contentHash, err := hasher.HashBuildContext(
		serviceInfo.ContextPath,
		serviceInfo.Build.Dockerfile,
		serviceInfo.Build.Args,
	)
	if err != nil {
		return BuildResult{
			ServiceName: serviceName,
			Success:     false,
			Error:       fmt.Errorf("failed to generate content hash: %w", err),
		}
	}

	// Generate image tag
	tagGenerator := NewTagGenerator(bo.stackName, bo.config.TagFormat)
	imageTag := tagGenerator.GenerateTag(serviceName, contentHash)

	// Check if image already exists (unless force build is enabled)
	if !bo.config.ForceBuild {
		exists, err := bo.client.ImageExists(bo.envID, imageTag)
		if err != nil {
			bo.logger.LogWarn(fmt.Sprintf("Could not check if image exists for %s: %v", serviceName, err))
		} else if exists {
			// Styled message for unchanged service (skipping build)
			skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true)
			bo.logger.LogService(serviceName, skipStyle.Render("No changes detected; skipping build")+fmt.Sprintf(" (image: %s)", imageTag))
			return BuildResult{
				ServiceName: serviceName,
				ImageTag:    imageTag,
				Success:     true,
			}
		}
	}

	// Styled message prior to build (force implies no-cache)
	if bo.config.ForceBuild {
		// Force rebuild requested via CLI/config
		forceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
		bo.logger.LogService(serviceName, forceStyle.Render("Force rebuild requested; rebuilding service (no-cache)"))
	} else {
		changeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
		bo.logger.LogService(serviceName, changeStyle.Render("Changes detected; triggering build"))
	}

	// Build based on mode
	switch bo.config.Mode {
	case config.BuildModeRemoteBuild:
		return bo.buildRemote(serviceInfo, imageTag)
	case config.BuildModeLoad:
		return bo.buildLocal(serviceInfo, imageTag)
	default:
		return BuildResult{
			ServiceName: serviceName,
			Success:     false,
			Error:       fmt.Errorf("unsupported build mode: %s", bo.config.Mode),
		}
	}
}

// buildRemote builds the service on the remote Docker engine
func (bo *BuildOrchestrator) buildRemote(serviceInfo compose.ServiceBuildInfo, imageTag string) BuildResult {
	serviceName := serviceInfo.ServiceName
	bo.logger.LogService(serviceName, "Building on remote engine...")

	// Create context tar stream
	streamer := NewContextTarStreamer(bo.config.WarnThresholdMB)
	ctxTar, err := streamer.CreateTarStream(serviceInfo.ContextPath)
	if err != nil {
		return BuildResult{
			ServiceName: serviceName,
			Success:     false,
			Error:       fmt.Errorf("failed to create context tar: %w", err),
		}
	}
	defer ctxTar.Close()

	// Prepare build options (force build implies no-cache)
	buildOpts := portainer.BuildOptions{
		Tag:        imageTag,
		Dockerfile: serviceInfo.Build.Dockerfile,
		BuildArgs:  serviceInfo.Build.Args,
		Target:     serviceInfo.Build.Target,
		NoCache:    bo.config.ForceBuild,
	}

	// Merge extra build args
	for key, value := range bo.config.ExtraBuildArgs {
		if buildOpts.BuildArgs == nil {
			buildOpts.BuildArgs = make(map[string]string)
		}
		buildOpts.BuildArgs[key] = value
	}

	// Build on remote
	err = bo.client.BuildImage(bo.envID, ctxTar, buildOpts, func(line string) {
		bo.logger.LogService(serviceName, line)
	})

	if err != nil {
		return BuildResult{
			ServiceName: serviceName,
			Success:     false,
			Error:       fmt.Errorf("remote build failed: %w", err),
		}
	}

	return BuildResult{
		ServiceName: serviceName,
		ImageTag:    imageTag,
		Success:     true,
	}
}

// buildLocal builds the service locally and loads it to the remote engine
func (bo *BuildOrchestrator) buildLocal(serviceInfo compose.ServiceBuildInfo, imageTag string) BuildResult {
	serviceName := serviceInfo.ServiceName
	bo.logger.LogService(serviceName, "Building locally...")

	// Build locally using docker buildx
	imageTar, err := bo.buildLocalImage(serviceInfo, imageTag)
	if err != nil {
		return BuildResult{
			ServiceName: serviceName,
			Success:     false,
			Error:       fmt.Errorf("local build failed: %w", err),
		}
	}
	defer imageTar.Close()

	// Load image to remote engine
	bo.logger.LogService(serviceName, "Loading image to remote engine...")
	err = bo.client.LoadImage(bo.envID, imageTar, func(line string) {
		bo.logger.LogService(serviceName, line)
	})

	if err != nil {
		return BuildResult{
			ServiceName: serviceName,
			Success:     false,
			Error:       fmt.Errorf("failed to load image: %w", err),
		}
	}

	return BuildResult{
		ServiceName: serviceName,
		ImageTag:    imageTag,
		Success:     true,
	}
}

// buildLocalImage builds an image locally and returns a tar stream
func (bo *BuildOrchestrator) buildLocalImage(serviceInfo compose.ServiceBuildInfo, imageTag string) (io.ReadCloser, error) {
	// Create pipe for streaming
	reader, writer := io.Pipe()

	// Start goroutine to build and stream
	go func() {
		defer writer.Close()

		// Build command arguments
		args := []string{"buildx", "build"}

		// Add platforms
		for _, platform := range bo.config.Platforms {
			args = append(args, "--platform", platform)
		}

		// Add output type and progress format (tar stream on stdout, logs to stderr)
		args = append(args, "--output", "type=docker,dest=-")
		args = append(args, "--progress", "plain")

		// Add tag
		args = append(args, "-t", imageTag)

		// Add no-cache if force build is specified
		if bo.config.ForceBuild {
			args = append(args, "--no-cache")
		}

		// Add build args
		for key, value := range serviceInfo.Build.Args {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
		}

		// Add extra build args
		for key, value := range bo.config.ExtraBuildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
		}

		// Add target if specified
		if serviceInfo.Build.Target != "" {
			args = append(args, "--target", serviceInfo.Build.Target)
		}

		// Add context path
		args = append(args, serviceInfo.ContextPath)

		// Execute docker buildx build
		cmd := exec.Command("docker", args...)

		// Stream tar archive to the pipe via stdout ONLY
		cmd.Stdout = writer

		// Capture stderr separately and log lines (to avoid corrupting tar stream)
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			writer.CloseWithError(fmt.Errorf("failed to open stderr pipe: %w", err))
			return
		}
		go func(svc string) {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				bo.logger.LogService(svc, scanner.Text())
			}
		}(serviceInfo.ServiceName)

		if err := cmd.Run(); err != nil {
			writer.CloseWithError(fmt.Errorf("docker buildx build failed: %w", err))
			return
		}
	}()

	return reader, nil
}

// getParallelism determines the number of parallel builds
func (bo *BuildOrchestrator) getParallelism() int {
	if bo.config.Parallel == config.BuildParallelAuto {
		// Try to get remote CPU count
		info, err := bo.client.GetDockerInfo(bo.envID)
		if err != nil {
			// Fallback to local CPU count
			return max(1, runtime.NumCPU()-1)
		}

		// Extract CPU count from Docker info
		if ncpus, ok := info["NCPU"]; ok {
			if cpuCount, ok := ncpus.(float64); ok {
				return max(1, int(cpuCount)-1)
			}
		}

		// Fallback to local CPU count
		return max(1, runtime.NumCPU()-1)
	}

	// Parse explicit parallelism value
	parallel, err := strconv.Atoi(bo.config.Parallel)
	if err != nil {
		return 1 // Default to sequential
	}

	return max(1, parallel)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SimpleBuildLogger is a simple implementation of BuildLogger
type SimpleBuildLogger struct {
	prefix string
}

// NewSimpleBuildLogger creates a new simple build logger
func NewSimpleBuildLogger(prefix string) *SimpleBuildLogger {
	return &SimpleBuildLogger{prefix: prefix}
}

// LogService logs a service-specific message
func (sbl *SimpleBuildLogger) LogService(serviceName, message string) {
	fmt.Printf("[%s] %s: %s\n", sbl.prefix, serviceName, message)
}

// LogInfo logs an info message
func (sbl *SimpleBuildLogger) LogInfo(message string) {
	fmt.Printf("[%s] %s\n", sbl.prefix, message)
}

// LogWarn logs a warning message
func (sbl *SimpleBuildLogger) LogWarn(message string) {
	fmt.Printf("[%s] WARNING: %s\n", sbl.prefix, message)
}

// LogError logs an error message
func (sbl *SimpleBuildLogger) LogError(message string) {
	fmt.Printf("[%s] ERROR: %s\n", sbl.prefix, message)
}
