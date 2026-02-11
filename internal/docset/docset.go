// Package docset provides functionality for discovering and reading Dash docsets.
package docset

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // Register sqlite driver
)

// Docset represents a Dash documentation set.
type Docset struct {
	Name       string // Display name (from Info.plist or folder name)
	Identifier string // Bundle identifier
	Version    string // Docset version
	Path       string // Full path to .docset folder
	IndexPath  string // Path to docSet.dsidx SQLite database
	DocsPath   string // Path to Documents folder
	EntryCount int    // Number of entries in the index
}

// Discover finds all docsets in the given directory.
func Discover(baseDir string) ([]Docset, error) {
	var docsets []Docset

	// Check if directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return docsets, nil // Return empty list if dir doesn't exist
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".docset") {
			continue
		}

		docsetPath := filepath.Join(baseDir, entry.Name())
		ds, err := Load(docsetPath)
		if err != nil {
			// Log warning but continue with other docsets
			fmt.Fprintf(os.Stderr, "Warning: skipping %s: %v\n", entry.Name(), err)
			continue
		}

		docsets = append(docsets, ds)
	}

	return docsets, nil
}

// Load reads a docset from the given path.
func Load(path string) (Docset, error) {
	ds := Docset{
		Path:      path,
		IndexPath: filepath.Join(path, "Contents", "Resources", "docSet.dsidx"),
		DocsPath:  filepath.Join(path, "Contents", "Resources", "Documents"),
	}

	// Parse Info.plist
	plistPath := filepath.Join(path, "Contents", "Info.plist")
	if err := ds.parseInfoPlist(plistPath); err != nil {
		// Use folder name as fallback
		ds.Name = strings.TrimSuffix(filepath.Base(path), ".docset")
	}

	// Count entries in the index
	if err := ds.countEntries(); err != nil {
		// Non-fatal, just set to 0
		ds.EntryCount = 0
	}

	return ds, nil
}

// parseInfoPlist reads the Info.plist file and extracts metadata.
func (ds *Docset) parseInfoPlist(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse as simple key-value pairs from plist
	// The plist format is: <key>name</key><string>value</string>
	content := string(data)

	ds.Name = extractPlistValue(content, "CFBundleName")
	if ds.Name == "" {
		ds.Name = extractPlistValue(content, "DocSetPlatformFamily")
	}
	if ds.Name == "" {
		ds.Name = strings.TrimSuffix(filepath.Base(ds.Path), ".docset")
	}

	ds.Identifier = extractPlistValue(content, "CFBundleIdentifier")
	ds.Version = extractPlistValue(content, "CFBundleVersion")
	if ds.Version == "" {
		ds.Version = extractPlistValue(content, "CFBundleShortVersionString")
	}
	if ds.Version == "" {
		ds.Version = "-"
	}

	return nil
}

// extractPlistValue extracts a string value from a plist XML string.
func extractPlistValue(content, key string) string {
	keyTag := "<key>" + key + "</key>"
	idx := strings.Index(content, keyTag)
	if idx == -1 {
		return ""
	}

	// Find the next <string> tag after the key
	rest := content[idx+len(keyTag):]
	startTag := "<string>"
	endTag := "</string>"

	startIdx := strings.Index(rest, startTag)
	if startIdx == -1 {
		return ""
	}

	rest = rest[startIdx+len(startTag):]
	endIdx := strings.Index(rest, endTag)
	if endIdx == -1 {
		return ""
	}

	return strings.TrimSpace(rest[:endIdx])
}

// countEntries counts the number of entries in the docset's SQLite index.
func (ds *Docset) countEntries() error {
	if _, err := os.Stat(ds.IndexPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", ds.IndexPath)
	}

	db, err := sql.Open("sqlite", ds.IndexPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM searchIndex").Scan(&ds.EntryCount)
	if err != nil {
		return fmt.Errorf("counting entries: %w", err)
	}

	return nil
}

// Entry represents a single documentation entry from the index.
type Entry struct {
	ID       int64
	Name     string
	Type     string // Function, Class, Method, Property, etc.
	Path     string // Relative path to HTML file (may include anchor)
	Docset   string // Name of the docset this entry belongs to
	FullPath string // Full path to the HTML file
}

// Search queries the docset index for entries matching the pattern.
func (ds *Docset) Search(pattern string, entryType string, limit int) ([]Entry, error) {
	db, err := sql.Open("sqlite", ds.IndexPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Build query with LIKE for basic matching
	query := "SELECT id, name, type, path FROM searchIndex WHERE name LIKE ?"
	args := []any{"%" + pattern + "%"}

	if entryType != "" {
		query += " AND type = ?"
		args = append(args, entryType)
	}

	query += " ORDER BY name LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying index: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Path); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		e.Docset = ds.Name
		e.FullPath = filepath.Join(ds.DocsPath, strings.Split(e.Path, "#")[0])
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return entries, nil
}

// GetContent reads the HTML content for an entry.
func (ds *Docset) GetContent(entry Entry) ([]byte, error) {
	return os.ReadFile(entry.FullPath)
}
