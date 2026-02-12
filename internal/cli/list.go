package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/config"
	"github.com/icampana/dsearch/internal/devdocs"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed documentation",
	Long:  `Lists all DevDocs documentation installed in the docs directory.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg := config.DefaultPaths()
	store := devdocs.NewStore(cfg.DataDir, cfg.CacheDir)

	installedSlugs := store.ListInstalled()

	if len(installedSlugs) == 0 {
		fmt.Println("No documentation installed.")
		fmt.Printf("\nDocs directory: %s\n", cfg.DataDir)
		fmt.Println("\nTo install documentation, run:")
		fmt.Println("  dsearch install <doc-name>")
		fmt.Println("\nTo see available documentation:")
		fmt.Println("  dsearch available")
		return nil
	}

	// Load metadata for each installed doc
	type installedDoc struct {
		slug       string
		name       string
		release    string
		version    string
		entryCount int
		dbSize     int64
	}

	var installed []installedDoc

	// Load manifest for display names
	manifest, _ := store.LoadManifest()
	manifestMap := make(map[string]*devdocs.Doc)
	for i := range manifest {
		manifestMap[manifest[i].Slug] = &manifest[i]
	}

	for _, slug := range installedSlugs {
		// Load meta.json
		metaPath := filepath.Join(cfg.DataDir, "docs", slug, "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var meta devdocs.Meta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}

		// Load index for entry count
		index, err := store.LoadIndex(slug)
		if err != nil {
			continue
		}

		// Get name from manifest
		name := slug
		release := "unknown"
		version := ""
		if doc, ok := manifestMap[slug]; ok {
			name = doc.Name
			release = doc.Release
			version = doc.Version
		}

		installed = append(installed, installedDoc{
			slug:       slug,
			name:       name,
			release:    release,
			version:    version,
			entryCount: len(index.Entries),
			dbSize:     meta.DBSize,
		})
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tENTRIES\tSIZE")
	fmt.Fprintln(w, "----\t-------\t-------\t----")

	for _, doc := range installed {
		versionStr := doc.release
		if doc.version != "" {
			versionStr = fmt.Sprintf("%s (%s)", doc.release, doc.version)
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			doc.name,
			versionStr,
			doc.entryCount,
			formatBytes(doc.dbSize),
		)
	}
	w.Flush()

	fmt.Printf("\n%d documentation set(s) installed in %s\n", len(installed), cfg.DataDir)
	return nil
}
