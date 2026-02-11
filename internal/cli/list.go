package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/docset"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed docsets",
	Long:  `Lists all Dash docsets installed in the docsets directory.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	docsets, err := docset.Discover(paths.DataDir)
	if err != nil {
		return fmt.Errorf("failed to discover docsets: %w", err)
	}

	if len(docsets) == 0 {
		fmt.Println("No docsets installed.")
		fmt.Printf("\nDocsets directory: %s\n", paths.DataDir)
		fmt.Println("\nTo install docsets, run:")
		fmt.Println("  dsearch install <docset-name>")
		fmt.Println("\nTo see available docsets:")
		fmt.Println("  dsearch available")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tENTRIES\tPATH")
	fmt.Fprintln(w, "----\t-------\t-------\t----")

	for _, ds := range docsets {
		// Shorten path for display
		displayPath := ds.Path
		if home, err := os.UserHomeDir(); err == nil {
			displayPath = strings.Replace(ds.Path, home, "~", 1)
		}
		displayPath = filepath.Base(displayPath)

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			ds.Name,
			ds.Version,
			ds.EntryCount,
			displayPath,
		)
	}
	w.Flush()

	fmt.Printf("\n%d docset(s) installed in %s\n", len(docsets), paths.DataDir)
	return nil
}
