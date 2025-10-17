package deploy

import (
	"fmt"

	"github.com/deviantony/pctl/internal/compose"
	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/portainer"

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
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Stack already exists"))
		fmt.Println()
		fmt.Printf("Stack '%s' already exists in environment %d.\n", cfg.StackName, cfg.EnvironmentID)
		fmt.Println()
		fmt.Println(infoStyle.Render("To update this stack, run:"))
		fmt.Printf("  %s\n", infoStyle.Render("pctl redeploy"))
		fmt.Println()
		return nil // Exit cleanly without error
	}

	// Create new stack
	fmt.Println(infoStyle.Render("Creating new stack..."))
	stack, err := client.CreateStack(cfg.StackName, composeContent, cfg.EnvironmentID)
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Failed to create stack"))
		fmt.Println()
		fmt.Printf("Error: %v\n", err)
		fmt.Println()
		fmt.Println(infoStyle.Render("Common issues:"))
		fmt.Println("  • Port conflicts - check if ports are already in use")
		fmt.Println("  • Invalid compose file - verify your docker-compose.yml")
		fmt.Println("  • Network issues - check Portainer connectivity")
		fmt.Println()
		return nil // Exit cleanly without error
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
