package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadComposeFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a valid compose file
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Test reading the compose file
	content, err := ReadComposeFile(composeFile)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "version: '3.8'")
	assert.Contains(t, content, "services:")
	assert.Contains(t, content, "web:")
	assert.Contains(t, content, "db:")
}

func TestReadComposeFile_NotFound(t *testing.T) {
	// Test with non-existent file
	nonExistentFile := "/non/existent/docker-compose.yml"
	content, err := ReadComposeFile(nonExistentFile)

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "compose file '/non/existent/docker-compose.yml' not found")
}

func TestReadComposeFile_Empty(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an empty compose file
	composeFile := filepath.Join(tempDir, "empty-compose.yml")
	err := os.WriteFile(composeFile, []byte(""), 0644)
	require.NoError(t, err)

	// Test reading the empty compose file
	content, err := ReadComposeFile(composeFile)

	assert.Error(t, err)
	assert.Empty(t, content)
	assert.Contains(t, err.Error(), "compose file '"+composeFile+"' is empty")
}

func TestValidateComposeFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a valid compose file
	composeContent := `
version: '3.8'
services:
  web:
    build: .
    ports:
      - "3000:3000"
`
	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Test validating the compose file
	err = ValidateComposeFile(composeFile)
	assert.NoError(t, err)
}

func TestValidateComposeFile_NotFound(t *testing.T) {
	// Test with non-existent file
	nonExistentFile := "/non/existent/docker-compose.yml"
	err := ValidateComposeFile(nonExistentFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compose file '/non/existent/docker-compose.yml' not found")
}

func TestValidateComposeFile_Empty(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an empty compose file
	composeFile := filepath.Join(tempDir, "empty-compose.yml")
	err := os.WriteFile(composeFile, []byte(""), 0644)
	require.NoError(t, err)

	// Test validating the empty compose file
	err = ValidateComposeFile(composeFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compose file '"+composeFile+"' is empty")
}

func TestValidateComposeFile_Unreadable(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a compose file with restricted permissions (read-only for owner)
	composeFile := filepath.Join(tempDir, "restricted-compose.yml")
	composeContent := "version: '3.8'\nservices:\n  web:\n    build: ."
	err := os.WriteFile(composeFile, []byte(composeContent), 0400) // Read-only
	require.NoError(t, err)

	// Test validating the compose file (should still work as we can read it)
	err = ValidateComposeFile(composeFile)
	assert.NoError(t, err)
}

func TestReadComposeFile_WithComplexContent(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a complex compose file
	composeContent := `
version: '3.8'

services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        NODE_ENV: production
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - DATABASE_URL=postgres://user:pass@db:5432/myapp
    depends_on:
      - db
    volumes:
      - ./src:/app/src
    networks:
      - app-network

  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - app-network

  redis:
    image: redis:6-alpine
    networks:
      - app-network

volumes:
  postgres_data:

networks:
  app-network:
    driver: bridge
`
	composeFile := filepath.Join(tempDir, "complex-compose.yml")
	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Test reading the complex compose file
	content, err := ReadComposeFile(composeFile)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "version: '3.8'")
	assert.Contains(t, content, "services:")
	assert.Contains(t, content, "web:")
	assert.Contains(t, content, "db:")
	assert.Contains(t, content, "redis:")
	assert.Contains(t, content, "volumes:")
	assert.Contains(t, content, "networks:")
}

func TestReadComposeFile_WithWhitespace(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a compose file with only whitespace
	composeContent := "   \n\t  \n  "
	composeFile := filepath.Join(tempDir, "whitespace-compose.yml")
	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Test reading the whitespace-only compose file
	content, err := ReadComposeFile(composeFile)
	require.NoError(t, err)
	assert.Equal(t, composeContent, content)
}

func TestReadComposeFile_WithComments(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a compose file with comments
	composeContent := `
# This is a comment
version: '3.8'
services:
  web:
    build: .  # Another comment
    ports:
      - "3000:3000"
`
	composeFile := filepath.Join(tempDir, "commented-compose.yml")
	err := os.WriteFile(composeFile, []byte(composeContent), 0644)
	require.NoError(t, err)

	// Test reading the compose file with comments
	content, err := ReadComposeFile(composeFile)
	require.NoError(t, err)
	assert.Contains(t, content, "# This is a comment")
	assert.Contains(t, content, "# Another comment")
	assert.Contains(t, content, "version: '3.8'")
}
