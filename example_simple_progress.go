package main

// EXAMPLE: Minimal code to add progress bars to builds
// This shows the 3 simple changes needed

// STEP 1: Add to internal/build/logger.go (line 23)
/*
type StyledBuildLogger struct {
	prefix       string
	mu           sync.Mutex
	// ... existing styles ...

	// NEW: Add progress tracking
	progressBars map[string]progress.Model
	progressMu   sync.Mutex
}
*/

// STEP 2: Add method to internal/build/logger.go (after line 86)
/*
import "github.com/charmbracelet/bubbles/progress"

func (l *StyledBuildLogger) UpdateProgress(serviceName string, current, total int) {
	l.progressMu.Lock()
	defer l.progressMu.Unlock()

	if l.progressBars == nil {
		l.progressBars = make(map[string]progress.Model)
	}

	if _, exists := l.progressBars[serviceName]; !exists {
		l.progressBars[serviceName] = progress.New(progress.WithDefaultGradient())
	}

	percent := float64(current) / float64(total)
	bar := l.progressBars[serviceName].ViewAs(percent)

	// Print inline progress (overwrites previous line)
	fmt.Printf("\r%s %s %s [%d/%d]",
		l.styleBadge.Render(l.prefix),
		l.styleBadge.Copy().Foreground(lipgloss.Color("219")).Render(serviceName),
		bar,
		current,
		total,
	)

	// New line when complete
	if current == total {
		fmt.Println()
	}
}
*/

// STEP 3: Update internal/build/orchestrator.go buildRemote() (line 214)
/*
import (
	"regexp"
	"strconv"
)

func (bo *BuildOrchestrator) buildRemote(serviceInfo compose.ServiceBuildInfo, imageTag string) BuildResult {
	serviceName := serviceInfo.ServiceName

	// ... existing setup code ...

	stepRegex := regexp.MustCompile(`Step (\d+)/(\d+)`)

	err = bo.client.BuildImage(bo.envID, ctxTar, buildOpts, func(line string) {
		// NEW: Extract and report progress
		if matches := stepRegex.FindStringSubmatch(line); len(matches) == 3 {
			current, _ := strconv.Atoi(matches[1])
			total, _ := strconv.Atoi(matches[2])

			// Check if logger supports progress
			if pl, ok := bo.logger.(interface{ UpdateProgress(string, int, int) }); ok {
				pl.UpdateProgress(serviceName, current, total)
			}
		}

		// Always log the line
		bo.logger.LogService(serviceName, line)
	})

	// ... rest of code ...
}
*/

// RESULT: You'll see output like:
//
// BUILD  frontend  ████████████░░░░░░░░ [3/5]
// BUILD  backend   ██████░░░░░░░░░░░░░░ [2/7]
//
// With real-time updates as builds progress!
