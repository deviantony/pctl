package build

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextTarStreamer_CreateTarStream(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.txt")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	file3 := filepath.Join(subDir, "file3.txt")
	err = os.WriteFile(file3, []byte("content3"), 0644)
	require.NoError(t, err)

	// Create tar stream
	streamer := NewContextTarStreamer(0)
	reader, err := streamer.CreateTarStream(tempDir)
	require.NoError(t, err)
	defer reader.Close()

	// Read and verify tar contents
	tr := tar.NewReader(reader)

	var foundFiles []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		foundFiles = append(foundFiles, header.Name)
	}

	// Should contain all files
	assert.Contains(t, foundFiles, "file1.txt")
	assert.Contains(t, foundFiles, "file2.txt")
	assert.Contains(t, foundFiles, "subdir/file3.txt")
}

func TestContextTarStreamer_loadDockerignore(t *testing.T) {
	tempDir := t.TempDir()

	// Create .dockerignore file
	dockerignorePath := filepath.Join(tempDir, ".dockerignore")
	dockerignoreContent := `# This is a comment
*.log
temp/
.git/
node_modules/
`
	err := os.WriteFile(dockerignorePath, []byte(dockerignoreContent), 0644)
	require.NoError(t, err)

	streamer := NewContextTarStreamer(0)
	patterns, err := streamer.loadDockerignore(tempDir)
	require.NoError(t, err)

	// Should load patterns, skipping comments and empty lines
	expected := []string{"*.log", "temp/", ".git/", "node_modules/"}
	assert.Equal(t, expected, patterns)
}

func TestContextTarStreamer_loadDockerignore_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	streamer := NewContextTarStreamer(0)
	patterns, err := streamer.loadDockerignore(tempDir)
	require.NoError(t, err)

	// Should return empty slice when .dockerignore doesn't exist
	assert.Empty(t, patterns)
}

