package build

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TagGenerator handles generation of deterministic image tags
type TagGenerator struct {
	StackName string
	TagFormat string
}

// NewTagGenerator creates a new tag generator
func NewTagGenerator(stackName, tagFormat string) *TagGenerator {
	return &TagGenerator{
		StackName: stackName,
		TagFormat: tagFormat,
	}
}

// GenerateTag generates a tag for a service using the configured format
func (tg *TagGenerator) GenerateTag(serviceName, contentHash string) string {
	tag := tg.TagFormat

	// Replace template variables
	tag = strings.ReplaceAll(tag, "{{stack}}", tg.StackName)
	tag = strings.ReplaceAll(tag, "{{service}}", serviceName)
	tag = strings.ReplaceAll(tag, "{{hash}}", contentHash)
	tag = strings.ReplaceAll(tag, "{{timestamp}}", fmt.Sprintf("%d", time.Now().Unix()))

	return tag
}

// GenerateTagWithTimestamp generates a tag with timestamp (for force builds)
func (tg *TagGenerator) GenerateTagWithTimestamp(serviceName string) string {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	return tg.GenerateTag(serviceName, timestamp)
}

// ContentHasher handles generation of content hashes for build contexts
type ContentHasher struct{}

// NewContentHasher creates a new content hasher
func NewContentHasher() *ContentHasher {
	return &ContentHasher{}
}

// HashBuildContext generates a content hash for a build context
func (ch *ContentHasher) HashBuildContext(contextPath string, dockerfilePath string, buildArgs map[string]string) (string, error) {
	hasher := sha256.New()

	// Normalize and ensure absolute context path for consistent walking
	absContext, err := filepath.Abs(contextPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve context path: %w", err)
	}

	// Load .dockerignore patterns
	streamer := NewContextTarStreamer(0)
	ignorePatterns, err := streamer.loadDockerignore(absContext)
	if err != nil {
		return "", fmt.Errorf("failed to load .dockerignore: %w", err)
	}

	// Include Dockerfile path and contents (relative to context)
	dockerfileRel := dockerfilePath
	if dockerfileRel == "" {
		dockerfileRel = "Dockerfile"
	}
	hasher.Write([]byte("DOCKERFILE_PATH:\n"))
	hasher.Write([]byte(dockerfileRel))
	dockerfileFull := filepath.Join(absContext, dockerfileRel)
	if f, err := os.Open(dockerfileFull); err == nil {
		defer f.Close()
		hasher.Write([]byte("\nDOCKERFILE_CONTENTS:\n"))
		if _, copyErr := io.Copy(hasher, f); copyErr != nil {
			return "", fmt.Errorf("failed to read Dockerfile for hashing: %w", copyErr)
		}
	}

	// Include build args deterministically (sorted by key)
	if len(buildArgs) > 0 {
		hasher.Write([]byte("\nBUILD_ARGS:\n"))
		keys := make([]string, 0, len(buildArgs))
		for k := range buildArgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			hasher.Write([]byte(k))
			hasher.Write([]byte("="))
			hasher.Write([]byte(buildArgs[k]))
			hasher.Write([]byte("\n"))
		}
	}

	// Walk the context and hash file paths + contents, respecting .dockerignore
	var files []string
	err = filepath.Walk(absContext, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == absContext {
			return nil
		}
		// Compute slash-normalized relative path
		rel, err := filepath.Rel(absContext, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		// Respect .dockerignore
		if streamer.shouldIgnore(rel, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Mode().IsRegular() {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk context for hashing: %w", err)
	}

	// Sort files for deterministic order
	sort.Strings(files)

	// Hash file paths and contents
	for _, rel := range files {
		hasher.Write([]byte("FILE:\n"))
		hasher.Write([]byte(rel))
		hasher.Write([]byte("\n"))
		full := filepath.Join(absContext, rel)
		f, err := os.Open(full)
		if err != nil {
			return "", fmt.Errorf("failed to open file for hashing: %w", err)
		}
		if _, copyErr := io.Copy(hasher, f); copyErr != nil {
			f.Close()
			return "", fmt.Errorf("failed to read file for hashing: %w", copyErr)
		}
		f.Close()
	}

	sum := hasher.Sum(nil)
	return fmt.Sprintf("%x", sum)[:12], nil
}

// HashFileContents generates a hash of file contents in a directory
// This is a placeholder for the full implementation that would:
// 1. Walk the directory tree
// 2. Apply .dockerignore patterns
// 3. Hash file contents and metadata
// 4. Return a deterministic hash
func (ch *ContentHasher) HashFileContents(contextPath string) (string, error) {
	// For now, return a placeholder hash
	// In a real implementation, this would:
	// - Walk the directory
	// - Apply .dockerignore rules
	// - Hash file contents and metadata
	// - Return deterministic hash

	hasher := sha256.New()
	hasher.Write([]byte(contextPath))
	hasher.Write([]byte(fmt.Sprintf("%d", time.Now().Unix()))) // Placeholder

	hash := hasher.Sum(nil)
	return fmt.Sprintf("%x", hash)[:12], nil
}

// TagValidator validates image tag formats
type TagValidator struct{}

// NewTagValidator creates a new tag validator
func NewTagValidator() *TagValidator {
	return &TagValidator{}
}

// ValidateTag validates that a tag follows Docker naming conventions
func (tv *TagValidator) ValidateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	// Check length
	if len(tag) > 128 {
		return fmt.Errorf("tag too long: %d characters (max 128)", len(tag))
	}

	// Check for invalid characters
	invalidChars := []string{" ", "\t", "\n", "\r"}
	for _, char := range invalidChars {
		if strings.Contains(tag, char) {
			return fmt.Errorf("tag contains invalid character: '%s'", char)
		}
	}

	// Check for valid format (simplified)
	parts := strings.Split(tag, ":")
	if len(parts) > 2 {
		return fmt.Errorf("tag has too many parts separated by ':' (max 2)")
	}

	// Validate each part
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("tag part %d is empty", i+1)
		}

		// Check for valid characters (alphanumeric, hyphens, underscores, dots)
		for _, char := range part {
			if !isValidTagChar(char) {
				return fmt.Errorf("tag part %d contains invalid character: '%c'", i+1, char)
			}
		}
	}

	return nil
}

