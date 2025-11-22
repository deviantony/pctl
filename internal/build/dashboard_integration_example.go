package build

// Example: How to integrate the interactive dashboard into BuildOrchestrator

/*
Usage in cmd/deploy/deploy.go or cmd/redeploy/redeploy.go:

import (
	"github.com/deviantony/pctl/internal/build"
	"os"
)

func buildWithDashboard(orchestrator *build.BuildOrchestrator, services []compose.ServiceBuildInfo) error {
	// Check if running in interactive terminal
	isInteractive := isatty.IsTerminal(os.Stdout.Fd())

	if isInteractive {
		// Extract service names
		serviceNames := make([]string, len(services))
		for i, svc := range services {
			serviceNames[i] = svc.ServiceName
		}

		// Create dashboard
		dashboard := build.NewBuildDashboard(serviceNames)

		// Create logger that updates dashboard
		fallbackLogger := build.NewStyledBuildLogger("BUILD")
		interactiveLogger := build.NewInteractiveBuildLogger(dashboard, fallbackLogger)

		// Start dashboard TUI
		dashboard.Start()
		defer dashboard.Stop()

		// Replace orchestrator's logger
		orchestrator.SetLogger(interactiveLogger)

		// Build services (dashboard will update automatically)
		imageTags, err := orchestrator.BuildServices(services)

		// Keep dashboard open for a moment to show final state
		time.Sleep(2 * time.Second)

		return err
	} else {
		// Non-interactive mode - use regular logger
		return orchestrator.BuildServices(services)
	}
}

// Add this method to BuildOrchestrator to allow logger replacement:
func (bo *BuildOrchestrator) SetLogger(logger BuildLogger) {
	bo.logger = logger
}
*/

// Alternative: Simpler progress bars without full TUI

/*
For a lighter-weight solution, just add progress tracking to StyledBuildLogger:

import (
	"github.com/charmbracelet/bubbles/progress"
	"regexp"
	"strconv"
)

type StyledBuildLogger struct {
	// ... existing fields ...

	// Add progress tracking
	serviceProgress map[string]*serviceProgress
	progressMu      sync.Mutex
}

type serviceProgress struct {
	current int
	total   int
	prog    progress.Model
}

func NewStyledBuildLogger(prefix string) *StyledBuildLogger {
	return &StyledBuildLogger{
		// ... existing initialization ...
		serviceProgress: make(map[string]*serviceProgress),
	}
}

// Add UpdateProgress method
func (l *StyledBuildLogger) UpdateProgress(serviceName string, current, total int) {
	l.progressMu.Lock()
	defer l.progressMu.Unlock()

	if l.serviceProgress[serviceName] == nil {
		l.serviceProgress[serviceName] = &serviceProgress{
			prog: progress.New(progress.WithDefaultGradient()),
		}
	}

	sp := l.serviceProgress[serviceName]
	sp.current = current
	sp.total = total

	percent := float64(current) / float64(total)

	// Print inline progress bar
	fmt.Printf("\r%s %s %s %d/%d",
		l.styleBadge.Render(l.prefix),
		l.styleBadge.Copy().Foreground(lipgloss.Color("219")).Render(serviceName),
		sp.prog.ViewAs(percent),
		current,
		total,
	)

	if current == total {
		fmt.Println() // New line when complete
	}
}

// Then in orchestrator.go buildRemote(), add progress extraction:

func (bo *BuildOrchestrator) buildRemote(serviceInfo compose.ServiceBuildInfo, imageTag string) BuildResult {
	serviceName := serviceInfo.ServiceName

	// ... existing setup code ...

	// Regex to extract "Step X/Y"
	stepRegex := regexp.MustCompile(`Step (\d+)/(\d+)`)

	err = bo.client.BuildImage(bo.envID, ctxTar, buildOpts, func(line string) {
		// Check if logger supports progress
		if progressLogger, ok := bo.logger.(interface{ UpdateProgress(string, int, int) }); ok {
			// Extract step info from line
			if matches := stepRegex.FindStringSubmatch(line); len(matches) == 3 {
				current, _ := strconv.Atoi(matches[1])
				total, _ := strconv.Atoi(matches[2])
				progressLogger.UpdateProgress(serviceName, current, total)
			}
		}

		// Always log the full line
		bo.logger.LogService(serviceName, line)
	})

	// ... rest of code ...
}
*/
