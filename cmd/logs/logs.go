package logs

import (
	"fmt"
	"strings"

	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/portainer"
	"github.com/deviantony/pctl/internal/spinner"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	logStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
)

var (
	tailLines int
	service   string
)

var LogsCmd = &cobra.Command{
	Use:   "logs [flags]",
	Short: "View stack container logs",
	Long: `Display logs from containers in your deployed stack.
By default, shows the last 50 lines from all containers.
Use --service to filter logs from a specific service.`,
	RunE: runLogs,
}

func init() {
	LogsCmd.Flags().IntVarP(&tailLines, "tail", "t", 50, "Number of lines to show from the end of logs")
	LogsCmd.Flags().StringVarP(&service, "service", "s", "", "Show logs from specific service only")
}

func runLogs(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
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
	err = spinner.RunWithSpinner("Checking if stack exists...", func() error {
		var fetchErr error
		existingStack, fetchErr = client.GetStack(cfg.StackName, cfg.EnvironmentID)
		return fetchErr
	})
	if err != nil {
		return fmt.Errorf("failed to check for existing stack: %w", err)
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

	fmt.Println(successStyle.Render("✓ Stack found"))

	// Get containers for the stack
	var containers []portainer.Container
	err = spinner.RunWithSpinner("Fetching container information...", func() error {
		var fetchErr error
		containers, fetchErr = client.GetStackContainers(cfg.EnvironmentID, cfg.StackName)
		return fetchErr
	})
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Failed to fetch container information"))
		fmt.Println()
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println(infoStyle.Render("Note: Container information could not be retrieved."))
		fmt.Println("This might be due to Docker API access restrictions or filter issues.")
		fmt.Println()
		return nil // Exit cleanly without error
	}

	if len(containers) == 0 {
		fmt.Println()
		fmt.Println(infoStyle.Render("No containers found for this stack"))
		return nil
	}

	// Filter containers by service if specified
	if service != "" {
		containers = filterContainersByService(containers, service)
		if len(containers) == 0 {
			fmt.Println()
			fmt.Printf("No containers found for service '%s'\n", service)
			return nil
		}
	}

	// Display logs for each container
	fmt.Println()
	return displayLogs(client, containers, cfg.EnvironmentID)
}

func filterContainersByService(containers []portainer.Container, serviceName string) []portainer.Container {
	var filtered []portainer.Container
	for _, container := range containers {
		// Check if any of the container names match the service name
		for _, name := range container.Names {
			cleanName := strings.TrimPrefix(name, "/")

			// Try both underscore and hyphen separators
			// Docker Compose can use either format: stackname_servicename_1 or stackname-servicename-1
			var parts []string
			if strings.Contains(cleanName, "_") {
				parts = strings.Split(cleanName, "_")
			} else if strings.Contains(cleanName, "-") {
				parts = strings.Split(cleanName, "-")
			}

			if len(parts) >= 2 {
				// Get the service name part (second to last part)
				servicePart := parts[len(parts)-2]
				if servicePart == serviceName {
					filtered = append(filtered, container)
					break
				}
			}

			// Also check if the service name appears anywhere in the container name
			if strings.Contains(cleanName, serviceName) {
				filtered = append(filtered, container)
				break
			}
		}
	}
	return filtered
}

func displayLogs(client *portainer.Client, containers []portainer.Container, environmentID int) error {
	// Collect logs for all containers
	var containerLogs []ContainerLogs

	for _, container := range containers {
		containerName := getPrimaryContainerName(container.Names)

		// Fetch logs for this container
		logs, err := client.GetContainerLogs(environmentID, container.ID, tailLines)
		if err != nil {
			fmt.Printf("Error fetching logs for %s: %v\n", containerName, err)
			// Add empty logs entry to maintain container order
			containerLogs = append(containerLogs, ContainerLogs{
				Name: containerName,
				Logs: fmt.Sprintf("Error fetching logs: %v", err),
			})
			continue
		}

		containerLogs = append(containerLogs, ContainerLogs{
			Name: containerName,
			Logs: logs,
		})
	}

	// Run the interactive viewer
	return RunViewer(containerLogs)
}

func getPrimaryContainerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}

	// Get the first name and remove leading slash
	name := strings.TrimPrefix(names[0], "/")

	// Truncate if too long
	if len(name) > 50 {
		name = name[:47] + "..."
	}

	return name
}
