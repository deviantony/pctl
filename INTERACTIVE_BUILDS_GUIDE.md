# Interactive Remote Builds - Implementation Guide

## Current State ✅

Your codebase **already has** the foundation for interactive builds:

- ✅ **Real-time streaming**: `internal/portainer/client.go:360-366` streams JSON lines from Docker API
- ✅ **Log parsing**: `internal/build/logger.go` extracts build steps, errors, and progress
- ✅ **Parallel execution**: `internal/build/orchestrator.go` builds multiple services concurrently
- ✅ **TUI libraries**: Already using `charmbracelet/bubbletea`, `bubbles`, and `lipgloss`

## What's Missing 🎯

- ❌ Progress bars showing build completion percentage
- ❌ Visual feedback for parallel builds (hard to track multiple services)
- ❌ Interactive dashboard with real-time updates
- ❌ Build timing and performance metrics

## Implementation Options

### Option A: Simple Progress Bars (⏱️ 1-2 hours)

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

### Option B: Full Interactive Dashboard (⏱️ 4-6 hours)

**What you get:**
```
┌─ Building Services | Complete: 2 | Building: 1 | Failed: 0 | Total: 5 ─┐
│                                                                          │
│  ✓ frontend      ████████████████████ 100%   (45s)                      │
│  ● backend       ████████░░░░░░░░░░░░  60%   (12s)                      │
│  ⏳ database     ░░░░░░░░░░░░░░░░░░░░   0%   (queue)                    │
│  ✓ nginx         ████████████████████ 100%   (8s)                       │
│  ● worker        ██████░░░░░░░░░░░░░░  40%   (22s)                      │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘

── Logs: backend ─────────────────────────────────────────────────────────
Step 6/10 : RUN npm install
 ---> Running in a1b2c3d4e5f6
npm WARN deprecated package@1.0.0
Successfully built image

Controls: ↑/k up | ↓/j down | q quit
```

**Changes needed:**
1. Use `internal/build/dashboard.go` (already created for you)
2. Integrate into `cmd/deploy/deploy.go` or `cmd/redeploy/redeploy.go`
3. Add terminal detection with `golang.org/x/term/isatty`

**See:** `internal/build/dashboard_integration_example.go:9-49`

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

## Recommended Implementation Path

### Phase 1: Start with Progress Bars ⭐

**Why:** Immediate visual feedback with minimal code changes

**Steps:**
```bash
# 1. Add progress dependencies (already have bubbles!)
go get github.com/charmbracelet/bubbles/progress

# 2. Modify 3 files:
#    - internal/build/logger.go (add UpdateProgress method)
#    - internal/build/orchestrator.go (add to BuildLogger interface)
#    - internal/build/orchestrator.go (extract Step X/Y in buildRemote)

# 3. Test
pctl deploy --stack my-stack --environment 1
```

**Files to change:**
- `internal/build/logger.go` (add lines from `example_simple_progress.go`)
- `internal/build/orchestrator.go` (update callback in `buildRemote()`)

### Phase 2: Add Interactive Dashboard

**Why:** Better experience for multi-service builds

**Steps:**
```bash
# 1. Add terminal detection
go get golang.org/x/term

# 2. Integrate dashboard.go
#    - Modify cmd/deploy/deploy.go
#    - Check if stdout is a terminal
#    - Launch dashboard TUI if interactive
#    - Fall back to regular logs if not

# 3. Test both modes
pctl deploy --stack my-stack --environment 1              # Interactive
pctl deploy --stack my-stack --environment 1 > log.txt   # Non-interactive
```

**Files to change:**
- `cmd/deploy/deploy.go` (see `dashboard_integration_example.go:9-49`)
- `cmd/redeploy/redeploy.go` (same pattern)

### Phase 3: Enhancements

Add features based on user feedback:
- Cancellation (high priority)
- Log export (useful for debugging)
- Analytics (nice to have)

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
