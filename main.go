package main

import (
	"fmt"
	"os"

	"pctl/cmd/deploy"
	initcmd "pctl/cmd/init"
	"pctl/cmd/logs"
	"pctl/cmd/ps"
	"pctl/cmd/redeploy"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pctl",
	Short: "Portainer Control CLI - Deploy and manage Docker Compose applications via Portainer",
	Long: `pctl is a developer companion tool for deploying and managing Docker Compose 
applications via Portainer. It streamlines the deployment workflow by providing 
simple commands to create, deploy, and redeploy stacks through Portainer's API.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initcmd.InitCmd)
	rootCmd.AddCommand(deploy.DeployCmd)
	rootCmd.AddCommand(logs.LogsCmd)
	rootCmd.AddCommand(ps.PsCmd)
	rootCmd.AddCommand(redeploy.RedeployCmd)
}
