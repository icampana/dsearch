// Package search provides search functionality across docsets.
package search

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"

	"github.com/icampana/dsearch/internal/docset"
)

// Engine handles searching across multiple docsets.
type Engine struct {
	docsets   []docset.Docset
	entryType string
	limit     int
}

// New creates a new search engine.
func New(docsets []docset.Docset, entryType string, limit int) *Engine {
	return &Engine{
		docsets:   docsets,
		entryType: entryType,
		limit:     limit,
	}
}

// Result represents a search result with fuzzy match score.
type Result struct {
	docset.Entry
	Score float64 // Fuzzy match score (0-1)
}

// Search performs a search across all docsets with fuzzy matching.
func (e *Engine) Search(query string, docsetNames []string) ([]Result, error) {
	var results []Result

	// Filter docsets by name if specified
	docsetsToSearch := e.docsets
	if len(docsetNames) > 0 {
		docsetsToSearch = make([]docset.Docset, 0)
		nameSet := make(map[string]bool)
		for _, name := range docsetNames {
			nameSet[strings.ToLower(name)] = true
		}
		for _, ds := range e.docsets {
			if nameSet[strings.ToLower(ds.Name)] {
				docsetsToSearch = append(docsetsToSearch, ds)
			}
		}
	}

	if len(docsetsToSearch) == 0 {
		return nil, fmt.Errorf("no matching docsets found")
	}

	// Collect all entries from all docsets
	var allEntries []docset.Entry
	for _, ds := range docsetsToSearch {
		entries, err := ds.Search(query, e.entryType, e.limit)
		if err != nil {
			return nil, fmt.Errorf("searching %s: %w", ds.Name, err)
		}
		allEntries = append(allEntries, entries...)
	}

	if len(allEntries) == 0 {
		return nil, fmt.Errorf("no results found for %q", query)
	}

	// Apply fuzzy matching to rank results
	names := make([]string, len(allEntries))
	for i, entry := range allEntries {
		names[i] = entry.Name
	}

	matches := fuzzy.Find(query, names)

	// Build results with scores
	for _, match := range matches {
		results = append(results, Result{
			Entry: allEntries[match.Index],
			Score: float64(match.Score) / 100.0, // Normalize to 0-1
		})
	}

	// Sort by score (descending) then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Name < results[j].Name
	})

	// Limit results
	if len(results) > e.limit {
		results = results[:e.limit]
	}

	return results, nil
}
