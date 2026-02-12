package cli

import (
	"fmt"
	"os"

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

	var uninstallErrors []string
	successCount := 0

	for _, input := range args {
		slug := parseDocSlug(input)

		if !store.IsInstalled(slug) {
			uninstallErrors = append(uninstallErrors, fmt.Sprintf("doc '%s' is not installed", input))
			continue
		}

		fmt.Printf("Uninstalling %s...\n", slug)
		if err := store.Uninstall(slug); err != nil {
			uninstallErrors = append(uninstallErrors, fmt.Sprintf("failed to uninstall %s: %v", input, err))
			continue
		}
		fmt.Printf("Successfully uninstalled %s\n", slug)
		successCount++
	}

	// Report results
	if len(uninstallErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d uninstallation(s) failed:\n", len(uninstallErrors))
		for _, errMsg := range uninstallErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
		}
		if successCount == 0 {
			return fmt.Errorf("all uninstallations failed")
		}
		return fmt.Errorf("%d uninstallation(s) failed (see above)", len(uninstallErrors))
	}

	return nil
}
