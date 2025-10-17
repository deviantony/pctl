package redeploy

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

var RedeployCmd = &cobra.Command{
	Use:   "redeploy",
	Short: "Redeploy an existing stack in Portainer",
	Long: `Redeploy an existing Docker Compose stack in Portainer.
This command will update the existing stack with the latest compose file
and pull the latest images. The stack must already exist (created via 'pctl deploy').`,
	RunE: runRedeploy,
}

func runRedeploy(cmd *cobra.Command, args []string) error {
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

	// Check if stack exists
	fmt.Println(infoStyle.Render("Checking if stack exists..."))
	existingStack, err := client.GetStack(cfg.StackName, cfg.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to check for existing stack: %w", err)
	}

	if existingStack == nil {
		return fmt.Errorf("stack '%s' not found in environment %d. Use 'pctl deploy' to create it first", cfg.StackName, cfg.EnvironmentID)
	}

	fmt.Printf("  Found existing stack with ID: %d\n", existingStack.ID)

	// Update existing stack
	fmt.Println(infoStyle.Render("Updating stack..."))
	err = client.UpdateStack(existingStack.ID, composeContent, true, cfg.EnvironmentID) // Pull images = true
	if err != nil {
		return fmt.Errorf("failed to update stack: %w", err)
	}

	// Display success message
	fmt.Println()
	fmt.Println(successStyle.Render("✓ Stack redeployed successfully!"))
	fmt.Println()
	fmt.Println(infoStyle.Render("Stack Details:"))
	fmt.Printf("  ID: %d\n", existingStack.ID)
	fmt.Printf("  Name: %s\n", existingStack.Name)
	fmt.Printf("  Environment ID: %d\n", existingStack.EnvironmentID)
	fmt.Println()
	fmt.Println(infoStyle.Render("The stack has been updated with the latest compose file and images have been pulled."))

	return nil
}
