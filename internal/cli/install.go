package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/manager"
)

var installCmd = &cobra.Command{
	Use:   "install <docset>...",
	Short: "Install docsets from Kapeli feeds",
	Long:  `Downloads and installs Dash docsets from Kapeli feeds.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	for _, name := range args {
		fmt.Printf("Installing %s...\n", name)
		if err := manager.Install(name, paths.DataDir); err != nil {
			fmt.Printf("Error installing %s: %v\n", name, err)
			continue
		}
		fmt.Printf("Successfully installed %s\n", name)
	}
	return nil
}
