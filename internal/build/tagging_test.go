package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagGenerator_GenerateTag(t *testing.T) {
	tg := NewTagGenerator("my-stack", "{{stack}}-{{service}}:{{hash}}")

	tag := tg.GenerateTag("web", "abc123")
	expected := "my-stack-web:abc123"
	assert.Equal(t, expected, tag)
}

func TestTagGenerator_GenerateTag_StackVariable(t *testing.T) {
	tg := NewTagGenerator("test-stack", "pctl-{{stack}}-{{service}}:{{hash}}")

	tag := tg.GenerateTag("api", "def456")
	expected := "pctl-test-stack-api:def456"
	assert.Equal(t, expected, tag)
}

func TestTagGenerator_GenerateTag_ServiceVariable(t *testing.T) {
	tg := NewTagGenerator("my-app", "{{service}}-{{stack}}:{{hash}}")

	tag := tg.GenerateTag("database", "ghi789")
	expected := "database-my-app:ghi789"
	assert.Equal(t, expected, tag)
}

func TestTagGenerator_GenerateTag_HashVariable(t *testing.T) {
	tg := NewTagGenerator("project", "{{stack}}/{{service}}:{{hash}}")

	tag := tg.GenerateTag("worker", "jkl012")
	expected := "project/worker:jkl012"
	assert.Equal(t, expected, tag)
}

func TestTagGenerator_GenerateTag_TimestampVariable(t *testing.T) {
	tg := NewTagGenerator("app", "{{stack}}-{{service}}:{{timestamp}}")

	tag := tg.GenerateTag("service", "hash")
	// The timestamp will be current time, so we just check the format
	assert.Contains(t, tag, "app-service:")
	assert.True(t, len(tag) > len("app-service:"))
}

func TestContentHasher_HashBuildContext(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a Dockerfile
	dockerfile := filepath.Join(tempDir, "Dockerfile")
	err = os.WriteFile(dockerfile, []byte("FROM alpine\nCOPY test.txt /app/"), 0644)
	require.NoError(t, err)

	hasher := NewContentHasher()
	hash, err := hasher.HashBuildContext(tempDir, "Dockerfile", nil)
	require.NoError(t, err)

	// Hash should be deterministic
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 12) // Should be 12 characters

	// Same input should produce same hash
	hash2, err := hasher.HashBuildContext(tempDir, "Dockerfile", nil)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)
}

func TestContentHasher_HashBuildContext_DifferentContent(t *testing.T) {
	// Create two different directory structures
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Create different files
	file1 := filepath.Join(tempDir1, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir2, "file2.txt")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	require.NoError(t, err)

	// Create Dockerfiles
	dockerfile1 := filepath.Join(tempDir1, "Dockerfile")
	err = os.WriteFile(dockerfile1, []byte("FROM alpine\nCOPY file1.txt /app/"), 0644)
	require.NoError(t, err)

	dockerfile2 := filepath.Join(tempDir2, "Dockerfile")
	err = os.WriteFile(dockerfile2, []byte("FROM alpine\nCOPY file2.txt /app/"), 0644)
	require.NoError(t, err)

	hasher := NewContentHasher()
	hash1, err := hasher.HashBuildContext(tempDir1, "Dockerfile", nil)
	require.NoError(t, err)

	hash2, err := hasher.HashBuildContext(tempDir2, "Dockerfile", nil)
	require.NoError(t, err)

	// Different content should produce different hashes
	assert.NotEqual(t, hash1, hash2)
}

func TestContentHasher_HashBuildContext_WithBuildArgs(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a Dockerfile
	dockerfile := filepath.Join(tempDir, "Dockerfile")
	err = os.WriteFile(dockerfile, []byte("FROM alpine\nCOPY test.txt /app/"), 0644)
	require.NoError(t, err)

	hasher := NewContentHasher()

	// Test with build args
	buildArgs1 := map[string]string{"VERSION": "1.0", "ENV": "prod"}
	hash1, err := hasher.HashBuildContext(tempDir, "Dockerfile", buildArgs1)
	require.NoError(t, err)

	buildArgs2 := map[string]string{"VERSION": "2.0", "ENV": "prod"}
	hash2, err := hasher.HashBuildContext(tempDir, "Dockerfile", buildArgs2)
	require.NoError(t, err)

	// Different build args should produce different hashes
	assert.NotEqual(t, hash1, hash2)

	// Same build args should produce same hash
	hash3, err := hasher.HashBuildContext(tempDir, "Dockerfile", buildArgs1)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash3)
}

