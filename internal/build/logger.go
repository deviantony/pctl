package build

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// StyledBuildLogger is a modern, styled logger for build output.
// It implements BuildLogger and cleans Docker JSON lines into readable text.
type StyledBuildLogger struct {
	prefix       string
	mu           sync.Mutex
	styleBadge   lipgloss.Style
	styleInfo    lipgloss.Style
	styleSuccess lipgloss.Style
	styleWarn    lipgloss.Style
	styleError   lipgloss.Style
	styleDim     lipgloss.Style
}

// NewStyledBuildLogger returns a logger with consistent, modern styles.
func NewStyledBuildLogger(prefix string) *StyledBuildLogger {
	return &StyledBuildLogger{
		prefix:       prefix,
		styleBadge:   lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Background(lipgloss.Color("236")).Padding(0, 1).Bold(true),
		styleInfo:    lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		styleSuccess: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		styleWarn:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		styleError:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		styleDim:     lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
}

// LogService logs a service-specific message
func (l *StyledBuildLogger) LogService(serviceName, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	text := l.cleanDockerLine(message)
	line := fmt.Sprintf("%s %s %s",
		l.styleBadge.Render(l.prefix),
		l.styleBadge.Copy().Foreground(lipgloss.Color("219")).Render(serviceName),
		text,
	)
	fmt.Println(line)
}

// LogInfo logs an info message
func (l *StyledBuildLogger) LogInfo(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("%s %s",
		l.styleBadge.Render(l.prefix),
		l.styleInfo.Render(message),
	)
	fmt.Println(line)
}

// LogWarn logs a warning message
func (l *StyledBuildLogger) LogWarn(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("%s %s",
		l.styleBadge.Render(l.prefix),
		l.styleWarn.Render("WARN: "+message),
	)
	fmt.Println(line)
}

// LogError logs an error message
func (l *StyledBuildLogger) LogError(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf("%s %s",
		l.styleBadge.Render(l.prefix),
		l.styleError.Render("ERROR: "+message),
	)
	fmt.Println(line)
}

// cleanDockerLine parses docker-build JSON lines and returns a concise, pretty string.
func (l *StyledBuildLogger) cleanDockerLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	if line[0] != '{' {
		return l.styleDim.Render(line)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		return l.styleDim.Render(line)
	}

	if s, ok := m["stream"].(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}
		if strings.HasPrefix(s, "Step ") || strings.HasPrefix(s, "Successfully") || strings.HasPrefix(s, "---") {
			return s
		}
		if strings.HasPrefix(s, "Running in ") || strings.HasPrefix(s, "Removed intermediate container ") {
			return l.styleDim.Render(s)
		}
		return l.styleDim.Render(s)
	}

	if ed, ok := m["errorDetail"].(map[string]any); ok {
		if msg, ok := ed["message"].(string); ok && msg != "" {
			return l.styleError.Render(msg)
		}
	}
	if msg, ok := m["error"].(string); ok && msg != "" {
		return l.styleError.Render(msg)
	}

	if aux, ok := m["aux"].(map[string]any); ok {
		if id, ok := aux["ID"].(string); ok && id != "" {
			return l.styleSuccess.Render("Built " + id)
		}
	}

	return l.styleDim.Render(line)
}
