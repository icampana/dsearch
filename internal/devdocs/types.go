// Package devdocs provides types and client for interacting with the DevDocs API
package devdocs

// Doc represents a documentation entry from docs.json manifest
type Doc struct {
	Name        string `json:"name"`        // Display name (e.g., "Angular", "React")
	Slug        string `json:"slug"`        // URL slug (e.g., "angular", "react~18")
	Type        string `json:"type"`        // Doc type (e.g., "angular", "react")
	Version     string `json:"version"`     // Version string (may be empty)
	Release     string `json:"release"`     // Release version (e.g., "21.0.3", "18.3.1")
	Mtime       int64  `json:"mtime"`       // Modification time (unix timestamp)
	DBSize      int64  `json:"db_size"`     // Size of db.json in bytes
	Attribution string `json:"attribution"` // Attribution text (HTML)
	Alias       string `json:"alias"`       // Short alias (e.g., "ng" for Angular)
}

// Index represents the search index from index.json
type Index struct {
	Entries []Entry `json:"entries"` // Searchable entries
	Types   []Type  `json:"types"`   // Entry categories with counts
}

// Entry represents a searchable document entry
type Entry struct {
	Name string `json:"name"` // Entry name (e.g., "useState", "$localize")
	Path string `json:"path"` // Path to content (key in db.json)
	Type string `json:"type"` // Entry type/category
}

// Type represents a category of entries
type Type struct {
	Name  string `json:"name"`  // Category name
	Count int    `json:"count"` // Number of entries in this category
	Slug  string `json:"slug"`  // URL-safe category name
}
