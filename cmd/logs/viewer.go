package logs

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsViewer represents the TUI model for viewing logs
type LogsViewer struct {
	viewport    viewport.Model
	containers  []ContainerLogs
	currentIdx  int
	ready       bool
	width       int
	height      int
	headerStyle lipgloss.Style
	logStyle    lipgloss.Style
	helpStyle   lipgloss.Style
}

// ContainerLogs holds logs for a single container
type ContainerLogs struct {
	Name string
	Logs string
}

// NewLogsViewer creates a new logs viewer
func NewLogsViewer(containers []ContainerLogs) *LogsViewer {
	return &LogsViewer{
		containers: containers,
		currentIdx: 0,
		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
		logStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")),
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true),
	}
}

// Init implements the tea.Model interface
func (m LogsViewer) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (m LogsViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate viewport height more conservatively
		// Reserve space for header (3 lines), help (2 lines), and padding (3 lines)
		reservedHeight := 8
		viewportHeight := msg.Height - reservedHeight

		if viewportHeight < 10 {
			viewportHeight = 10 // Minimum viewport height
		}

		m.viewport = viewport.New(msg.Width, viewportHeight)
		m.viewport.SetContent(m.getCurrentContent())
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		case "pageup":
			m.viewport.PageUp()
		case "pagedown":
			m.viewport.PageDown()
		case "n", "right":
			if m.currentIdx < len(m.containers)-1 {
				m.currentIdx++
				m.viewport.SetContent(m.getCurrentContent())
				m.viewport.GotoTop()
			}
		case "p", "left":
			if m.currentIdx > 0 {
				m.currentIdx--
				m.viewport.SetContent(m.getCurrentContent())
				m.viewport.GotoTop()
			}
		}
	}

	// Update viewport
	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements the tea.Model interface
func (m LogsViewer) View() string {
	if !m.ready {
		return "Loading..."
	}

	var content strings.Builder

	// Header
	header := m.headerStyle.Render(fmt.Sprintf("Container: %s (%d/%d)",
		m.containers[m.currentIdx].Name,
		m.currentIdx+1,
		len(m.containers)))
	content.WriteString(header)
	content.WriteString("\n\n")

	// Viewport content
	content.WriteString(m.viewport.View())
	content.WriteString("\n\n")

	// Help text
	help := m.helpStyle.Render("↑/↓: scroll • n/p: next/prev container • g/G: top/bottom • q: quit")
	content.WriteString(help)

	return content.String()
}

// getCurrentContent returns the formatted content for the current container
func (m LogsViewer) getCurrentContent() string {
	if len(m.containers) == 0 {
		return "No logs available"
	}

	container := m.containers[m.currentIdx]
	if container.Logs == "" {
		return "(no logs available)"
	}

	// Split logs into lines and apply styling
	lines := strings.Split(strings.TrimSpace(container.Logs), "\n")
	var styledLines []string

	for _, line := range lines {
		if line != "" {
			// Clean up Docker log format (remove stream prefixes)
			cleanLine := cleanDockerLogLine(line)
			styledLines = append(styledLines, m.logStyle.Render(cleanLine))
		}
	}

	return strings.Join(styledLines, "\n")
}

// cleanDockerLogLine removes Docker's log format prefixes
func cleanDockerLogLine(line string) string {
	// Docker logs come with a prefix like: [8 bytes of stream info][timestamp] actual log
	// We need to skip the first 8 bytes and find the timestamp
	if len(line) < 8 {
		return line
	}

	// Skip the first 8 bytes (stream info) and look for timestamp
	content := line[8:]

	// Look for timestamp pattern (ISO 8601 format)
	// Timestamp is usually at the beginning after the stream info
	if len(content) > 26 && content[0] == '2' && content[4] == '-' && content[7] == '-' {
		// Found timestamp, return the content as-is
		return content
	}

	// If no timestamp found, return the original line
	return line
}

// RunViewer starts the interactive logs viewer
func RunViewer(containers []ContainerLogs) error {
	// Check if we're in an interactive terminal
	if !isInteractive() {
		return runNonInteractiveViewer(containers)
	}

	model := NewLogsViewer(containers)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run logs viewer: %w", err)
	}

	return nil
}

// isInteractive checks if we're running in an interactive terminal
func isInteractive() bool {
	// Simple check - if we can't open /dev/tty, we're probably not interactive
	_, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	return err == nil
}

// runNonInteractiveViewer displays logs in a simple format for non-interactive environments
func runNonInteractiveViewer(containers []ContainerLogs) error {
	for i, container := range containers {
		if i > 0 {
			fmt.Println()
		}

		fmt.Println(headerStyle.Render(fmt.Sprintf("=== %s ===", container.Name)))

		if container.Logs == "" {
			fmt.Println("(no logs available)")
		} else {
			lines := strings.Split(strings.TrimSpace(container.Logs), "\n")
			for _, line := range lines {
				if line != "" {
					cleanLine := cleanDockerLogLine(line)
					fmt.Println(logStyle.Render(cleanLine))
				}
			}
		}
	}

	return nil
}
