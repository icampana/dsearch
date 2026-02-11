package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/config"
	"github.com/icampana/dsearch/internal/docset"
	"github.com/icampana/dsearch/internal/render"
	"github.com/icampana/dsearch/internal/search"
)

var (
	// Global flags
	cfgFile   string
	docsets   []string
	format    string
	limit     int
	entryType string
	listOnly  bool
	full      bool

	// Paths for XDG directories
	paths config.Paths

	// Search engine instance
	searchEngine *search.Engine
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dsearch [query]",
	Short: "Search API documentation offline",
	Long: `dsearch is a fast, offline documentation search tool.

It searches through Dash-compatible docsets and displays results
in your terminal. Supports fuzzy matching, multiple output formats,
and an interactive TUI mode.

Examples:
  dsearch useState                # Search for "useState" in all docsets
  dsearch useState -d react       # Search only in React docset
  dsearch useState --format md    # Output as markdown
  dsearch useState --full          # Show full content without truncation
  dsearch                             # Launch interactive TUI`,
	RunE: runSearch,
	Args: cobra.MaximumNArgs(1), // Accept at most one query argument
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize search engine before running commands
		if cmd.Name() == "list" || cmd.Name() == "version" {
			return nil // Skip engine initialization for these commands
		}
		allDocsets, err := docset.Discover(paths.DataDir)
		if err != nil {
			return fmt.Errorf("discovering docsets: %w", err)
		}
		searchEngine = search.New(allDocsets, entryType, limit)
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
	rootCmd.PersistentFlags().StringSliceVarP(&docsets, "docset", "d", nil, "filter to specific docset(s)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "output format: text, md")
	rootCmd.PersistentFlags().IntVarP(&limit, "limit", "l", 10, "maximum number of results")
	rootCmd.PersistentFlags().StringVarP(&entryType, "type", "t", "", "filter by entry type (Function, Class, Method, etc.)")
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
		// No query provided - launch interactive TUI
		return runInteractive()
	}

	// Direct search mode
	query := args[0]
	return runDirectSearch(query)
}

func runInteractive() error {
	// TODO: Implement TUI
	fmt.Println("Interactive TUI mode - coming soon!")
	fmt.Println("For now, provide a search query: dsearch <query>")
	fmt.Println("\nTo see available commands:")
	fmt.Println("  dsearch --help")
	return nil
}

func runDirectSearch(query string) error {
	// Perform search
	results, err := searchEngine.Search(query, docsets)
	if err != nil {
		return err
	}

	// Display results
	if listOnly {
		printResultList(results)
		return nil
	}

	// For now, just show the first result
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Display best match
	result := results[0]
	fmt.Printf("\n%s [%s]\n", result.Name, result.Type)
	fmt.Printf("  Docset: %s\n", result.Docset)
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
	maxDocset := 0
	for _, r := range results {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
		if len(r.Type) > maxType {
			maxType = len(r.Type)
		}
		if len(r.Docset) > maxDocset {
			maxDocset = len(r.Docset)
		}
	}

	// Print results
	for i, r := range results {
		fmt.Printf("%2d. %-*s  %-*s  %-*s  %.2f\n",
			i+1,
			maxName, r.Name,
			maxType, r.Type,
			maxDocset, r.Docset,
			r.Score,
		)
	}
}

// getDocsetContent reads the HTML content for a search result.
func getDocsetContent(result search.Result) ([]byte, error) {
	return os.ReadFile(result.FullPath)
}
