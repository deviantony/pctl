package compose

import (
	"fmt"
	"os"
)

// ReadComposeFile reads and validates a Docker Compose file
func ReadComposeFile(path string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("compose file '%s' not found", path)
	}

	// Read file contents
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read compose file '%s': %w", path, err)
	}

	// Basic validation - check if file is not empty
	if len(content) == 0 {
		return "", fmt.Errorf("compose file '%s' is empty", path)
	}

	return string(content), nil
}

// ValidateComposeFile checks if a compose file exists and is readable
func ValidateComposeFile(path string) error {
	_, err := ReadComposeFile(path)
	return err
}
