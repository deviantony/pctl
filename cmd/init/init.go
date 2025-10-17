package init

import (
	"fmt"
	"os"
	"strings"

	"github.com/deviantony/pctl/internal/compose"
	"github.com/deviantony/pctl/internal/config"
	"github.com/deviantony/pctl/internal/portainer"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize pctl configuration",
	Long: `Initialize pctl by creating a configuration file with your Portainer settings.
This command will guide you through setting up your Portainer URL, API token,
environment selection, and other deployment options.`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if config already exists
	if _, err := os.Stat(config.ConfigFileName); err == nil {
		fmt.Println(errorStyle.Render("Configuration file 'pctl.yml' already exists."))
		fmt.Println(infoStyle.Render("If you want to reconfigure, please delete the existing file first."))
		return nil
	}

	var formData struct {
		PortainerURL  string
		APIToken      string
		EnvironmentID int
		StackName     string
		ComposeFile   string
	}

	// Create the interactive form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Portainer URL").
				Description("Enter your Portainer instance URL (e.g., https://portainer.example.com)").
				Value(&formData.PortainerURL).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("Portainer URL is required")
					}
					return portainer.ValidateURL(str)
				}),

			huh.NewInput().
				Title("API Token").
				Description("Enter your Portainer API token (starts with 'ptr_')").
				Value(&formData.APIToken).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("API token is required")
					}
					if !strings.HasPrefix(str, "ptr_") {
						return fmt.Errorf("API token should start with 'ptr_'")
					}
					return nil
				}),
		),
	)

	// Run the form
	if err := form.Run(); err != nil {
		return fmt.Errorf("failed to run form: %w", err)
	}

	// Create Portainer client to fetch environments (using default TLS settings)
	client := portainer.NewClient(formData.PortainerURL, formData.APIToken)
	environments, err := client.GetEnvironments()
	if err != nil {
		return fmt.Errorf("failed to fetch environments from Portainer: %w", err)
	}

	if len(environments) == 0 {
		return fmt.Errorf("no environments found in Portainer")
	}

	// Create environment selection options
	var envOptions []huh.Option[int]
	for _, env := range environments {
		envOptions = append(envOptions, huh.NewOption(env.Name, env.ID))
	}

	// Create second form for environment selection and other options
	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Environment").
				Description("Select the Portainer environment where you want to deploy").
				Options(envOptions...).
				Value(&formData.EnvironmentID),

			huh.NewInput().
				Title("Stack Name").
				Description("Enter a name for your stack").
				Value(&formData.StackName).
				Placeholder(config.GetDefaultStackName()).
				Validate(func(str string) error {
					if str == "" {
						formData.StackName = config.GetDefaultStackName()
					}
					return nil
				}),

			huh.NewInput().
				Title("Compose File").
				Description("Path to your Docker Compose file").
				Value(&formData.ComposeFile).
				Placeholder(config.GetDefaultComposeFile()).
				Validate(func(str string) error {
					if str == "" {
						formData.ComposeFile = config.GetDefaultComposeFile()
					}
					// Validate that the compose file exists
					return compose.ValidateComposeFile(formData.ComposeFile)
				}),
		),
	)

	// Run the second form
	if err := form2.Run(); err != nil {
		return fmt.Errorf("failed to run form: %w", err)
	}

	// Set defaults if not provided
	if formData.StackName == "" {
		formData.StackName = config.GetDefaultStackName()
	}
	if formData.ComposeFile == "" {
		formData.ComposeFile = config.GetDefaultComposeFile()
	}

	// Create and save configuration
	cfg := &config.Config{
		PortainerURL:  formData.PortainerURL,
		APIToken:      formData.APIToken,
		EnvironmentID: formData.EnvironmentID,
		StackName:     formData.StackName,
		ComposeFile:   formData.ComposeFile,
		SkipTLSVerify: config.GetDefaultSkipTLSVerify(), // Use default value
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Display success message
	fmt.Println()
	fmt.Println(successStyle.Render("✓ Configuration saved successfully!"))
	fmt.Println()
	fmt.Println(infoStyle.Render("Configuration Summary:"))
	fmt.Printf("  Portainer URL: %s\n", formData.PortainerURL)
	fmt.Printf("  Environment: %s (ID: %d)\n", getEnvironmentName(environments, formData.EnvironmentID), formData.EnvironmentID)
	fmt.Printf("  Stack Name: %s\n", formData.StackName)
	fmt.Printf("  Compose File: %s\n", formData.ComposeFile)
	fmt.Println()
	fmt.Println(infoStyle.Render("You can now use 'pctl deploy' to deploy your stack or 'pctl redeploy' to update an existing stack."))

	return nil
}

func getEnvironmentName(environments []portainer.Environment, id int) string {
	for _, env := range environments {
		if env.ID == id {
			return env.Name
		}
	}
	return "Unknown"
}
