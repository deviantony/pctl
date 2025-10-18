package errors

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// FormatError converts technical errors to user-friendly messages
func FormatError(err error) string {
	errStr := err.Error()

	if containsAny(errStr, []string{"context deadline exceeded", "timeout"}) {
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s",
			warningStyle.Render("Network connection timeout"),
			infoStyle.Render("This usually means:"),
			infoStyle.Render("• Your internet connection is unstable"),
			infoStyle.Render("• The Portainer server is slow to respond"),
			infoStyle.Render("• The server might be temporarily unavailable"),
			infoStyle.Render("\nPlease check your connection and try again."))
	}

	if containsAny(errStr, []string{"connection refused"}) {
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s",
			warningStyle.Render("Connection refused"),
			infoStyle.Render("This usually means:"),
			infoStyle.Render("• The Portainer URL is incorrect"),
			infoStyle.Render("• The Portainer server is not running"),
			infoStyle.Render("• There's a firewall blocking the connection"),
			infoStyle.Render("\nPlease verify your Portainer URL and try again."))
	}

	if containsAny(errStr, []string{"certificate", "TLS"}) {
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s\n%s",
			warningStyle.Render("SSL/TLS certificate error"),
			infoStyle.Render("This usually means:"),
			infoStyle.Render("• The SSL certificate is invalid or expired"),
			infoStyle.Render("• You're using a self-signed certificate"),
			infoStyle.Render("• There's a certificate authority issue"),
			infoStyle.Render("\nYou can try again or contact your administrator."))
	}

	// Generic error message
	return fmt.Sprintf("%s\n\n%s\n%s",
		warningStyle.Render("Operation failed"),
		infoStyle.Render("Error details:"),
		errorStyle.Render(errStr))
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
