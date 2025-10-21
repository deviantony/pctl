package spinner

import (
	"errors"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
)

func TestNewSpinnerModel(t *testing.T) {
	message := "Testing spinner"
	model := NewSpinnerModel(message)

	assert.Equal(t, message, model.message)
	assert.Equal(t, "", model.successMessage)
	assert.False(t, model.done)
	assert.Nil(t, model.err)
}

func TestNewSpinnerModelWithSuccess(t *testing.T) {
	message := "Testing spinner"
	successMessage := "✓ Test completed"
	model := NewSpinnerModelWithSuccess(message, successMessage)

	assert.Equal(t, message, model.message)
	assert.Equal(t, successMessage, model.successMessage)
	assert.False(t, model.done)
	assert.Nil(t, model.err)
}

func TestSpinnerModel_View_Spinning(t *testing.T) {
	model := NewSpinnerModel("Testing spinner")

	// When not done, should show spinner and message
	view := model.View()
	assert.Contains(t, view, "Testing spinner")
	assert.Contains(t, view, "\n") // Should have newlines for spinner display
}

func TestSpinnerModel_View_Success_WithCustomMessage(t *testing.T) {
	model := NewSpinnerModelWithSuccess("Testing spinner", "✓ Custom success")
	model.done = true

	view := model.View()
	assert.Equal(t, "✓ Custom success", view)
	assert.NotContains(t, view, "\n") // Should not have extra newlines
}

func TestSpinnerModel_View_Success_WithDefaultMessage(t *testing.T) {
	model := NewSpinnerModel("Testing spinner")
	model.done = true

	view := model.View()
	assert.Equal(t, "✓ Operation completed", view)
	assert.NotContains(t, view, "\n") // Should not have extra newlines
}

func TestSpinnerModel_View_Error(t *testing.T) {
	model := NewSpinnerModel("Testing spinner")
	model.done = true
	model.err = errors.New("test error")

	view := model.View()
	assert.Contains(t, view, "✗ Operation failed")
	assert.Contains(t, view, "test error")
	assert.Contains(t, view, "\n") // Should have newlines for error display
}

func TestSpinnerModel_Update_SpinnerTick(t *testing.T) {
	model := NewSpinnerModel("Testing spinner")

	// Simulate a spinner tick
	updatedModel, cmd := model.Update(spinner.TickMsg{})

	// Should return the same model with a command
	assert.NotNil(t, cmd)

	// Cast back to SpinnerModel to check fields
	spinnerModel := updatedModel.(SpinnerModel)
	assert.False(t, spinnerModel.done)
}

func TestSpinnerModel_Update_Complete(t *testing.T) {
	model := NewSpinnerModel("Testing spinner")

	// Simulate completion
	updatedModel, cmd := model.Update(spinnerCompleteMsg{})

	// Cast back to SpinnerModel to check fields
	spinnerModel := updatedModel.(SpinnerModel)
	assert.True(t, spinnerModel.done)
	assert.NotNil(t, cmd)
}

func TestSpinnerModel_Update_Error(t *testing.T) {
	model := NewSpinnerModel("Testing spinner")
	testErr := errors.New("test error")

	// Simulate error
	updatedModel, cmd := model.Update(spinnerErrorMsg{err: testErr})

	// Cast back to SpinnerModel to check fields
	spinnerModel := updatedModel.(SpinnerModel)
	assert.True(t, spinnerModel.done)
	assert.Equal(t, testErr, spinnerModel.err)
	assert.NotNil(t, cmd)
}

func TestRunWithSpinner(t *testing.T) {
	// Test successful operation
	err := RunWithSpinner("Testing operation", func() error {
		return nil
	})

	assert.NoError(t, err)
}

func TestRunWithSpinner_Error(t *testing.T) {
	// Test operation that returns error
	testErr := errors.New("operation failed")
	err := RunWithSpinner("Testing operation", func() error {
		return testErr
	})

	assert.Equal(t, testErr, err)
}

func TestRunWithSpinnerAndSuccess(t *testing.T) {
	// Test successful operation with custom success message
	err := RunWithSpinnerAndSuccess("Testing operation", "✓ Custom success", func() error {
		return nil
	})

	assert.NoError(t, err)
}

func TestRunWithSpinnerAndSuccess_Error(t *testing.T) {
	// Test operation that returns error with custom success message
	testErr := errors.New("operation failed")
	err := RunWithSpinnerAndSuccess("Testing operation", "✓ Custom success", func() error {
		return testErr
	})

	assert.Equal(t, testErr, err)
}

func TestRunWithSpinnerAndSuccess_EmptySuccessMessage(t *testing.T) {
	// Test with empty success message (should use default)
	err := RunWithSpinnerAndSuccess("Testing operation", "", func() error {
		return nil
	})

	assert.NoError(t, err)
}