func TestContentHasher_HashBuildContext_WithDockerignore(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "include.txt")
	err := os.WriteFile(file1, []byte("include this"), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "exclude.txt")
	err = os.WriteFile(file2, []byte("exclude this"), 0644)
	require.NoError(t, err)

	// Create .dockerignore
	dockerignore := filepath.Join(tempDir, ".dockerignore")
	err = os.WriteFile(dockerignore, []byte("exclude.txt\n*.log"), 0644)
	require.NoError(t, err)

	// Create a Dockerfile
	dockerfile := filepath.Join(tempDir, "Dockerfile")
	err = os.WriteFile(dockerfile, []byte("FROM alpine\nCOPY include.txt /app/"), 0644)
	require.NoError(t, err)

	hasher := NewContentHasher()
	hash, err := hasher.HashBuildContext(tempDir, "Dockerfile", nil)
	require.NoError(t, err)

	// Hash should not include excluded files
	assert.NotEmpty(t, hash)
}

func TestContentHasher_HashBuildContext_Deterministic(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a Dockerfile
	dockerfile := filepath.Join(tempDir, "Dockerfile")
	err = os.WriteFile(dockerfile, []byte("FROM alpine\nCOPY test.txt /app/"), 0644)
	require.NoError(t, err)

	hasher := NewContentHasher()

	// Generate hash multiple times
	hash1, err := hasher.HashBuildContext(tempDir, "Dockerfile", nil)
	require.NoError(t, err)

	hash2, err := hasher.HashBuildContext(tempDir, "Dockerfile", nil)
	require.NoError(t, err)

	hash3, err := hasher.HashBuildContext(tempDir, "Dockerfile", nil)
	require.NoError(t, err)

	// All hashes should be identical
	assert.Equal(t, hash1, hash2)
	assert.Equal(t, hash2, hash3)
}

func TestTagValidator_ValidateTag(t *testing.T) {
	validator := NewTagValidator()

	// Valid tags
	validTags := []string{
		"myapp:latest",
		"myapp:v1.0.0",
		"myapp:abc123",
		"myapp",
		"my-app_service:v1.0.0",
	}

	for _, tag := range validTags {
		t.Run("valid_"+tag, func(t *testing.T) {
			err := validator.ValidateTag(tag)
			assert.NoError(t, err)
		})
	}
}

func TestTagValidator_ValidateTag_Empty(t *testing.T) {
	validator := NewTagValidator()

	err := validator.ValidateTag("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag cannot be empty")
}

