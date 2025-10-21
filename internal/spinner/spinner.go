package spinner

import (
	"fmt"

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
	spinner        spinner.Model
	message        string
	successMessage string
	done           bool
	err            error
}

// NewSpinnerModel creates a new spinner model with a custom message
func NewSpinnerModel(message string) *SpinnerModel {
	return NewSpinnerModelWithSuccess(message, "")
}

// NewSpinnerModelWithSuccess creates a new spinner model with custom message and success message
func NewSpinnerModelWithSuccess(message, successMessage string) *SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &SpinnerModel{
		spinner:        s,
		message:        message,
		successMessage: successMessage,
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
		// Use custom success message if provided, otherwise use default
		successMsg := m.successMessage
		if successMsg == "" {
			successMsg = "✓ Operation completed"
		}
		return successStyle.Render(successMsg)
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
	return RunWithSpinnerAndSuccess(message, "", operation)
}

// RunWithSpinnerAndSuccess runs a function with a spinner display and custom success message
func RunWithSpinnerAndSuccess(message, successMessage string, operation func() error) error {
	// Create spinner model with custom success message
	model := NewSpinnerModelWithSuccess(message, successMessage)

	// Create tea program
	p := tea.NewProgram(model)

	// Channel to receive operation result
	resultChan := make(chan error, 1)
	doneChan := make(chan bool, 1)

	// Start the operation in a goroutine
	go func() {
		err := operation()
		resultChan <- err
	}()

	// Start the spinner in a goroutine
	go func() {
		if _, err := p.Run(); err != nil {
			// If spinner fails, just continue
		}
		doneChan <- true
	}()

	// Wait for operation to complete
	err := <-resultChan

	// Send completion message to spinner
	if err != nil {
		p.Send(spinnerErrorMsg{err: err})
	} else {
		p.Send(spinnerCompleteMsg{})
	}

	// Wait for spinner to finish displaying
	<-doneChan

	return err
}
