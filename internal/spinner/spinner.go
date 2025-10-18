package spinner

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// SpinnerModel represents a spinner for API operations
type SpinnerModel struct {
	spinner spinner.Model
	message string
	done    bool
	err     error
}

// NewSpinnerModel creates a new spinner model with a custom message
func NewSpinnerModel(message string) *SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &SpinnerModel{
		spinner: s,
		message: message,
	}
}

// Init implements the tea.Model interface
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements the tea.Model interface
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.done {
			return m, tea.Quit
		}
		return m, cmd

	case spinnerCompleteMsg:
		m.done = true
		return m, tea.Quit

	case spinnerErrorMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

// View implements the tea.Model interface
func (m SpinnerModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n%s\n\n%s\n",
			errorStyle.Render("✗ Operation failed"),
			m.GetFriendlyErrorMessage(m.err))
	}

	if m.done {
		return fmt.Sprintf("\n%s\n", successStyle.Render("✓ Operation completed successfully!"))
	}

	return fmt.Sprintf("\n%s %s\n",
		m.spinner.View(),
		infoStyle.Render(m.message))
}

// GetFriendlyErrorMessage converts technical errors to user-friendly messages
func (m SpinnerModel) GetFriendlyErrorMessage(err error) string {
	errStr := err.Error()

	if containsAny(errStr, []string{"context deadline exceeded", "timeout"}) {
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s",
			warningStyle.Render("Network connection timeout"),
			infoStyle.Render("This usually means:"),
			infoStyle.Render("• Your internet connection is unstable"),
			infoStyle.Render("• The Portainer server is slow to respond"),
			infoStyle.Render("• The server might be temporarily unavailable"),
			infoStyle.Render("\nPlease check your connection and try again."))
	}

	if containsAny(errStr, []string{"connection refused"}) {
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s",
			warningStyle.Render("Connection refused"),
			infoStyle.Render("This usually means:"),
			infoStyle.Render("• The Portainer URL is incorrect"),
			infoStyle.Render("• The Portainer server is not running"),
			infoStyle.Render("• There's a firewall blocking the connection"),
			infoStyle.Render("\nPlease verify your Portainer URL and try again."))
	}

	if containsAny(errStr, []string{"certificate", "TLS"}) {
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s",
			warningStyle.Render("SSL/TLS certificate error"),
			infoStyle.Render("This usually means:"),
			infoStyle.Render("• The SSL certificate is invalid or expired"),
			infoStyle.Render("• You're using a self-signed certificate"),
			infoStyle.Render("• There's a certificate authority issue"),
			infoStyle.Render("\nYou can try again or contact your administrator."))
	}

	// Generic error message
	return fmt.Sprintf("%s\n\n%s\n%s",
		warningStyle.Render("Operation failed"),
		infoStyle.Render("Error details:"),
		errorStyle.Render(errStr))
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// Message types for spinner updates
type spinnerCompleteMsg struct{}
type spinnerErrorMsg struct{ err error }

// RunWithSpinner runs a function with a spinner display
func RunWithSpinner(message string, operation func() error) error {
	// Create and run spinner
	spinnerModel := NewSpinnerModel(message)
	p := tea.NewProgram(spinnerModel, tea.WithOutput(os.Stderr))

	// Start the operation in a goroutine
	var operationErr error
	done := make(chan bool)

	go func() {
		operationErr = operation()
		done <- true
	}()

	// Run spinner until operation completes
	go func() {
		<-done
		spinnerModel.done = true
		p.Quit()
	}()

	// Run the spinner
	if _, err := p.Run(); err != nil {
		// Spinner display error - not critical, continue
	}

	return operationErr
}
