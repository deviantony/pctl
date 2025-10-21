package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatError_Timeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: "Network connection timeout",
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: "Network connection timeout",
		},
		{
			name:     "timeout in error message",
			err:      errors.New("operation failed: timeout occurred"),
			expected: "Network connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)
			assert.Contains(t, result, tt.expected)
			assert.Contains(t, result, "This usually means:")
			assert.Contains(t, result, "• Your internet connection is unstable")
			assert.Contains(t, result, "• The Portainer server is slow to respond")
			assert.Contains(t, result, "• The server might be temporarily unavailable")
			assert.Contains(t, result, "Please check your connection and try again.")
		})
	}
}

func TestFormatError_ConnectionRefused(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: "Connection refused",
		},
		{
			name:     "connection refused with details",
			err:      errors.New("dial tcp: connection refused"),
			expected: "Connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)
			assert.Contains(t, result, tt.expected)
			assert.Contains(t, result, "This usually means:")
			assert.Contains(t, result, "• The Portainer URL is incorrect")
			assert.Contains(t, result, "• The Portainer server is not running")
			assert.Contains(t, result, "• There's a firewall blocking the connection")
			assert.Contains(t, result, "Please verify your Portainer URL and try again.")
		})
	}
}

func TestFormatError_Certificate(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "certificate error",
			err:      errors.New("certificate verify failed"),
			expected: "SSL/TLS certificate error",
		},
		{
			name:     "TLS error",
			err:      errors.New("TLS handshake failed"),
			expected: "SSL/TLS certificate error",
		},
		{
			name:     "certificate with details",
			err:      errors.New("x509: certificate signed by unknown authority"),
			expected: "SSL/TLS certificate error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)
			assert.Contains(t, result, tt.expected)
			assert.Contains(t, result, "This usually means:")
			assert.Contains(t, result, "• The SSL certificate is invalid or expired")
			assert.Contains(t, result, "• You're using a self-signed certificate")
			assert.Contains(t, result, "• There's a certificate authority issue")
			assert.Contains(t, result, "You can try again or contact your administrator.")
		})
	}
}

func TestFormatError_Generic(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: "Operation failed",
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown error occurred"),
			expected: "Operation failed",
		},
		{
			name:     "custom error message",
			err:      errors.New("custom error with details"),
			expected: "Operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.err)
			assert.Contains(t, result, tt.expected)
			assert.Contains(t, result, "Error details:")
			assert.Contains(t, result, tt.err.Error())
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		expected   bool
	}{
		{
			name:       "contains first substring",
			s:          "hello world",
			substrings: []string{"hello", "goodbye"},
			expected:   true,
		},
		{
			name:       "contains second substring",
			s:          "goodbye world",
			substrings: []string{"hello", "goodbye"},
			expected:   true,
		},
		{
			name:       "contains substring in middle",
			s:          "the quick brown fox",
			substrings: []string{"brown", "red"},
			expected:   true,
		},
		{
			name:       "contains substring at end",
			s:          "hello world",
			substrings: []string{"world", "universe"},
			expected:   true,
		},
		{
			name:       "no match",
			s:          "hello world",
			substrings: []string{"goodbye", "farewell"},
			expected:   false,
		},
		{
			name:       "partial match not counted",
			s:          "hello",
			substrings: []string{"helloworld", "goodbye"},
			expected:   false,
		},
		{
			name:       "case sensitive match",
			s:          "Hello World",
			substrings: []string{"hello", "world"},
			expected:   false,
		},
		{
			name:       "exact match",
			s:          "exact",
			substrings: []string{"exact", "match"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsAny_EmptyString(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		expected   bool
	}{
		{
			name:       "empty string with non-empty substrings",
			s:          "",
			substrings: []string{"hello", "world"},
			expected:   false,
		},
		{
			name:       "empty string with empty substring",
			s:          "",
			substrings: []string{""},
			expected:   true,
		},
		{
			name:       "empty string with empty substrings",
			s:          "",
			substrings: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsAny_NoMatch(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		expected   bool
	}{
		{
			name:       "no matching substrings",
			s:          "hello world",
			substrings: []string{"goodbye", "farewell", "adios"},
			expected:   false,
		},
		{
			name:       "longer substrings than string",
			s:          "hi",
			substrings: []string{"hello", "world", "goodbye"},
			expected:   false,
		},
		{
			name:       "empty substrings slice",
			s:          "hello world",
			substrings: []string{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test that FormatError handles different error types correctly
func TestFormatError_ErrorPriority(t *testing.T) {
	// Test that timeout errors take priority over other patterns
	timeoutErr := errors.New("context deadline exceeded: connection refused")
	result := FormatError(timeoutErr)
	assert.Contains(t, result, "Network connection timeout")
	assert.NotContains(t, result, "Connection refused")

	// Test that connection refused takes priority over certificate
	connErr := errors.New("connection refused: certificate error")
	result = FormatError(connErr)
	assert.Contains(t, result, "Connection refused")
	assert.NotContains(t, result, "SSL/TLS certificate error")

	// Test that certificate takes priority over generic
	certErr := errors.New("certificate error: something else")
	result = FormatError(certErr)
	assert.Contains(t, result, "SSL/TLS certificate error")
	assert.NotContains(t, result, "Operation failed")
}

// Test that the formatting includes proper styling markers
func TestFormatError_Styling(t *testing.T) {
	result := FormatError(errors.New("test error"))

	// Check that the result contains styling information (lipgloss styles)
	// The exact styling output depends on the terminal, but we can check for
	// the presence of the error message content
	assert.Contains(t, result, "Operation failed")
	assert.Contains(t, result, "Error details:")
	assert.Contains(t, result, "test error")
}
