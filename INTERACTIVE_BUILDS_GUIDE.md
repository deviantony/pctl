# Interactive Remote Builds - Implementation Guide

## Current State ✅

Your codebase **already has** the foundation for interactive builds:

- ✅ **Real-time streaming**: `internal/portainer/client.go:360-366` streams JSON lines from Docker API
- ✅ **Log parsing**: `internal/build/logger.go` extracts build steps, errors, and progress
- ✅ **Parallel execution**: `internal/build/orchestrator.go` builds multiple services concurrently
- ✅ **TUI libraries**: Already using `charmbracelet/bubbletea`, `bubbles`, and `lipgloss`

## What's New 🎯

- ✅ **Passive TUI Dashboard** - Real-time build monitoring with progress bars and live logs
- ✅ **Auto-updating display** - No interaction required, just watch builds progress
- ✅ **Multi-service tracking** - See all builds at once with individual progress
- ✅ **Build timing metrics** - Duration tracking per service

## Recommended: Passive TUI Dashboard ⭐

**What you get:**
```
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
```

**Features:**
- 🎯 **Fully passive** - Just displays information, no interaction needed
- 📊 **Progress bars** - Visual progress for each service
- 📜 **Auto-scrolling logs** - Last 3 lines shown per building service
- ⏱️ **Build timers** - Real-time duration tracking
- 🎨 **Color-coded status** - Queued (gray), Building (blue), Complete (green), Failed (red)
- 🔄 **Real-time updates** - Dashboard refreshes automatically as builds progress

**Already implemented in:**
- `internal/build/dashboard.go` - Complete TUI dashboard
- `internal/build/dashboard_integration_example.go` - Integration guide

---

### Alternative: Simple Progress Bars

**What you get:**
```
BUILD  frontend  ████████████░░░░░░░░ [6/10] (23s)
BUILD  backend   ██████░░░░░░░░░░░░░░ [2/8]  (5s)
```

**Changes needed:**
1. Add `UpdateProgress(serviceName, current, total)` to `BuildLogger` interface
2. Implement progress tracking in `StyledBuildLogger` using `bubbles/progress`
3. Extract "Step X/Y" from Docker output in `buildRemote()` callback

**See:** `example_simple_progress.go` for exact code

---

### Option C: Enhanced Features (⏱️ Ongoing)

Additional capabilities to add later:

1. **Cancellation Support**
   - Gracefully stop builds with Ctrl+C
   - Add context cancellation to `BuildImage()`

2. **Log Export**
   - Save per-service build logs to files
   - Format: JSON or plain text

3. **Build Analytics**
   - Cache hit/miss statistics
   - Build speed tracking over time
   - Slowest build step identification

4. **Resource Monitoring**
   - Real-time CPU/memory usage during builds
   - Network I/O for image pulls
   - Requires Docker stats API integration

## Implementation Guide

### Using the Passive TUI Dashboard

**The dashboard is already fully implemented!** You just need to integrate it into your build commands.

**Step 1: Add terminal detection dependency**
```bash
go get golang.org/x/term
```

**Step 2: Integrate into `cmd/deploy/deploy.go`** (or `cmd/redeploy/redeploy.go`)

See complete example in `internal/build/dashboard_integration_example.go`.

**Minimal integration:**
```go
import (
	"os"
	"time"
	"golang.org/x/term"
	"github.com/deviantony/pctl/internal/build"
)

// Before calling BuildServices:
var logger build.BuildLogger = build.NewStyledBuildLogger("BUILD")
var dashboard *build.BuildDashboard

// Check if terminal supports TUI and multiple services
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
```

**Step 3: Test both modes**
```bash
# Interactive mode (shows TUI dashboard)
pctl deploy --stack my-stack --environment 1

# Non-interactive mode (regular logs)
pctl deploy --stack my-stack --environment 1 > log.txt
```

That's it! The dashboard will:
- ✅ Auto-detect if running in a terminal
- ✅ Fall back to regular logs if not interactive
- ✅ Update in real-time as builds progress
- ✅ Show progress bars, logs, and timings
- ✅ Work with parallel builds out of the box

### Optional Enhancements

Add features based on user feedback:
- Cancellation support (high priority)
- Log export to files (useful for debugging)
- Build analytics and cache statistics (nice to have)

## Docker JSON Stream Format

Understanding what Docker sends helps with parsing:

```json
{"stream":"Step 1/8 : FROM node:18-alpine\n"}
{"stream":" ---> 7d5b57e3d3e5\n"}
{"stream":"Step 2/8 : WORKDIR /app\n"}
{"stream":" ---> Running in a1b2c3d4e5f6\n"}
{"stream":"Removing intermediate container a1b2c3d4e5f6\n"}
{"stream":" ---> c2d3e4f5a6b7\n"}
...
{"aux":{"ID":"sha256:abc123..."}}
{"stream":"Successfully built abc123...\n"}
{"stream":"Successfully tagged myapp:latest\n"}
```

**Error format:**
```json
{"error":"build failed","errorDetail":{"message":"RUN failed with exit code 1"}}
```

**Current parsing:** `internal/build/logger.go:99-133`

## Testing Checklist

- [ ] Single service build shows progress
- [ ] Multiple services build in parallel with separate progress bars
- [ ] Progress resets correctly between builds
- [ ] Works in non-interactive mode (e.g., CI/CD pipelines)
- [ ] Handles build failures gracefully
- [ ] Ctrl+C cancels cleanly
- [ ] Terminal resize handled (for dashboard mode)

## Troubleshooting

### Progress bar not updating
- Check that regex matches Docker output: `Step (\d+)/(\d+)`
- Verify callback is being called: add debug print

### Dashboard crashes on resize
- Ensure `tea.WindowSizeMsg` handler updates viewport dimensions

### Works locally but not in CI
- Add terminal detection: `isatty.IsTerminal(os.Stdout.Fd())`
- Fall back to simple logger when not a TTY

## Additional Resources

- **Charmbracelet Bubbles**: https://github.com/charmbracelet/bubbles
- **Bubbletea Tutorial**: https://github.com/charmbracelet/bubbletea/tree/master/tutorials
- **Docker Build API**: https://docs.docker.com/engine/api/v1.43/#tag/Image/operation/ImageBuild
- **Docker JSON Stream**: Newline-delimited JSON (NDJSON)

## Quick Reference

| Feature | Location | Lines | Purpose |
|---------|----------|-------|---------|
| Streaming | `internal/portainer/client.go` | 360-366 | Reads Docker JSON lines |
| Log parsing | `internal/build/logger.go` | 89-134 | Cleans and styles output |
| Build callback | `internal/build/orchestrator.go` | 214-216 | Receives each log line |
| Parallel builds | `internal/build/orchestrator.go` | 66-88 | Semaphore + goroutines |
| Dashboard UI | `internal/build/dashboard.go` | - | Interactive TUI (created) |

## Next Steps

1. **Try the simple progress bar implementation first** (`example_simple_progress.go`)
2. Test with a multi-service stack
3. Gather feedback from users
4. Add dashboard if needed for complex builds
5. Iterate based on usage patterns

Good luck! 🚀
