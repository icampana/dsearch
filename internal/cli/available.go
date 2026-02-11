package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/icampana/dsearch/internal/manager"
)

var availableCmd = &cobra.Command{
	Use:   "available [query]",
	Short: "List available docsets for download",
	Long:  `Lists all available docsets from Kapeli feeds that can be installed.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAvailable,
}

func runAvailable(cmd *cobra.Command, args []string) error {
	feeds, err := manager.AvailableFeeds()
	if err != nil {
		return fmt.Errorf("fetching available docsets: %w", err)
	}

	if len(feeds) == 0 {
		fmt.Println("No docsets available.")
		return nil
	}

	// Filter by query if provided
	query := ""
	if len(args) > 0 {
		query = strings.ToLower(args[0])
	}

	fmt.Printf("Available docsets (%d total):\n\n", len(feeds))

	// Group by first letter for easier navigation
	currentLetter := rune(' ')

	count := 0
	for _, feed := range feeds {
		// Filter by query
		if query != "" && !strings.Contains(strings.ToLower(feed.Name), query) {
			continue
		}

		// Print letter header
		if len(feed.Name) > 0 {
			firstLetter := rune(strings.ToUpper(string(feed.Name[0]))[0])
			if firstLetter != currentLetter {
				currentLetter = firstLetter
				fmt.Printf("\n[%s]\n", string(firstLetter))
			}
		}

		fmt.Printf("  %s\n", feed.Name)
		count++

		// Show limited results if there's a query
		if query != "" && count > 50 {
			fmt.Println("\n... (showing first 50 matches)")
			break
		}
	}

	if query == "" {
		fmt.Printf("\nTo install a docset, run:\n")
		fmt.Println("  dsearch install <docset-name>")
	}
	return nil
}