// isValidTagChar checks if a character is valid in a Docker tag
func isValidTagChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '_' || char == '.'
}

// TagTemplateValidator validates tag format templates
type TagTemplateValidator struct{}

// NewTagTemplateValidator creates a new tag template validator
func NewTagTemplateValidator() *TagTemplateValidator {
	return &TagTemplateValidator{}
}

// ValidateTagFormat validates a tag format template
func (ttv *TagTemplateValidator) ValidateTagFormat(tagFormat string) error {
	if tagFormat == "" {
		return fmt.Errorf("tag format cannot be empty")
	}

	// Check for valid template variables
	validVars := []string{"{{stack}}", "{{service}}", "{{hash}}", "{{timestamp}}"}

	// Find all template variables
	var foundVars []string
	remaining := tagFormat
	for {
		start := strings.Index(remaining, "{{")
		if start == -1 {
			break
		}

		end := strings.Index(remaining[start:], "}}")
		if end == -1 {
			return fmt.Errorf("unclosed template variable in tag format")
		}

		varName := remaining[start : start+end+2]
		foundVars = append(foundVars, varName)
		remaining = remaining[start+end+2:]
	}

	// Validate each found variable
	for _, varName := range foundVars {
		valid := false
		for _, validVar := range validVars {
			if varName == validVar {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid template variable: %s (valid variables: %s)",
				varName, strings.Join(validVars, ", "))
		}
	}

	// Test the template with sample values
	testTag := tagFormat
	testTag = strings.ReplaceAll(testTag, "{{stack}}", "test-stack")
	testTag = strings.ReplaceAll(testTag, "{{service}}", "test-service")
	testTag = strings.ReplaceAll(testTag, "{{hash}}", "abc123")
	testTag = strings.ReplaceAll(testTag, "{{timestamp}}", "1234567890")

	// Validate the resulting tag
	validator := NewTagValidator()
	if err := validator.ValidateTag(testTag); err != nil {
		return fmt.Errorf("tag format produces invalid tag: %w", err)
	}

	return nil
}

// GetDefaultTagFormat returns the default tag format
func GetDefaultTagFormat() string {
	return "pctl-{{stack}}-{{service}}:{{hash}}"
}

// SanitizeServiceName sanitizes a service name for use in tags
func SanitizeServiceName(serviceName string) string {
	// Replace invalid characters with hyphens
	sanitized := strings.ReplaceAll(serviceName, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ToLower(sanitized)

	// Remove any remaining invalid characters
	var result strings.Builder
	for _, char := range sanitized {
		if isValidTagChar(char) {
			result.WriteRune(char)
		} else {
			result.WriteRune('-')
		}
	}

	return result.String()
}

// SanitizeStackName sanitizes a stack name for use in tags
func SanitizeStackName(stackName string) string {
	return SanitizeServiceName(stackName)
}