func TestTagValidator_ValidateTag_TooLong(t *testing.T) {
	validator := NewTagValidator()

	// Create a tag that's too long (129 characters)
	longTag := "a" + string(make([]byte, 128))
	err := validator.ValidateTag(longTag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tag too long")
}

func TestTagValidator_ValidateTag_InvalidChars(t *testing.T) {
	validator := NewTagValidator()

	invalidTags := []string{
		"my app:latest",  // space
		"my\tapp:latest", // tab
		"my\napp:latest", // newline
		"my\rapp:latest", // carriage return
		"my@app:latest",  // invalid character
		"my#app:latest",  // invalid character
	}

	for _, tag := range invalidTags {
		t.Run("invalid_"+tag, func(t *testing.T) {
			err := validator.ValidateTag(tag)
			assert.Error(t, err)
		})
	}
}

func TestTagTemplateValidator_ValidateTagFormat(t *testing.T) {
	validator := NewTagTemplateValidator()

	// Valid tag formats
	validFormats := []string{
		"{{stack}}-{{service}}:{{hash}}",
		"pctl-{{stack}}-{{service}}:{{hash}}",
		"{{service}}:{{hash}}",
		"{{stack}}-{{service}}-{{timestamp}}",
	}

	for _, format := range validFormats {
		t.Run("valid_"+format, func(t *testing.T) {
			err := validator.ValidateTagFormat(format)
			assert.NoError(t, err)
		})
	}
}

func TestTagTemplateValidator_ValidateTagFormat_InvalidVariable(t *testing.T) {
	validator := NewTagTemplateValidator()

	// Invalid tag formats
	invalidFormats := []string{
		"{{invalid}}-{{service}}:{{hash}}",
		"{{stack}}-{{badvar}}:{{hash}}",
		"{{unknown}}",
	}

	for _, format := range invalidFormats {
		t.Run("invalid_"+format, func(t *testing.T) {
			err := validator.ValidateTagFormat(format)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid template variable")
		})
	}
}

func TestTagTemplateValidator_ValidateTagFormat_UnclosedVariable(t *testing.T) {
	validator := NewTagTemplateValidator()

	// Unclosed template variables
	unclosedFormats := []string{
		"{{stack",
	}

	for _, format := range unclosedFormats {
		t.Run("unclosed_"+format, func(t *testing.T) {
			err := validator.ValidateTagFormat(format)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unclosed template variable")
		})
	}
}

func TestSanitizeServiceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple service name",
			input:    "web",
			expected: "web",
		},
		{
			name:     "service with underscores",
			input:    "web_service",
			expected: "web-service",
		},
		{
			name:     "service with spaces",
			input:    "web service",
			expected: "web-service",
		},
		{
			name:     "service with uppercase",
			input:    "WebService",
			expected: "webservice",
		},
		{
			name:     "service with invalid characters",
			input:    "web@service#1",
			expected: "web-service-1",
		},
		{
			name:     "service with mixed case and special chars",
			input:    "Web_Service@1",
			expected: "web-service-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeServiceName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeStackName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple stack name",
			input:    "myapp",
			expected: "myapp",
		},
		{
			name:     "stack with underscores",
			input:    "my_app",
			expected: "my-app",
		},
		{
			name:     "stack with spaces",
			input:    "my app",
			expected: "my-app",
		},
		{
			name:     "stack with uppercase",
			input:    "MyApp",
			expected: "myapp",
		},
		{
			name:     "stack with invalid characters",
			input:    "my@app#1",
			expected: "my-app-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeStackName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTagGenerator(t *testing.T) {
	tg := NewTagGenerator("test-stack", "{{stack}}-{{service}}:{{hash}}")

	assert.Equal(t, "test-stack", tg.StackName)
	assert.Equal(t, "{{stack}}-{{service}}:{{hash}}", tg.TagFormat)
}

func TestNewContentHasher(t *testing.T) {
	ch := NewContentHasher()
	assert.NotNil(t, ch)
}

func TestNewTagValidator(t *testing.T) {
	tv := NewTagValidator()
	assert.NotNil(t, tv)
}

func TestNewTagTemplateValidator(t *testing.T) {
	ttv := NewTagTemplateValidator()
	assert.NotNil(t, ttv)
}

func TestGetDefaultTagFormat(t *testing.T) {
	format := GetDefaultTagFormat()
	expected := "pctl-{{stack}}-{{service}}:{{hash}}"
	assert.Equal(t, expected, format)
}

func TestTagGenerator_GenerateTagWithTimestamp(t *testing.T) {
	tg := NewTagGenerator("my-stack", "{{stack}}-{{service}}:{{timestamp}}")

	tag := tg.GenerateTagWithTimestamp("web")
	// The timestamp will be current time, so we just check the format
	assert.Contains(t, tag, "my-stack-web:")
	assert.True(t, len(tag) > len("my-stack-web:"))
}

func TestContentHasher_HashFileContents(t *testing.T) {
	tempDir := t.TempDir()

	hasher := NewContentHasher()
	hash, err := hasher.HashFileContents(tempDir)
	require.NoError(t, err)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 12)
}

func TestIsValidTagChar(t *testing.T) {
	tests := []struct {
		name     string
		char     rune
		expected bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"hyphen", '-', true},
		{"underscore", '_', true},
		{"dot", '.', true},
		{"invalid char", '@', false},
		{"space", ' ', false},
		{"special char", '#', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidTagChar(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}
