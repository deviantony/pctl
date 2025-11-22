package build

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BuildDashboard provides a passive TUI for monitoring parallel builds
// It auto-updates and displays all services without requiring user interaction
type BuildDashboard struct {
	program      *tea.Program
	model        *dashboardModel
	serviceOrder []string // Preserve service order for consistent display
}

type ServiceBuildStatus struct {
	Name        string
	Status      BuildStatus
	CurrentStep int
	TotalSteps  int
	Logs        []string // Keep last few log lines
	StartTime   time.Time
	EndTime     time.Time
	Error       error
	mu          sync.Mutex
}

type BuildStatus int

const (
	StatusQueued BuildStatus = iota
	StatusBuilding
	StatusComplete
	StatusFailed
)

func (s BuildStatus) String() string {
	switch s {
	case StatusQueued:
		return "⏳"
	case StatusBuilding:
		return "●"
	case StatusComplete:
		return "✓"
	case StatusFailed:
		return "✗"
	default:
		return "?"
	}
}

func (s BuildStatus) Color() lipgloss.Color {
	switch s {
	case StatusQueued:
		return lipgloss.Color("8")  // Gray
	case StatusBuilding:
		return lipgloss.Color("12") // Blue
	case StatusComplete:
		return lipgloss.Color("10") // Green
	case StatusFailed:
		return lipgloss.Color("9")  // Red
	default:
		return lipgloss.Color("7")
	}
}

type dashboardModel struct {
	services     map[string]*ServiceBuildStatus
	serviceOrder []string
	progressBars map[string]progress.Model
	width        int
	height       int
	mu           sync.RWMutex
}

// UpdateMsg is sent when a service's status changes
type UpdateMsg struct {
	ServiceName string
	Status      BuildStatus
	Step        int
	Total       int
	LogLine     string
	Error       error
}

// NewBuildDashboard creates a passive build dashboard
func NewBuildDashboard(services []string) *BuildDashboard {
	serviceMap := make(map[string]*ServiceBuildStatus)
	progressBars := make(map[string]progress.Model)

	for _, name := range services {
		serviceMap[name] = &ServiceBuildStatus{
			Name:   name,
			Status: StatusQueued,
			Logs:   []string{},
		}
		progressBars[name] = progress.New(progress.WithDefaultGradient())
	}

	model := &dashboardModel{
		services:     serviceMap,
		serviceOrder: services, // Preserve order
		progressBars: progressBars,
	}

	return &BuildDashboard{
		model:        model,
		serviceOrder: services,
	}
}

// Start launches the TUI
func (bd *BuildDashboard) Start() {
	bd.program = tea.NewProgram(bd.model)
	go func() {
		if _, err := bd.program.Run(); err != nil {
			fmt.Printf("Error running dashboard: %v\n", err)
		}
	}()
}

// UpdateService updates a service's build status
func (bd *BuildDashboard) UpdateService(serviceName string, status BuildStatus, currentStep, totalSteps int, logLine string, err error) {
	if bd.program != nil {
		bd.program.Send(UpdateMsg{
			ServiceName: serviceName,
			Status:      status,
			Step:        currentStep,
			Total:       totalSteps,
			LogLine:     logLine,
			Error:       err,
		})
	}
}

// Stop stops the dashboard
func (bd *BuildDashboard) Stop() {
	if bd.program != nil {
		bd.program.Quit()
	}
}

// Bubbletea Model implementation
func (m *dashboardModel) Init() tea.Cmd {
	return nil
}

