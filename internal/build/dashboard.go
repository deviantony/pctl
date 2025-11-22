package build

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BuildDashboard provides an interactive TUI for monitoring parallel builds
type BuildDashboard struct {
	program *tea.Program
	model   *dashboardModel
}

type ServiceBuildStatus struct {
	Name        string
	Status      BuildStatus
	CurrentStep int
	TotalSteps  int
	Logs        []string
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

type dashboardModel struct {
	services      map[string]*ServiceBuildStatus
	progressBars  map[string]progress.Model
	viewport      viewport.Model
	width         int
	height        int
	selectedIndex int
	mu            sync.RWMutex
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

// NewBuildDashboard creates an interactive build dashboard
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
		progressBars: progressBars,
		viewport:     viewport.New(80, 20),
	}

	return &BuildDashboard{
		model: model,
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
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 10 // Reserve space for service list

	case UpdateMsg:
		m.mu.Lock()
		if svc, ok := m.services[msg.ServiceName]; ok {
			svc.mu.Lock()
			svc.Status = msg.Status
			svc.CurrentStep = msg.Step
			svc.TotalSteps = msg.Total
			if msg.LogLine != "" {
				svc.Logs = append(svc.Logs, msg.LogLine)
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
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "down", "j":
			if m.selectedIndex < len(m.services)-1 {
				m.selectedIndex++
			}
		}
	}

	return m, nil
}

func (m *dashboardModel) View() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("13")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

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

	header := fmt.Sprintf("Building Services | Complete: %d | Building: %d | Failed: %d | Total: %d",
		completed, building, failed, len(m.services))
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Service list with progress bars
	serviceListStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1)

	var serviceList strings.Builder
	i := 0
	for _, svc := range m.services {
		svc.mu.Lock()

		// Status icon and name
		line := fmt.Sprintf("%s %-20s ", svc.Status.String(), svc.Name)

		// Progress bar
		if svc.TotalSteps > 0 {
			percent := float64(svc.CurrentStep) / float64(svc.TotalSteps)
			progBar := m.progressBars[svc.Name].ViewAs(percent)
			line += progBar + " "
			line += fmt.Sprintf("%d/%d ", svc.CurrentStep, svc.TotalSteps)
		} else {
			line += strings.Repeat("░", 20) + " "
		}

		// Duration
		var duration time.Duration
		if svc.Status == StatusBuilding {
			duration = time.Since(svc.StartTime)
		} else if !svc.EndTime.IsZero() {
			duration = svc.EndTime.Sub(svc.StartTime)
		}
		if duration > 0 {
			line += fmt.Sprintf("(%s)", duration.Round(time.Second))
		}

		// Highlight selected service
		if i == m.selectedIndex {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("237")).
				Render(line)
		}

		serviceList.WriteString(line + "\n")
		svc.mu.Unlock()
		i++
	}

	b.WriteString(serviceListStyle.Render(serviceList.String()))
	b.WriteString("\n\n")

	// Log viewer for selected service
	if m.selectedIndex < len(m.services) {
		// Find selected service (need to iterate since map order is random)
		i := 0
		for _, svc := range m.services {
			if i == m.selectedIndex {
				svc.mu.Lock()
				logStyle := lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("238")).
					Padding(1)

				logHeader := fmt.Sprintf("── Logs: %s ──", svc.Name)
				b.WriteString(logHeader + "\n")

				// Show last 10 log lines
				startIdx := 0
				if len(svc.Logs) > 10 {
					startIdx = len(svc.Logs) - 10
				}
				logContent := strings.Join(svc.Logs[startIdx:], "\n")
				b.WriteString(logStyle.Render(logContent))

				svc.mu.Unlock()
				break
			}
			i++
		}
	}

	b.WriteString("\n\nControls: ↑/k up | ↓/j down | q quit")

	return b.String()
}

// InteractiveBuildLogger implements BuildLogger with dashboard integration
type InteractiveBuildLogger struct {
	dashboard *BuildDashboard
	fallback  BuildLogger
}

// NewInteractiveBuildLogger creates a logger that updates the dashboard
func NewInteractiveBuildLogger(dashboard *BuildDashboard, fallback BuildLogger) *InteractiveBuildLogger {
	return &InteractiveBuildLogger{
		dashboard: dashboard,
		fallback:  fallback,
	}
}

func (l *InteractiveBuildLogger) LogService(serviceName, message string) {
	// Parse progress from message if present
	// Extract "Step X/Y" pattern

	// Update dashboard
	l.dashboard.UpdateService(serviceName, StatusBuilding, 0, 0, message, nil)

	// Also log to fallback for non-interactive mode
	if l.fallback != nil {
		l.fallback.LogService(serviceName, message)
	}
}

func (l *InteractiveBuildLogger) LogInfo(message string) {
	if l.fallback != nil {
		l.fallback.LogInfo(message)
	}
}

func (l *InteractiveBuildLogger) LogWarn(message string) {
	if l.fallback != nil {
		l.fallback.LogWarn(message)
	}
}

func (l *InteractiveBuildLogger) LogError(message string) {
	if l.fallback != nil {
		l.fallback.LogError(message)
	}
}

func (l *InteractiveBuildLogger) UpdateProgress(serviceName string, current, total int) {
	l.dashboard.UpdateService(serviceName, StatusBuilding, current, total, "", nil)

	if fb, ok := l.fallback.(interface{ UpdateProgress(string, int, int) }); ok {
		fb.UpdateProgress(serviceName, current, total)
	}
}
