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
	schemaType string // "standard" or "coredata"
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

	// Detect schema type
	if err := ds.detectSchema(); err != nil {
		// Non-fatal, just log
		fmt.Fprintf(os.Stderr, "Warning: could not detect schema for %s: %v\n", ds.Name, err)
	}

	// Count entries in the index
	if err := ds.countEntries(); err != nil {
		// Non-fatal, just set to 0
		fmt.Fprintf(os.Stderr, "Warning: counting entries for %s: %v\n", ds.Name, err)
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

// detectSchema determines which database schema is being used.
func (ds *Docset) detectSchema() error {
	if _, err := os.Stat(ds.IndexPath); os.IsNotExist(err) {
		ds.schemaType = "none"
		return nil
	}

	db, err := sql.Open("sqlite", ds.IndexPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Check for standard Dash schema (searchIndex table)
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='searchIndex' LIMIT 1").Scan(&tableName)
	if err == nil && tableName == "searchIndex" {
		ds.schemaType = "standard"
		return nil
	}

	// Check for Core Data schema (ZNODE table)
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='ZNODE' LIMIT 1").Scan(&tableName)
	if err == nil && tableName == "ZNODE" {
		ds.schemaType = "coredata"
		return nil
	}

	ds.schemaType = "unknown"
	return nil
}

// countEntries counts the number of entries in the docset's SQLite index.
func (ds *Docset) countEntries() error {
	if _, err := os.Stat(ds.IndexPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", ds.IndexPath)
	}

	switch ds.schemaType {
	case "standard":
		return ds.countStandardEntries()
	case "coredata":
		return ds.countCoreDataEntries()
	default:
		// Try standard first, then fallback
		if err := ds.countStandardEntries(); err == nil {
			ds.schemaType = "standard"
			return nil
		}
		return ds.countCoreDataEntries()
	}
}

// countStandardEntries counts entries from a standard Dash schema database.
func (ds *Docset) countStandardEntries() error {
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

// countCoreDataEntries counts entries from a Core Data schema database.
func (ds *Docset) countCoreDataEntries() error {
	db, err := sql.Open("sqlite", ds.IndexPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// For Core Data schema, try to count searchable nodes
	err = db.QueryRow("SELECT COUNT(*) FROM ZNODE WHERE ZKISSEARCHABLE=1").Scan(&ds.EntryCount)
	if err != nil {
		return fmt.Errorf("counting entries: %w", err)
	}

	// If the count is very low, the index might be incomplete
	// Fall back to file system scan
	if ds.EntryCount < 10 {
		fmt.Fprintf(os.Stderr, "Warning: %s has only %d indexed entries. Scanning file system...\n", ds.Name, ds.EntryCount)
		return ds.countFilesFallback()
	}

	return nil
}

// countFilesFallback counts HTML files as a last resort.
func (ds *Docset) countFilesFallback() error {
	count := 0
	filepath.Walk(ds.DocsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".html") {
			count++
		}
		return nil
	})
	ds.EntryCount = count
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
	switch ds.schemaType {
	case "standard":
		return ds.searchStandard(pattern, entryType, limit)
	case "coredata":
		entries, err := ds.searchCoreData(pattern, entryType, limit)
		if err != nil || len(entries) > 0 {
			return entries, err
		}
		// Fall back to file system search if core data search is empty
		fmt.Fprintf(os.Stderr, "Warning: Core data search returned no results. Falling back to file system...\n")
		return ds.searchFiles(pattern, limit)
	default:
		// Try standard first
		entries, err := ds.searchStandard(pattern, entryType, limit)
		if err == nil && len(entries) > 0 {
			return entries, nil
		}
		// Try core data
		entries, err = ds.searchCoreData(pattern, entryType, limit)
		if err == nil && len(entries) > 0 {
			ds.schemaType = "coredata"
			return entries, nil
		}
		// Fall back to file system
		return ds.searchFiles(pattern, limit)
	}
}

// searchStandard queries a standard Dash schema database.
func (ds *Docset) searchStandard(pattern string, entryType string, limit int) ([]Entry, error) {
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

// searchCoreData queries a Core Data schema database.
func (ds *Docset) searchCoreData(pattern string, entryType string, limit int) ([]Entry, error) {
	db, err := sql.Open("sqlite", ds.IndexPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Query the ZNODE table for searchable nodes
	query := "SELECT Z_PK, ZKNAME, ZKDOCUMENTTYPE FROM ZNODE WHERE ZKISSEARCHABLE=1 AND ZKNAME LIKE ?"
	args := []any{"%" + pattern + "%"}

	query += " ORDER BY ZKNAME LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying index: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var docType int
		if err := rows.Scan(&e.ID, &e.Name, &docType); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		e.Docset = ds.Name
		e.Type = mapDocType(docType)
		e.Path = fmt.Sprintf("node-%d", e.ID)

		// Try to find the associated file path
		// This is a simplified approach; some Core Data schemas have complex relationships
		e.FullPath = filepath.Join(ds.DocsPath, "index.html")
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return entries, nil
}

// mapDocType converts numeric document types to strings.
func mapDocType(docType int) string {
	types := map[int]string{
		0: "Guide",
		1: "Package",
		2: "Topic",
		3: "Class",
		4: "Method",
		5: "Property",
		6: "Function",
	}
	if t, ok := types[docType]; ok {
		return t
	}
	return "Unknown"
}

// searchFiles scans the file system for matching HTML files.
func (ds *Docset) searchFiles(pattern string, limit int) ([]Entry, error) {
	var entries []Entry
	patternLower := strings.ToLower(pattern)
	count := 0

	filepath.Walk(ds.DocsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}

		// Extract filename from path
		filename := strings.ToLower(filepath.Base(path))
		filename = strings.TrimSuffix(filename, ".html")

		// Match against pattern
		if strings.Contains(filename, patternLower) {
			relativePath, _ := filepath.Rel(ds.DocsPath, path)
			entries = append(entries, Entry{
				ID:       int64(count),
				Name:     strings.TrimSuffix(filepath.Base(path), ".html"),
				Type:     "File",
				Path:     relativePath,
				Docset:   ds.Name,
				FullPath: path,
			})
			count++
		}

		// Stop if we've reached the limit
		if count >= limit {
			return filepath.SkipDir
		}

		return nil
	})

	return entries, nil
}

// GetContent reads the HTML content for an entry.
func (ds *Docset) GetContent(entry Entry) ([]byte, error) {
	// Check if file exists first
	if _, err := os.Stat(entry.FullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("documentation file not found: %s", entry.FullPath)
	}
	return os.ReadFile(entry.FullPath)
}
