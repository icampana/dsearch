package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/config"
	"github.com/icampana/dsearch/internal/devdocs"
)

var installCmd = &cobra.Command{
	Use:   "install <doc>...",
	Short: "Install documentation from DevDocs",
	Long:  `Downloads and installs documentation from DevDocs. Supports version syntax: react@18 for React 18.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runInstall,
}

// parseDocSlug converts user input like "react@18" to DevDocs slug "react~18"
func parseDocSlug(input string) string {
	if strings.Contains(input, "@") {
		parts := strings.Split(input, "@")
		if len(parts) == 2 {
			return fmt.Sprintf("%s~%s", parts[0], parts[1])
		}
	}
	return input
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Initialize paths
	cfg := config.DefaultPaths()
	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create DevDocs client and store
	// Empty string uses default DevDocs URLs (devdocs.io for manifest, documents.devdocs.io for content)
	client := devdocs.NewClient()
	store := devdocs.NewStore(cfg.DataDir, cfg.CacheDir)

	// Fetch manifest (or use cached)
	manifest, err := store.LoadManifest()
	if err != nil {
		// Manifest not cached, fetch it
		manifest, err = client.FetchManifest()
		if err != nil {
			return fmt.Errorf("failed to fetch manifest: %w", err)
		}
		if err := store.SaveManifest(manifest); err != nil {
			return fmt.Errorf("failed to cache manifest: %w", err)
		}
	}

	// Install each doc
	var installErrors []string
	successCount := 0

	for _, input := range args {
		slug := parseDocSlug(input)

		// Find doc in manifest
		var doc *devdocs.Doc
		for i := range manifest {
			if manifest[i].Slug == slug {
				doc = &manifest[i]
				break
			}
		}
		if doc == nil {
			installErrors = append(installErrors, fmt.Sprintf("doc '%s' not found in DevDocs catalog", input))
			continue
		}

		fmt.Printf("Installing %s (%s, %s)...\n", doc.Name, doc.Release, formatBytes(doc.DBSize))

		// Check for updates if already installed
		if store.IsInstalled(slug) {
			// TODO: Check mtime for updates
			fmt.Printf("Already installed, checking for updates...\n")
		}

		// Fetch index
		index, err := client.FetchIndex(slug)
		if err != nil {
			installErrors = append(installErrors, fmt.Sprintf("failed to fetch index for %s: %v", input, err))
			continue
		}

		// Fetch db
		db, err := client.FetchDB(slug)
		if err != nil {
			installErrors = append(installErrors, fmt.Sprintf("failed to fetch db for %s: %v", input, err))
			continue
		}

		// Install
		_, err = store.Install(slug, index, db, manifest)
		if err != nil {
			installErrors = append(installErrors, fmt.Sprintf("failed to install %s: %v", input, err))
			continue
		}

		fmt.Printf("Successfully installed %s (%d entries)\n", doc.Name, len(index.Entries))
		successCount++
	}

	// Report results
	if len(installErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d installation(s) failed:\n", len(installErrors))
		for _, errMsg := range installErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
		}
		if successCount == 0 {
			return fmt.Errorf("all installations failed")
		}
		return fmt.Errorf("%d installation(s) failed (see above)", len(installErrors))
	}

	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