func (m *dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case UpdateMsg:
		m.mu.Lock()
		if svc, ok := m.services[msg.ServiceName]; ok {
			svc.mu.Lock()
			svc.Status = msg.Status
			svc.CurrentStep = msg.Step
			svc.TotalSteps = msg.Total
			if msg.LogLine != "" {
				// Keep only last 3 log lines per service
				svc.Logs = append(svc.Logs, msg.LogLine)
				if len(svc.Logs) > 3 {
					svc.Logs = svc.Logs[len(svc.Logs)-3:]
				}
			}
			if msg.Error != nil {
				svc.Error = msg.Error
			}
			if msg.Status == StatusBuilding && svc.StartTime.IsZero() {
				svc.StartTime = time.Now()
			}
			if msg.Status == StatusComplete || msg.Status == StatusFailed {
				svc.EndTime = time.Now()
			}
			svc.mu.Unlock()
		}
		m.mu.Unlock()

	case tea.KeyMsg:
		// Only allow quitting - no other interactions
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *dashboardModel) View() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder

	// Define styles
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("13"))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	// Count statuses
	completed := 0
	building := 0
	failed := 0
	for _, svc := range m.services {
		switch svc.Status {
		case StatusComplete:
			completed++
		case StatusBuilding:
			building++
		case StatusFailed:
			failed++
		}
	}

	// Build the content
	var content strings.Builder
	content.WriteString(headerStyle.Render("Building Services") + "\n")
	content.WriteString(dimStyle.Render(fmt.Sprintf("Complete: %d | Building: %d | Failed: %d | Total: %d",
		completed, building, failed, len(m.services))) + "\n\n")

	// Iterate through services in order
	for _, serviceName := range m.serviceOrder {
		svc, ok := m.services[serviceName]
		if !ok {
			continue
		}

		svc.mu.Lock()

		// Service status line with progress bar
		statusStyle := lipgloss.NewStyle().
			Foreground(svc.Status.Color()).
			Bold(true)

		line := fmt.Sprintf("%s %-20s ", statusStyle.Render(svc.Status.String()), svc.Name)

		// Progress bar
		if svc.TotalSteps > 0 {
			percent := float64(svc.CurrentStep) / float64(svc.TotalSteps)
			progBar := m.progressBars[svc.Name].ViewAs(percent)
			line += progBar + " "
			line += fmt.Sprintf("%3d%% ", int(percent*100))
		} else {
			// Empty progress bar for queued services
			line += strings.Repeat("░", 20) + "   0% "
		}

		// Duration
		var duration time.Duration
		if svc.Status == StatusBuilding && !svc.StartTime.IsZero() {
			duration = time.Since(svc.StartTime)
		} else if !svc.EndTime.IsZero() && !svc.StartTime.IsZero() {
			duration = svc.EndTime.Sub(svc.StartTime)
		}
		if duration > 0 {
			line += dimStyle.Render(fmt.Sprintf("(%s)", duration.Round(time.Second)))
		} else if svc.Status == StatusQueued {
			line += dimStyle.Render("(queued)")
		}

		content.WriteString(line + "\n")

		// Show last few log lines for building services
		if svc.Status == StatusBuilding && len(svc.Logs) > 0 {
			logStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				MarginLeft(2)

			for _, logLine := range svc.Logs {
				// Clean and indent log lines
				cleanedLog := cleanLogLine(logLine)
				if cleanedLog != "" {
					content.WriteString(logStyle.Render("  "+cleanedLog) + "\n")
				}
			}
		}

		// Show error for failed services
		if svc.Status == StatusFailed && svc.Error != nil {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				MarginLeft(2)
			content.WriteString(errorStyle.Render(fmt.Sprintf("  Error: %v", svc.Error)) + "\n")
		}

		content.WriteString("\n")
		svc.mu.Unlock()
	}

	// Add quit hint at bottom
	content.WriteString(dimStyle.Render("Press q or Ctrl+C to quit"))

	// Wrap in border
	b.WriteString(borderStyle.Render(content.String()))

	return b.String()
}

// cleanLogLine removes JSON wrapping and cleans up Docker build output
func cleanLogLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	// If it starts with {, try to parse as JSON
	if line[0] == '{' {
		var m map[string]interface{}
		if err := regexp.MustCompile(`"stream"\s*:\s*"([^"]*)"`)
			.FindStringSubmatch(line); err != nil && len(err) > 1 {
			return strings.TrimSpace(err[1])
		}
	}

	// Return as-is if not JSON or parsing failed
	return line
}

