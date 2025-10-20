package redeploy

import (
	"fmt"

	"github.com/deviantony/pctl/internal/build"
	"github.com/deviantony/pctl/internal/compose"
	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/errors"
	"github.com/deviantony/pctl/internal/portainer"
	"github.com/deviantony/pctl/internal/spinner"

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
	RunE:         runRedeploy,
	SilenceUsage: true,
}

// forceRebuild toggles forcing both build.ForceBuild and build.NoCache during this run
var forceRebuild bool

func init() {
	RedeployCmd.Flags().BoolVarP(&forceRebuild, "force-rebuild", "f", false, "Force rebuild images (sets force_build and no_cache for this run)")
}

func runRedeploy(cmd *cobra.Command, args []string) error {
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
	fmt.Printf("  Compose File: %s\n", cfg.ComposeFile)
	fmt.Println()

	// Read compose file
	fmt.Println(infoStyle.Render("Reading compose file..."))
	composeContent, err := compose.ReadComposeFile(cfg.ComposeFile)
	if err != nil {
		return fmt.Errorf("failed to read compose file: %w", err)
	}
	fmt.Println(successStyle.Render("✓ Compose file loaded"))

	// Parse compose file to check for build directives
	composeFile, err := compose.ParseComposeFile(composeContent)
	if err != nil {
		return fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Check if there are build directives
	hasBuild, err := composeFile.HasBuildDirectives()
	if err != nil {
		return fmt.Errorf("failed to check for build directives: %w", err)
	}

	var finalComposeContent string
	if hasBuild {
		// Get build configuration
		buildConfig := cfg.GetBuildConfig()

		// Apply CLI override if requested
		if forceRebuild {
			buildConfig.ForceBuild = true
			fmt.Println(infoStyle.Render("Force rebuild enabled: force_build=true (no-cache)"))
		}

		// Validate build configuration
		if err := buildConfig.Validate(); err != nil {
			return fmt.Errorf("invalid build configuration: %w", err)
		}

		fmt.Println(infoStyle.Render("Build directives detected, processing builds..."))

		// Find services with build directives
		servicesWithBuild, err := composeFile.FindServicesWithBuild()
		if err != nil {
			return fmt.Errorf("failed to find services with build directives: %w", err)
		}

		// Validate build contexts
		if err := composeFile.ValidateBuildContexts(); err != nil {
			return fmt.Errorf("build context validation failed: %w", err)
		}

		// Create Portainer client
		client := portainer.NewClientWithTLS(cfg.PortainerURL, cfg.APIToken, cfg.SkipTLSVerify)

		// Create build orchestrator
		logger := build.NewStyledBuildLogger("BUILD")
		orchestrator := build.NewBuildOrchestrator(client, buildConfig, cfg.EnvironmentID, cfg.StackName, logger)

		// Build services
		imageTags, err := orchestrator.BuildServices(servicesWithBuild)
		if err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		// Transform compose file
		transformer, err := compose.TransformComposeFile(composeContent, imageTags)
		if err != nil {
			return fmt.Errorf("failed to transform compose file: %w", err)
		}

		// Validate transformation
		if err := transformer.ValidateTransformation(); err != nil {
			return fmt.Errorf("compose transformation validation failed: %w", err)
		}

		finalComposeContent = transformer.TransformedContent

		fmt.Println(successStyle.Render("✓ Build completed and compose file transformed"))
	} else {
		finalComposeContent = composeContent
		fmt.Println(infoStyle.Render("No build directives found, using compose file as-is"))
	}

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

	fmt.Printf("  Found existing stack with ID: %d\n", existingStack.ID)

	// Update existing stack
	pullImages := !hasBuild // Don't pull images if we just built them
	err = spinner.RunWithSpinner("Updating stack...", func() error {
		return client.UpdateStack(existingStack.ID, finalComposeContent, pullImages, cfg.EnvironmentID)
	})
	if err != nil {
		fmt.Println()
		fmt.Println(errorStyle.Render("✗ Failed to update stack"))
		fmt.Println()
		fmt.Println(errors.FormatError(err))
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
