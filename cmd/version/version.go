package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information - these will be set at build time via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long: `Display version information including:
- Version number
- Git commit hash
- Build timestamp
- Go version used to build the binary
- Target platform (OS/Architecture)`,
	Run: runVersion,
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("pctl version %s\n", Version)
	fmt.Printf("  commit: %s\n", Commit)
	fmt.Printf("  built:  %s\n", BuildTime)
	fmt.Printf("  go:     %s\n", runtime.Version())
	fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
