package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/config"
	"github.com/icampana/dsearch/internal/devdocs"
	"github.com/icampana/dsearch/internal/render"
	"github.com/icampana/dsearch/internal/search"
)

var (
	// Global flags
	cfgFile  string
	docs     []string
	format   string
	limit    int
	listOnly bool
	full     bool

	// Paths for XDG directories
	paths config.Paths

	// Search engine instance
	searchEngine  *search.Engine
	allIndices    []*devdocs.Index
	indicesBySlug map[string]*devdocs.Index
	store         *devdocs.Store
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dsearch [query]",
	Short: "Search API documentation offline",
	Long: `dsearch is a fast, offline documentation search tool.

It searches through DevDocs documentation and displays results
in your terminal. Supports fuzzy matching and multiple output formats.

Examples:
  dsearch useState              # Search for "useState" in all installed docs
  dsearch useState -d react    # Search only in React documentation
  dsearch useState --format md # Output as markdown
  dsearch useState --full       # Show full content without truncation
  dsearch --list               # List all search results`,
	RunE: runSearch,
	Args: cobra.MaximumNArgs(1), // Accept at most one query argument
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize search engine before running commands
		if cmd.Name() == "list" || cmd.Name() == "version" || cmd.Name() == "available" || cmd.Name() == "install" {
			return nil // Skip engine initialization for these commands
		}

		// Load all installed indices
		store = devdocs.NewStore(paths.DataDir, paths.CacheDir)
		installedSlugs := store.ListInstalled()

		allIndices = make([]*devdocs.Index, 0, len(installedSlugs))
		indicesBySlug = make(map[string]*devdocs.Index, len(installedSlugs))

		for _, slug := range installedSlugs {
			index, err := store.LoadIndex(slug)
			if err != nil {
				// Skip docs that can't be loaded
				continue
			}
			allIndices = append(allIndices, index)
			indicesBySlug[slug] = index
		}

		if len(allIndices) == 0 {
			return fmt.Errorf("no documentation installed. Run 'dsearch install <doc>' to install documentation")
		}

		searchEngine = search.New(allIndices, indicesBySlug, limit)
		return nil
	},
}

// Execute adds all child commands to root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags (available to all commands)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $XDG_CONFIG_HOME/dsearch/config.yaml)")
	rootCmd.PersistentFlags().StringSliceVarP(&docs, "doc", "d", nil, "filter to specific doc(s)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "output format: text, md")
	rootCmd.PersistentFlags().IntVarP(&limit, "limit", "l", 10, "maximum number of results")
	rootCmd.PersistentFlags().BoolVar(&listOnly, "list", false, "list results only, don't show content")
	rootCmd.PersistentFlags().BoolVar(&full, "full", false, "show full content without truncation")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(availableCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	paths = config.DefaultPaths()
	if err := paths.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create directories: %v\n", err)
	}
}

func runSearch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// No query provided - show help
		return cmd.Help()
	}

	// Direct search mode
	query := args[0]
	return runDirectSearch(query)
}

func runDirectSearch(query string) error {
	// Perform search
	results, warning, err := searchEngine.Search(query, docs)
	if err != nil {
		return err
	}

	// Display warning if any
	if warning != "" {
		fmt.Printf("⚠️  %s\n\n", warning)
	}

	// Display results
	if listOnly {
		printResultList(results)
		return nil
	}

	// For now, just show first result
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Display best match
	result := results[0]
	fmt.Printf("\n%s [%s]\n", result.Name, result.Type)
	fmt.Printf("  Doc: %s\n", result.Slug)
	fmt.Printf("  Score: %.2f\n", result.Score)
	fmt.Printf("  Path: %s\n", result.Path)
	fmt.Println("\n--- Content ---")

	// Get HTML content
	content, err := getDocsetContent(result)
	if err != nil {
		return fmt.Errorf("reading content: %w", err)
	}

	// Clean content (remove navigation/clutter)
	cleanContent := render.CleanContent(content)

	// Render based on format
	renderer := render.New(render.Format(format))
	rendered, err := renderer.Render(cleanContent)
	if err != nil {
		return fmt.Errorf("rendering content: %w", err)
	}

	// Truncate if too long (unless --full flag)
	maxLength := 2000
	if full {
		maxLength = len(rendered) // No truncation with --full
	}

	if len(rendered) > maxLength {
		rendered = rendered[:maxLength]
		if !full {
			rendered = rendered + "\n\n... (truncated)"
		}
	}

	fmt.Println(rendered)

	return nil
}

func printResultList(results []search.Result) {
	fmt.Printf("Found %d result(s):\n\n", len(results))

	// Calculate max widths for alignment
	maxName := 0
	maxType := 0
	maxDoc := 0
	for _, r := range results {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
		if len(r.Type) > maxType {
			maxType = len(r.Type)
		}
		if len(r.Slug) > maxDoc {
			maxDoc = len(r.Slug)
		}
	}

	// Print results
	for i, r := range results {
		fmt.Printf("%2d. %-*s  %-*s  %-*s  %.2f\n",
			i+1,
			maxName, r.Name,
			maxType, r.Type,
			maxDoc, r.Slug,
			r.Score,
		)
	}
}

// getDocsetContent reads HTML content for a search result from devdocs store.
func getDocsetContent(result search.Result) ([]byte, error) {
	// Load from store using slug and path
	content, err := store.LoadContent(result.Slug, result.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load content: %w", err)
	}
	return []byte(content), nil
}