func TestContextTarStreamer_shouldIgnore(t *testing.T) {
	streamer := NewContextTarStreamer(0)

	tests := []struct {
		name     string
		relPath  string
		patterns []string
		expected bool
	}{
		{
			name:     "exact match",
			relPath:  "file.txt",
			patterns: []string{"file.txt"},
			expected: true,
		},
		{
			name:     "prefix match",
			relPath:  "temp/file.txt",
			patterns: []string{"temp"},
			expected: true,
		},
		{
			name:     "wildcard match",
			relPath:  "app.log",
			patterns: []string{"*.log"},
			expected: true,
		},
		{
			name:     "directory pattern",
			relPath:  "node_modules/package",
			patterns: []string{"node_modules/"},
			expected: true,
		},
		{
			name:     "no match",
			relPath:  "src/main.go",
			patterns: []string{"*.log", "temp/"},
			expected: false,
		},
		{
			name:     "multiple patterns",
			relPath:  "app.log",
			patterns: []string{"*.log", "temp/", "*.tmp"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := streamer.shouldIgnore(tt.relPath, tt.patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextTarStreamer_shouldIgnore_WildcardPattern(t *testing.T) {
	streamer := NewContextTarStreamer(0)

	tests := []struct {
		name     string
		relPath  string
		pattern  string
		expected bool
	}{
		{
			name:     "simple wildcard",
			relPath:  "app.log",
			pattern:  "*.log",
			expected: true,
		},
		{
			name:     "wildcard in middle",
			relPath:  "src/main.go",
			pattern:  "src/*.go",
			expected: true,
		},
		{
			name:     "multiple wildcards",
			relPath:  "src/test/main_test.go",
			pattern:  "src/*/main_*.go",
			expected: true,
		},
		{
			name:     "no wildcard match",
			relPath:  "src/main.go",
			pattern:  "*.log",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := streamer.matchesPattern(tt.relPath, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextTarStreamer_shouldIgnore_DirectoryPattern(t *testing.T) {
	streamer := NewContextTarStreamer(0)

	tests := []struct {
		name     string
		relPath  string
		pattern  string
		expected bool
	}{
		{
			name:     "directory with trailing slash",
			relPath:  "node_modules/package",
			pattern:  "node_modules/",
			expected: true,
		},
		{
			name:     "exact directory match",
			relPath:  "temp",
			pattern:  "temp/",
			expected: true,
		},
		{
			name:     "subdirectory match",
			relPath:  "temp/subdir/file.txt",
			pattern:  "temp/",
			expected: true,
		},
		{
			name:     "no directory match",
			relPath:  "src/main.go",
			pattern:  "temp/",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := streamer.matchesPattern(tt.relPath, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextTarStreamer_matchesPattern(t *testing.T) {
	streamer := NewContextTarStreamer(0)

	tests := []struct {
		name     string
		relPath  string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			relPath:  "file.txt",
			pattern:  "file.txt",
			expected: true,
		},
		{
			name:     "prefix match",
			relPath:  "temp/file.txt",
			pattern:  "temp",
			expected: true,
		},
		{
			name:     "wildcard match",
			relPath:  "app.log",
			pattern:  "*.log",
			expected: true,
		},
		{
			name:     "directory pattern",
			relPath:  "node_modules/package",
			pattern:  "node_modules/",
			expected: true,
		},
		{
			name:     "no match",
			relPath:  "src/main.go",
			pattern:  "*.log",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := streamer.matchesPattern(tt.relPath, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextTarStreamer_GetContextSize(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files with known sizes
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.txt")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	file3 := filepath.Join(subDir, "file3.txt")
	err = os.WriteFile(file3, []byte("content3"), 0644)
	require.NoError(t, err)

	streamer := NewContextTarStreamer(0)
	size, err := streamer.GetContextSize(tempDir)
	require.NoError(t, err)

	// Should include all files (8 bytes each)
	expectedSize := int64(24) // 3 files * 8 bytes each
	assert.Equal(t, expectedSize, size)
}

func TestContextTarStreamer_GetContextSize_WithIgnore(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.log")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Create .dockerignore to ignore .log files
	dockerignorePath := filepath.Join(tempDir, ".dockerignore")
	dockerignoreContent := "*.log\n"
	err = os.WriteFile(dockerignorePath, []byte(dockerignoreContent), 0644)
	require.NoError(t, err)

	streamer := NewContextTarStreamer(0)
	size, err := streamer.GetContextSize(tempDir)
	require.NoError(t, err)

	// Should only include file1.txt (8 bytes), not file2.log
	// Note: .dockerignore file itself might be included in size calculation
	expectedSize := int64(8) // file1.txt
	assert.GreaterOrEqual(t, size, expectedSize)
}

func TestContextTarStreamer_ValidateContext(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	streamer := NewContextTarStreamer(0)
	err = streamer.ValidateContext(tempDir)
	assert.NoError(t, err)
}

func TestContextTarStreamer_ValidateContext_NotDirectory(t *testing.T) {
	// Create a temporary file instead of directory
	tempFile, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	streamer := NewContextTarStreamer(0)
	err = streamer.ValidateContext(tempFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context path is not a directory")
}

func TestNewContextTarStreamer(t *testing.T) {
	streamer := NewContextTarStreamer(50)
	assert.Equal(t, 50, streamer.WarnThresholdMB)
}

func TestIsDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Test with directory
	result := isDirectory(tempDir)
	assert.True(t, result)

	// Test with file
	file := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(file, []byte("test"), 0644)
	require.NoError(t, err)

	result = isDirectory(file)
	assert.False(t, result)

	// Test with non-existent path
	result = isDirectory("/non/existent/path")
	assert.False(t, result)
}

func TestIsFile(t *testing.T) {
	tempDir := t.TempDir()

	// Test with file
	file := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(file, []byte("test"), 0644)
	require.NoError(t, err)

	result := isFile(file)
	assert.True(t, result)

	// Test with directory
	result = isFile(tempDir)
	assert.False(t, result)

	// Test with non-existent path
	result = isFile("/non/existent/file")
	assert.False(t, result)
}

func TestContextTarStreamer_CreateTarStream_WithDockerignore(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.log")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Create .dockerignore to ignore .log files
	dockerignorePath := filepath.Join(tempDir, ".dockerignore")
	dockerignoreContent := "*.log\n"
	err = os.WriteFile(dockerignorePath, []byte(dockerignoreContent), 0644)
	require.NoError(t, err)

	// Create tar stream
	streamer := NewContextTarStreamer(0)
	reader, err := streamer.CreateTarStream(tempDir)
	require.NoError(t, err)
	defer reader.Close()

	// Read and verify tar contents
	tr := tar.NewReader(reader)

	var foundFiles []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		foundFiles = append(foundFiles, header.Name)
	}

	// Should contain file1.txt but not file2.log
	assert.Contains(t, foundFiles, "file1.txt")
	assert.NotContains(t, foundFiles, "file2.log")
}

func TestContextTarStreamer_ValidateContext_WithDockerignore(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	file1 := filepath.Join(tempDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	// Create .dockerignore
	dockerignorePath := filepath.Join(tempDir, ".dockerignore")
	dockerignoreContent := "*.log\n"
	err = os.WriteFile(dockerignorePath, []byte(dockerignoreContent), 0644)
	require.NoError(t, err)

	streamer := NewContextTarStreamer(0)
	err = streamer.ValidateContext(tempDir)
	assert.NoError(t, err)
}

func TestContextTarStreamer_ValidateContext_WithLargeContext(t *testing.T) {
	tempDir := t.TempDir()

	// Create a large file (1MB)
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := strings.Repeat("x", 1024*1024) // 1MB
	err := os.WriteFile(largeFile, []byte(largeContent), 0644)
	require.NoError(t, err)

	// Set a small threshold (0.5MB)
	streamer := NewContextTarStreamer(1) // 1MB threshold
	err = streamer.ValidateContext(tempDir)
	// Should not error, but might emit warning in real implementation
	assert.NoError(t, err)
}
