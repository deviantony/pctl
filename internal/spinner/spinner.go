package spinner

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/deviantony/pctl/internal/errors"
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
			errors.FormatError(m.err))
	}

	if m.done {
		return fmt.Sprintf("\n%s\n", successStyle.Render("✓ Operation completed successfully!"))
	}

	return fmt.Sprintf("\n%s %s\n",
		m.spinner.View(),
		infoStyle.Render(m.message))
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
