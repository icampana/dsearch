package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/config"
	"github.com/icampana/dsearch/internal/devdocs"
)

var availableCmd = &cobra.Command{
	Use:   "available [query]",
	Short: "List available documentation from DevDocs",
	Long:  `Lists all available documentation from DevDocs that can be installed. Use version syntax for specific versions (e.g., dsearch install react@18)`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAvailable,
}

func runAvailable(cmd *cobra.Command, args []string) error {
	// Initialize paths
	cfg := config.DefaultPaths()
	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Try to load cached manifest first
	store := devdocs.NewStore(cfg.DataDir, cfg.CacheDir)
	manifest, err := store.LoadManifest()

	// If not cached or stale, fetch from DevDocs
	if err != nil {
		client := devdocs.NewClient("https://documents.devdocs.io")
		manifest, err = client.FetchManifest()
		if err != nil {
			return fmt.Errorf("fetching available docs: %w", err)
		}
		if err := store.SaveManifest(manifest); err != nil {
			return fmt.Errorf("caching manifest: %w", err)
		}
	}

	if len(manifest) == 0 {
		fmt.Println("No documentation available.")
		return nil
	}

	// Filter by query if provided
	query := ""
	if len(args) > 0 {
		query = strings.ToLower(args[0])
	}

	fmt.Printf("Available documentation (%d total):\n\n", len(manifest))

	// Group by first letter for easier navigation
	currentLetter := rune(' ')

	count := 0
	for _, doc := range manifest {
		// Filter by query
		if query != "" && !strings.Contains(strings.ToLower(doc.Name), query) {
			continue
		}

		// Print letter header
		if len(doc.Name) > 0 {
			firstLetter := rune(strings.ToUpper(string(doc.Name[0]))[0])
			if firstLetter != currentLetter {
				currentLetter = firstLetter
				fmt.Printf("\n[%s]\n", string(firstLetter))
			}
		}

		// Format version info
		versionInfo := doc.Release
		if doc.Version != "" {
			versionInfo = fmt.Sprintf("%s (%s)", doc.Release, doc.Version)
		}

		// Show doc with alias if available
		aliasStr := ""
		if doc.Alias != "" {
			aliasStr = fmt.Sprintf("[%s]", doc.Alias)
		}

		fmt.Printf("  %-30s %-15s %s %s\n", doc.Name, versionInfo, formatBytes(doc.DBSize), aliasStr)
		count++

		// Show limited results if there's a query
		if query != "" && count > 50 {
			fmt.Println("\n... (showing first 50 matches)")
			break
		}
	}

	if query == "" {
		fmt.Printf("\nTo install documentation, run:\n")
		fmt.Println("  dsearch install <doc-name>              # Install latest version")
		fmt.Println("  dsearch install <doc-name>@<version>     # Install specific version (e.g., react@18)")
	}
	return nil
}
