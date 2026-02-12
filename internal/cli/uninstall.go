package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/config"
	"github.com/icampana/dsearch/internal/devdocs"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <doc>...",
	Short: "Uninstall documentation",
	Long:  `Uninstall documentation. Supports version syntax: react@18 for React 18.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runUninstall,
}

func runUninstall(cmd *cobra.Command, args []string) error {
	// Initialize paths
	cfg := config.DefaultPaths()
	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	store := devdocs.NewStore(cfg.DataDir, cfg.CacheDir)

	for _, input := range args {
		slug := parseDocSlug(input)

		if !store.IsInstalled(slug) {
			fmt.Printf("Doc '%s' is not installed\n", input)
			continue
		}

		fmt.Printf("Uninstalling %s...\n", slug)
		if err := store.Uninstall(slug); err != nil {
			fmt.Printf("Error uninstalling %s: %v\n", input, err)
			continue
		}
		fmt.Printf("Successfully uninstalled %s\n", slug)
	}

	return nil
}
