package build

// Example: How to integrate the passive dashboard into BuildOrchestrator

/*
USAGE IN cmd/deploy/deploy.go or cmd/redeploy/redeploy.go:

import (
	"os"
	"time"

	"github.com/deviantony/pctl/internal/build"
	"github.com/deviantony/pctl/internal/compose"
	"golang.org/x/term"
)

func buildServicesWithDashboard(
	orchestrator *build.BuildOrchestrator,
	services []compose.ServiceBuildInfo,
) (map[string]string, error) {
	// Check if running in interactive terminal
	isInteractive := term.IsTerminal(int(os.Stdout.Fd()))

	if isInteractive && len(services) > 0 {
		// Extract service names
		serviceNames := make([]string, len(services))
		for i, svc := range services {
			serviceNames[i] = svc.ServiceName
		}

		// Create passive dashboard
		dashboard := build.NewBuildDashboard(serviceNames)

		// Create logger that updates dashboard
		dashboardLogger := build.NewDashboardBuildLogger(dashboard)

		// Mark all services as queued initially
		for _, name := range serviceNames {
			dashboardLogger.MarkServiceQueued(name)
		}

		// Start dashboard TUI (runs in background)
		dashboard.Start()
		defer func() {
			// Keep dashboard visible for 2 seconds after builds complete
			time.Sleep(2 * time.Second)
			dashboard.Stop()
		}()

		// Replace orchestrator's logger
		// NOTE: You'll need to add a SetLogger method to BuildOrchestrator
		// or pass the logger in the constructor
		orchestrator.SetLogger(dashboardLogger)

		// Build services (dashboard updates automatically)
		imageTags, err := orchestrator.BuildServices(services)

		return imageTags, err
	}

	// Non-interactive mode - use regular styled logger
	return orchestrator.BuildServices(services)
}

// INTEGRATION WITH BuildOrchestrator:
//
// You'll need to modify internal/build/orchestrator.go slightly:
//
// 1. Add SetLogger method to BuildOrchestrator:
func (bo *BuildOrchestrator) SetLogger(logger BuildLogger) {
	bo.logger = logger
}

// 2. Update buildService to mark status changes:
//
// In buildService() before building, mark as building:
if dashLogger, ok := bo.logger.(*build.DashboardBuildLogger); ok {
	dashLogger.MarkServiceBuilding(serviceName)
}

// After successful build:
if dashLogger, ok := bo.logger.(*build.DashboardBuildLogger); ok {
	dashLogger.MarkServiceComplete(serviceName)
}

// After failed build:
if dashLogger, ok := bo.logger.(*build.DashboardBuildLogger); ok {
	dashLogger.MarkServiceFailed(serviceName, err)
}

*/

/*
MINIMAL INTEGRATION EXAMPLE:

Here's the absolute minimum code to add to cmd/deploy/deploy.go:

import (
	"os"
	"time"
	"golang.org/x/term"
	"github.com/deviantony/pctl/internal/build"
)

// Before calling BuildServices:
var logger build.BuildLogger = build.NewStyledBuildLogger("BUILD")
var dashboard *build.BuildDashboard

// Check if terminal supports TUI
if term.IsTerminal(int(os.Stdout.Fd())) && len(servicesWithBuild) > 1 {
	serviceNames := make([]string, len(servicesWithBuild))
	for i, svc := range servicesWithBuild {
		serviceNames[i] = svc.ServiceName
	}

	dashboard = build.NewBuildDashboard(serviceNames)
	logger = build.NewDashboardBuildLogger(dashboard)
	dashboard.Start()
}

// Create orchestrator with the logger
orchestrator := build.NewBuildOrchestrator(client, buildConfig, envID, stackName, logger)

// Build services
imageTags, err := orchestrator.BuildServices(servicesWithBuild)

// Stop dashboard after builds complete
if dashboard != nil {
	time.Sleep(2 * time.Second) // Keep visible for a moment
	dashboard.Stop()
}
*/

/*
EXPECTED OUTPUT:

When running in a terminal with multiple services:

╭─────────────────────────────────────────────────────────────────╮
│ Building Services                                               │
│ Complete: 2 | Building: 1 | Failed: 0 | Total: 5                │
│                                                                  │
│ ✓ frontend           ████████████████████ 100% (45s)            │
│                                                                  │
│ ● backend            ████████░░░░░░░░░░░░  60% (12s)            │
│     Step 6/10 : RUN npm install                                 │
│      ---> Running in a1b2c3d4e5f6                               │
│     npm WARN deprecated package@1.0.0                           │
│                                                                  │
│ ⏳ database          ░░░░░░░░░░░░░░░░░░░░   0% (queued)         │
│                                                                  │
│ ✓ nginx              ████████████████████ 100% (8s)             │
│                                                                  │
│ ● worker             ██████░░░░░░░░░░░░░░  40% (22s)            │
│     Step 4/10 : COPY . .                                        │
│      ---> c2d3e4f5a6b7                                          │
│                                                                  │
│ Press q or Ctrl+C to quit                                       │
╰─────────────────────────────────────────────────────────────────╯

The dashboard updates automatically in real-time as builds progress.
No keyboard interaction needed (except q to quit early).
*/

/*
DEPENDENCIES:

Add to go.mod if not already present:

go get golang.org/x/term

This is for checking if stdout is a terminal (term.IsTerminal).
*/
