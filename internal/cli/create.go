package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/crawler"
	"github.com/icampana/dsearch/internal/docset"
)

var (
	createURL       string
	createInclude   []string
	createExclude   []string
	createDepth     int
	createVersion   string
	createIcon      string
	createIndexPage string
	createTypeMap   string
	createSelector  string
	createDryRun    bool
	createForce     bool
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a custom docset from URL documentation",
	Long: `Creates a Dash-compatible docset by crawling documentation from a URL.

Uses Firecrawl to crawl and extract documentation, then builds a searchable
SQLite index. Supports differential updates to existing docsets.

Examples:
  # Create date-fns docset
  dsearch create date-fns --url https://date-fns.org/docs/Getting-Started

  # With custom type mappings
  dsearch create lodash --url https://lodash.com/docs \
    --type-map "H2:Function,H3:Method" \
    --version "4.17.21"

  # Dry run to preview
  dsearch create rxjs --url https://rxjs.dev/api --dry-run

  # Force complete rebuild
  dsearch create date-fns --url ... --force`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate required flag
	if createURL == "" {
		return fmt.Errorf("--url is required")
	}

	// Parse type mappings
	typeMappings, err := docset.ParseTypeMap(createTypeMap)
	if err != nil {
		return fmt.Errorf("parsing type-map: %w", err)
	}

	// Build configuration
	config := &docset.CreateConfig{
		Name:      name,
		URL:       createURL,
		Include:   createInclude,
		Exclude:   createExclude,
		Depth:     createDepth,
		Version:   createVersion,
		IconPath:  createIcon,
		IndexPage: createIndexPage,
		TypeMap:   typeMappings,
		Selector:  createSelector,
		DryRun:    createDryRun,
		Force:     createForce,
		DataDir:   paths.DataDir,
	}

	// Set defaults
	if config.Depth <= 0 {
		config.Depth = 2
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}

	// Check if docset already exists
	docsetPath := filepath.Join(paths.DataDir, fmt.Sprintf("%s.docset", name))
	exists := false
	if _, err := os.Stat(docsetPath); err == nil {
		exists = true
	}

	if exists && !createForce && !createDryRun {
		fmt.Printf("Docset '%s' already exists. Performing differential update...\n", name)
	}

	// Create crawler
	crawl, err := crawler.New(config.URL, config.Include, config.Exclude, config.Depth, config.Selector)
	if err != nil {
		return fmt.Errorf("initializing crawler: %w", err)
	}

	// Map URLs
	fmt.Printf("Mapping documentation URLs from %s...\n", config.URL)
	urls, err := crawl.MapURLs()
	if err != nil {
		return fmt.Errorf("mapping URLs: %w", err)
	}
	fmt.Printf("Found %d URLs to crawl\n", len(urls))

	if len(urls) == 0 {
		return fmt.Errorf("no URLs found to crawl")
	}

	// Scrape pages
	fmt.Println("Scraping documentation pages...")
	pages, err := crawl.ScrapePages(urls)
	if err != nil {
		return fmt.Errorf("scraping pages: %w", err)
	}
	fmt.Printf("Successfully scraped %d pages\n", len(pages))

	// Extract entries
	fmt.Println("Extracting searchable entries...")
	extractor := docset.NewExtractor(typeMappings)
	entries := extractor.ExtractFromPages(pages)
	fmt.Printf("Extracted %d entries\n", len(entries))

	if createDryRun {
		fmt.Println("\n--- Dry Run Preview ---")
		fmt.Printf("Docset: %s\n", config.Name)
		fmt.Printf("Version: %s\n", config.Version)
		fmt.Printf("URLs: %d\n", len(urls))
		fmt.Printf("Entries: %d\n", len(entries))
		fmt.Println("\nSample entries:")
		for i, e := range entries {
			if i >= 10 {
				fmt.Printf("... and %d more\n", len(entries)-10)
				break
			}
			fmt.Printf("  - %s [%s] -> %s\n", e.Name, e.Type, e.Path)
		}
		fmt.Println("\nUse without --dry-run to create/update the docset")
		return nil
	}

	// Build or update docset
	builder := docset.NewBuilder(config)

	if exists && !createForce {
		// Differential update
		fmt.Println("Performing differential update...")
		if err := builder.Update(entries, pages); err != nil {
			return fmt.Errorf("updating docset: %w", err)
		}
		fmt.Printf("Successfully updated docset '%s'\n", config.Name)
	} else {
		// Create new docset
		if exists && createForce {
			fmt.Println("Force flag set, removing existing docset...")
			os.RemoveAll(docsetPath)
		}

		fmt.Println("Creating new docset...")
		if err := builder.Create(entries, pages); err != nil {
			return fmt.Errorf("creating docset: %w", err)
		}
		fmt.Printf("Successfully created docset '%s'\n", config.Name)
	}

	return nil
}

func init() {
	createCmd.Flags().StringVar(&createURL, "url", "", "Starting URL to crawl (required)")
	createCmd.Flags().StringSliceVar(&createInclude, "include", nil, "URL patterns to include")
	createCmd.Flags().StringSliceVar(&createExclude, "exclude", nil, "URL patterns to skip")
	createCmd.Flags().IntVar(&createDepth, "depth", 2, "Crawl depth (default: 2)")
	createCmd.Flags().StringVar(&createVersion, "version", "1.0.0", "Docset version")
	createCmd.Flags().StringVar(&createIcon, "icon", "", "Path to icon file (.png)")
	createCmd.Flags().StringVar(&createIndexPage, "index-page", "", "Main page path/URL")
	createCmd.Flags().StringVar(&createTypeMap, "type-map", "", "Custom entry type mappings (e.g., 'H2:Function,H3:Method')")
	createCmd.Flags().StringVar(&createSelector, "selector", "", "CSS selector for Firecrawl extraction")
	createCmd.Flags().BoolVar(&createDryRun, "dry-run", false, "Preview changes without modifying docset")
	createCmd.Flags().BoolVar(&createForce, "force", false, "Overwrite existing docset completely")

	createCmd.MarkFlagRequired("url")
}
