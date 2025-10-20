package build

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ContextTarStreamer handles creating tar streams of build contexts
type ContextTarStreamer struct {
	WarnThresholdMB int
}

// NewContextTarStreamer creates a new context tar streamer
func NewContextTarStreamer(warnThresholdMB int) *ContextTarStreamer {
	return &ContextTarStreamer{
		WarnThresholdMB: warnThresholdMB,
	}
}

// CreateTarStream creates a tar stream of the build context
func (cts *ContextTarStreamer) CreateTarStream(contextPath string) (io.ReadCloser, error) {
	// Validate context path
	if !isDirectory(contextPath) {
		return nil, fmt.Errorf("context path is not a directory: %s", contextPath)
	}

	// Load .dockerignore patterns
	ignorePatterns, err := cts.loadDockerignore(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load .dockerignore: %w", err)
	}

	// Create pipe for streaming
	reader, writer := io.Pipe()

	// Start goroutine to write tar
	go func() {
		defer writer.Close()

		tw := tar.NewWriter(writer)
		defer tw.Close()

		err := cts.writeContextToTar(contextPath, ignorePatterns, tw)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
	}()

	return reader, nil
}

// loadDockerignore loads .dockerignore patterns from the context directory
func (cts *ContextTarStreamer) loadDockerignore(contextPath string) ([]string, error) {
	dockerignorePath := filepath.Join(contextPath, ".dockerignore")

	// Check if .dockerignore exists
	if _, err := os.Stat(dockerignorePath); os.IsNotExist(err) {
		return []string{}, nil // No .dockerignore file
	}

	file, err := os.Open(dockerignorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .dockerignore: %w", err)
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		patterns = append(patterns, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read .dockerignore: %w", err)
	}

	return patterns, nil
}

// writeContextToTar writes the build context to a tar writer
func (cts *ContextTarStreamer) writeContextToTar(contextPath string, ignorePatterns []string, tw *tar.Writer) error {
	var totalSize int64

	err := filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == contextPath {
			return nil
		}

		// Get relative path from context
		relPath, err := filepath.Rel(contextPath, path)
		if err != nil {
			return err
		}

		// Normalize path separators for cross-platform compatibility
		relPath = filepath.ToSlash(relPath)

		// Check if path should be ignored
		if cts.shouldIgnore(relPath, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Set the name in the tar header
		header.Name = relPath

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content for regular files
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			// Copy file content and track size
			written, err := io.Copy(tw, file)
			if err != nil {
				return err
			}

			totalSize += written

			// Check size threshold
			if cts.WarnThresholdMB > 0 && totalSize > int64(cts.WarnThresholdMB*1024*1024) {
				// Note: In a real implementation, you might want to emit a warning here
				// For now, we'll continue but this could be enhanced to emit warnings
			}
		}

		return nil
	})

	return err
}

// shouldIgnore checks if a path should be ignored based on .dockerignore patterns
func (cts *ContextTarStreamer) shouldIgnore(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if cts.matchesPattern(relPath, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a path matches a .dockerignore pattern
func (cts *ContextTarStreamer) matchesPattern(relPath, pattern string) bool {
	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		dirPattern := strings.TrimSuffix(pattern, "/")
		return strings.HasPrefix(relPath, dirPattern+"/") || relPath == dirPattern
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, relPath)
		return matched
	}

	// Handle exact matches
	if relPath == pattern {
		return true
	}

	// Handle prefix matches
	if strings.HasPrefix(relPath, pattern+"/") {
		return true
	}

	return false
}

// GetContextSize estimates the size of the build context
func (cts *ContextTarStreamer) GetContextSize(contextPath string) (int64, error) {
	ignorePatterns, err := cts.loadDockerignore(contextPath)
	if err != nil {
		return 0, err
	}

	var totalSize int64

	err = filepath.Walk(contextPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == contextPath {
			return nil
		}

		relPath, err := filepath.Rel(contextPath, path)
		if err != nil {
			return err
		}

		relPath = filepath.ToSlash(relPath)

		if cts.shouldIgnore(relPath, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.Mode().IsRegular() {
			totalSize += info.Size()
		}

		return nil
	})

	return totalSize, err
}

// ValidateContext validates that a build context is valid
func (cts *ContextTarStreamer) ValidateContext(contextPath string) error {
	// Check if context exists and is a directory
	if !isDirectory(contextPath) {
		return fmt.Errorf("context path is not a directory: %s", contextPath)
	}

	// Check if .dockerignore is readable (if it exists)
	dockerignorePath := filepath.Join(contextPath, ".dockerignore")
	if _, err := os.Stat(dockerignorePath); err == nil {
		file, err := os.Open(dockerignorePath)
		if err != nil {
			return fmt.Errorf("cannot read .dockerignore: %w", err)
		}
		file.Close()
	}

	// Check context size
	size, err := cts.GetContextSize(contextPath)
	if err != nil {
		return fmt.Errorf("failed to calculate context size: %w", err)
	}

	// Warn if context is too large
	if cts.WarnThresholdMB > 0 && size > int64(cts.WarnThresholdMB*1024*1024) {
		// In a real implementation, this would emit a warning
		// For now, we'll just continue
	}

	return nil
}

// Helper function to check if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Helper function to check if a path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

