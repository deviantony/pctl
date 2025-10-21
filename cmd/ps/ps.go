package ps

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/errors"
	"github.com/deviantony/pctl/internal/portainer"
	"github.com/deviantony/pctl/internal/spinner"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
)

var PsCmd = &cobra.Command{
	Use:   "ps",
	Short: "Show stack status and running containers",
	Long: `Display the status of your deployed stack and its running containers.
Shows stack information, container status, ports, and resource usage.`,
	RunE:         runPs,
	SilenceUsage: true,
}

func runPs(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(errorStyle.Render("✗ Configuration error"))
		fmt.Println()
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		return nil // Exit cleanly without showing usage
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	fmt.Println(infoStyle.Render("Loading configuration..."))
	fmt.Printf("  Environment ID: %d\n", cfg.EnvironmentID)
	fmt.Printf("  Stack Name: %s\n", cfg.StackName)
	fmt.Println()

	// Create Portainer client
	client := portainer.NewClientWithTLS(cfg.PortainerURL, cfg.APIToken, cfg.SkipTLSVerify)

	// Check if stack exists
	var existingStack *portainer.Stack
	err = spinner.RunWithSpinnerAndSuccess("Checking if stack exists...", "✓ Stack found", func() error {
		var fetchErr error
		existingStack, fetchErr = client.GetStack(cfg.StackName, cfg.EnvironmentID)
		return fetchErr
	})
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Failed to check for existing stack"))
		fmt.Println()
		fmt.Println(errors.FormatError(err))
		fmt.Println()
		return nil // Exit cleanly without showing usage
	}

	if existingStack == nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Stack not found"))
		fmt.Println()
		fmt.Printf("Stack '%s' not found in environment %d.\n", cfg.StackName, cfg.EnvironmentID)
		fmt.Println()
		fmt.Println(infoStyle.Render("To deploy this stack, run:"))
		fmt.Printf("  %s\n", infoStyle.Render("pctl deploy"))
		fmt.Println()
		return nil // Exit cleanly without error
	}

	// Get detailed stack information
	var stackDetails *portainer.StackDetails
	err = spinner.RunWithSpinnerAndSuccess("Fetching stack details...", "✓ Stack details retrieved", func() error {
		var fetchErr error
		stackDetails, fetchErr = client.GetStackDetails(existingStack.ID)
		return fetchErr
	})
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Failed to get stack details"))
		fmt.Println()
		fmt.Println(errors.FormatError(err))
		fmt.Println()
		return nil // Exit cleanly without showing usage
	}

	// Get containers for the stack
	var containers []portainer.Container
	err = spinner.RunWithSpinnerAndSuccess("Fetching container information...", "✓ Container information loaded", func() error {
		var fetchErr error
		containers, fetchErr = client.GetStackContainers(cfg.EnvironmentID, cfg.StackName)
		return fetchErr
	})
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Failed to fetch container information"))
		fmt.Println()
		fmt.Println(errors.FormatError(err))
		fmt.Println()
		fmt.Println(infoStyle.Render("Stack information (containers unavailable):"))
		fmt.Println()
		displayStackInfo(stackDetails)
		fmt.Println()
		fmt.Println(infoStyle.Render("Note: Container information could not be retrieved."))
		fmt.Println("This might be due to Docker API access restrictions or filter issues.")
		fmt.Println()
		return nil // Exit cleanly without error
	}

	// Display results
	fmt.Println()
	displayStackInfo(stackDetails)
	fmt.Println()
	displayContainers(containers)

	return nil
}

func displayStackInfo(stack *portainer.StackDetails) {
	fmt.Println(headerStyle.Render("Stack Information:"))
	fmt.Printf("  Name: %s\n", stack.Name)
	fmt.Printf("  ID: %d\n", stack.ID)
	fmt.Printf("  Status: %s\n", getStatusText(stack.Status))
	fmt.Printf("  Environment ID: %d\n", stack.EnvironmentID)

	if stack.CreatedAt > 0 {
		createdTime := time.Unix(stack.CreatedAt, 0)
		fmt.Printf("  Created: %s by %s\n", createdTime.Format("2006-01-02 15:04:05"), stack.CreatedBy)
	}

	if stack.UpdatedAt > 0 {
		updatedTime := time.Unix(stack.UpdatedAt, 0)
		fmt.Printf("  Updated: %s by %s\n", updatedTime.Format("2006-01-02 15:04:05"), stack.UpdatedBy)
	}
}

func displayContainers(containers []portainer.Container) {
	if len(containers) == 0 {
		fmt.Println(headerStyle.Render("Containers:"))
		fmt.Println("  No containers found for this stack")
		return
	}

	fmt.Println(headerStyle.Render("Containers:"))

	// No longer need table columns since we're using simple formatting

	// Create table rows
	var rows []table.Row
	for _, container := range containers {
		name := getPrimaryContainerName(container.Names)
		image := container.Image
		if len(image) > 20 {
			image = image[:17] + "..."
		}

		status := getContainerStatus(container)
		ports := formatExposedPorts(container.Ports)

		rows = append(rows, table.Row{name, image, status, ports})
	}

	// Use simple formatted output instead of table to avoid indentation issues
	fmt.Printf("%-25s %-20s %-15s %-15s\n",
		headerStyle.Render("NAME"),
		headerStyle.Render("IMAGE"),
		headerStyle.Render("STATUS"),
		headerStyle.Render("PORTS"))
	fmt.Println(strings.Repeat("─", 75))
	for _, row := range rows {
		fmt.Printf("%-25s %-20s %-15s %-15s\n", row[0], row[1], row[2], row[3])
	}
}

func getStatusText(status int) string {
	switch status {
	case 1:
		return "Active"
	case 2:
		return "Inactive"
	default:
		return fmt.Sprintf("Unknown (%d)", status)
	}
}

func getContainerStatus(container portainer.Container) string {
	status := container.Status
	if container.State == "running" {
		// Try to extract uptime from status if available
		if strings.Contains(status, "Up") {
			return status
		}
		return "Up"
	}
	return status
}

func getPrimaryContainerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}

	// Get the first name and remove leading slash
	name := strings.TrimPrefix(names[0], "/")

	// Truncate if too long
	if len(name) > 25 {
		name = name[:22] + "..."
	}

	return name
}

func formatExposedPorts(ports []portainer.Port) string {
	if len(ports) == 0 {
		return ""
	}

	// Use a map to track unique ports
	uniquePorts := make(map[int]bool)
	for _, port := range ports {
		// Only show ports that are exposed to the host (have PublicPort)
		if port.PublicPort > 0 {
			uniquePorts[port.PublicPort] = true
		}
	}

	if len(uniquePorts) == 0 {
		return "none"
	}

	// Convert map keys to slice and sort them
	var exposedPorts []string
	for port := range uniquePorts {
		exposedPorts = append(exposedPorts, fmt.Sprintf("%d", port))
	}

	// Sort ports for consistent display
	sort.Strings(exposedPorts)

	result := strings.Join(exposedPorts, ", ")
	if len(result) > 15 {
		result = result[:12] + "..."
	}
	return result
}
