package cli

import (
	"encoding/json"
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
	cfgFile    string
	docs       []string
	format     string
	limit      int
	listOnly   bool
	full       bool
	jsonOutput bool

	// Paths for XDG directories
	paths config.Paths
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
  dsearch useState --json      # Output results as JSON`,
	RunE: runSearch,
	Args: cobra.MaximumNArgs(1),
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
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output results as JSON")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(availableCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	paths = config.DefaultPaths()
	if err := paths.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create directories: %v\n", err)
	}

	// One-time migration from old double-nested path
	if err := config.MigrateDataDir(paths.DataDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: migration failed: %v\n", err)
	}
}

func loadSearchEngine() (*search.Engine, *devdocs.Store, error) {
	store := devdocs.NewStore(paths.DataDir, paths.CacheDir)
	installedSlugs := store.ListInstalled()

	if len(installedSlugs) == 0 {
		return nil, nil, fmt.Errorf("no documentation installed. Run 'dsearch install <doc>' to install documentation")
	}

	// Optimization: If user specified docs, only load those
	slugsToLoad := installedSlugs
	if len(docs) > 0 {
		// Verify requested docs are installed
		validSlug := make(map[string]bool)
		for _, s := range installedSlugs {
			validSlug[s] = true
		}

		filtered := make([]string, 0)
		for _, d := range docs {
			if validSlug[d] {
				filtered = append(filtered, d)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: doc '%s' is not installed\n", d)
			}
		}
		if len(filtered) > 0 {
			slugsToLoad = filtered
		}
	}

	allIndices := make([]*devdocs.Index, 0, len(slugsToLoad))
	indicesBySlug := make(map[string]*devdocs.Index, len(slugsToLoad))

	for _, slug := range slugsToLoad {
		index, err := store.LoadIndex(slug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load index for %s: %v\n", slug, err)
			continue
		}
		allIndices = append(allIndices, index)
		indicesBySlug[slug] = index
	}

	if len(allIndices) == 0 {
		return nil, nil, fmt.Errorf("no documentation could be loaded")
	}

	return search.New(allIndices, indicesBySlug, limit), store, nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	// Initialize search engine just-in-time
	engine, store, err := loadSearchEngine()
	if err != nil {
		return err
	}

	query := args[0]

	// Perform search
	// Pass nil for docs because we already filtered at load time (optimization)
	results, warning, err := engine.Search(query, nil)
	if err != nil {
		return err
	}

	if warning != "" && !jsonOutput {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n\n", warning)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	if listOnly {
		printResultList(results)
		return nil
	}

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

	content, err := store.LoadContent(result.Slug, result.Path)
	if err != nil {
		return fmt.Errorf("reading content: %w", err)
	}

	renderer := render.New(render.Format(format))
	rendered, err := renderer.Render([]byte(content))
	if err != nil {
		return fmt.Errorf("rendering content: %w", err)
	}

	maxLength := 2000
	if full {
		maxLength = len(rendered)
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
