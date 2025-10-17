package deploy

import (
	"fmt"

	"pctl/internal/compose"
	"pctl/internal/config"
	"pctl/internal/portainer"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

var DeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a new stack to Portainer",
	Long: `Deploy a new Docker Compose stack to Portainer.
This command will create a new stack in the configured environment.
If the stack already exists, use 'pctl redeploy' instead.`,
	RunE: runDeploy,
}

func runDeploy(cmd *cobra.Command, args []string) error {
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
	fmt.Printf("  Compose File: %s\n", cfg.ComposeFile)
	fmt.Println()

	// Read compose file
	fmt.Println(infoStyle.Render("Reading compose file..."))
	composeContent, err := compose.ReadComposeFile(cfg.ComposeFile)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}
	fmt.Println(successStyle.Render("✓ Compose file loaded"))

	// Create Portainer client
	client := portainer.NewClientWithTLS(cfg.PortainerURL, cfg.APIToken, cfg.SkipTLSVerify)

	// Check if stack already exists
	fmt.Println(infoStyle.Render("Checking if stack already exists..."))
	existingStack, err := client.GetStack(cfg.StackName, cfg.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to check for existing stack: %w", err)
	}

	if existingStack != nil {
		return fmt.Errorf("stack '%s' already exists in environment %d. Use 'pctl redeploy' to update it", cfg.StackName, cfg.EnvironmentID)
	}

	// Create new stack
	fmt.Println(infoStyle.Render("Creating new stack..."))
	stack, err := client.CreateStack(cfg.StackName, composeContent, cfg.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to create stack: %w", err)
	}

	// Display success message
	fmt.Println()
	fmt.Println(successStyle.Render("✓ Stack deployed successfully!"))
	fmt.Println()
	fmt.Println(infoStyle.Render("Stack Details:"))
	fmt.Printf("  ID: %d\n", stack.ID)
	fmt.Printf("  Name: %s\n", stack.Name)
	fmt.Printf("  Environment ID: %d\n", stack.EnvironmentID)
	fmt.Printf("  Status: %d\n", stack.Status)
	fmt.Println()
	fmt.Println(infoStyle.Render("You can now use 'pctl redeploy' to update this stack."))

	return nil
}
