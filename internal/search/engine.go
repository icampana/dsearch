// Package search provides search functionality across DevDocs documentation.
package search

import (
	"fmt"
	"sort"

	"github.com/sahilm/fuzzy"

	"github.com/icampana/dsearch/internal/devdocs"
)

// Engine handles searching across multiple DevDocs indices.
type Engine struct {
	indices       []*devdocs.Index
	indicesBySlug map[string]*devdocs.Index // slug -> Index lookup
	slugsByIndex  map[*devdocs.Index]string // Index -> slug lookup (for reverse mapping)
	limit         int
}

// New creates a new search engine.
func New(indices []*devdocs.Index, indicesBySlug map[string]*devdocs.Index, limit int) *Engine {
	// Build reverse map for O(1) index-to-slug lookup
	slugsByIndex := make(map[*devdocs.Index]string, len(indicesBySlug))
	for slug, idx := range indicesBySlug {
		slugsByIndex[idx] = slug
	}

	return &Engine{
		indices:       indices,
		indicesBySlug: indicesBySlug,
		slugsByIndex:  slugsByIndex,
		limit:         limit,
	}
}

// Result represents a search result with fuzzy match score.
type Result struct {
	devdocs.Entry
	Slug  string  // Which doc this result is from
	Score float64 // Fuzzy match score (0-1)
}

// Search performs a search across all indices with fuzzy matching.
// If docSlugs is specified, only those docs are searched.
// Warns via returned warning string if searching across >10 docs without filtering.
func (e *Engine) Search(query string, docSlugs []string) ([]Result, string, error) {
	var results []Result
	var warning string

	// Filter indices by slug if specified
	indicesToSearch := e.indices
	if len(docSlugs) > 0 {
		indicesToSearch = make([]*devdocs.Index, 0)
		for _, slug := range docSlugs {
			if idx, ok := e.indicesBySlug[slug]; ok {
				indicesToSearch = append(indicesToSearch, idx)
			}
		}
	}

	if len(indicesToSearch) == 0 {
		return nil, "", fmt.Errorf("no matching docs found")
	}

	// Warn if searching across many docs without filtering
	if len(indicesToSearch) > 10 && len(docSlugs) == 0 {
		warning = fmt.Sprintf("Searching across %d docs. Use -d <doc> for faster results.", len(indicesToSearch))
	}

	// Collect all entries from all indices with their source slug
	type indexedEntry struct {
		entry devdocs.Entry
		slug  string
	}
	var allEntries []indexedEntry
	for _, idx := range indicesToSearch {
		// Direct O(1) lookup using reverse map
		slug := e.slugsByIndex[idx]
		for _, entry := range idx.Entries {
			allEntries = append(allEntries, indexedEntry{entry: entry, slug: slug})
		}
	}

	if len(allEntries) == 0 {
		return nil, "", fmt.Errorf("no results found for %q", query)
	}

	// Apply fuzzy matching to rank results
	names := make([]string, len(allEntries))
	for i, ie := range allEntries {
		names[i] = ie.entry.Name
	}

	matches := fuzzy.Find(query, names)

	// Build results with scores
	for _, match := range matches {
		ie := allEntries[match.Index]
		results = append(results, Result{
			Entry: ie.entry,
			Slug:  ie.slug,
			Score: float64(match.Score) / 100.0, // Normalize to 0-1
		})
	}

	// Sort by score (descending) then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Entry.Name < results[j].Entry.Name
	})

	// Limit results
	if len(results) > e.limit {
		results = results[:e.limit]
	}

	return results, warning, nil
}