// DashboardBuildLogger implements BuildLogger with dashboard integration
// It's passive - just displays information, no interaction required
type DashboardBuildLogger struct {
	dashboard    *BuildDashboard
	stepRegex    *regexp.Regexp
	serviceState map[string]*serviceState
	mu           sync.Mutex
}

type serviceState struct {
	currentStep int
	totalSteps  int
	status      BuildStatus
}

// NewDashboardBuildLogger creates a logger that updates the dashboard
func NewDashboardBuildLogger(dashboard *BuildDashboard) *DashboardBuildLogger {
	return &DashboardBuildLogger{
		dashboard:    dashboard,
		stepRegex:    regexp.MustCompile(`Step (\d+)/(\d+)`),
		serviceState: make(map[string]*serviceState),
	}
}

func (l *DashboardBuildLogger) LogService(serviceName, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Initialize state if needed
	if l.serviceState[serviceName] == nil {
		l.serviceState[serviceName] = &serviceState{
			status: StatusBuilding,
		}
	}
	state := l.serviceState[serviceName]

	// Parse progress from message "Step X/Y"
	if matches := l.stepRegex.FindStringSubmatch(message); len(matches) == 3 {
		current, _ := strconv.Atoi(matches[1])
		total, _ := strconv.Atoi(matches[2])
		state.currentStep = current
		state.totalSteps = total
	}

	// Check for completion
	if strings.Contains(message, "Successfully built") || strings.Contains(message, "Successfully tagged") {
		state.status = StatusComplete
	}

	// Update dashboard with current state and log line
	l.dashboard.UpdateService(serviceName, state.status, state.currentStep, state.totalSteps, message, nil)
}

func (l *DashboardBuildLogger) LogInfo(message string) {
	// Info messages don't need to update the dashboard
	// Could be displayed in a separate info section if needed
}

func (l *DashboardBuildLogger) LogWarn(message string) {
	// Warnings could be shown in dashboard if needed
}

func (l *DashboardBuildLogger) LogError(message string) {
	// Errors could be shown in dashboard if needed
}

// MarkServiceQueued marks a service as queued
func (l *DashboardBuildLogger) MarkServiceQueued(serviceName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.serviceState[serviceName] == nil {
		l.serviceState[serviceName] = &serviceState{}
	}
	l.serviceState[serviceName].status = StatusQueued
	l.dashboard.UpdateService(serviceName, StatusQueued, 0, 0, "", nil)
}

// MarkServiceBuilding marks a service as building
func (l *DashboardBuildLogger) MarkServiceBuilding(serviceName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.serviceState[serviceName] == nil {
		l.serviceState[serviceName] = &serviceState{}
	}
	l.serviceState[serviceName].status = StatusBuilding
	l.dashboard.UpdateService(serviceName, StatusBuilding, 0, 0, "", nil)
}

// MarkServiceComplete marks a service as complete
func (l *DashboardBuildLogger) MarkServiceComplete(serviceName string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.serviceState[serviceName] == nil {
		l.serviceState[serviceName] = &serviceState{}
	}
	state := l.serviceState[serviceName]
	state.status = StatusComplete
	// Set to 100% when complete
	if state.totalSteps > 0 {
		state.currentStep = state.totalSteps
	}
	l.dashboard.UpdateService(serviceName, StatusComplete, state.currentStep, state.totalSteps, "", nil)
}

// MarkServiceFailed marks a service as failed
func (l *DashboardBuildLogger) MarkServiceFailed(serviceName string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.serviceState[serviceName] == nil {
		l.serviceState[serviceName] = &serviceState{}
	}
	l.serviceState[serviceName].status = StatusFailed
	l.dashboard.UpdateService(serviceName, StatusFailed, 0, 0, "", err)
}
