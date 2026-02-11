package docset

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icampana/dsearch/internal/crawler"
	_ "modernc.org/sqlite"
)

// Builder handles creation and updating of docsets
type Builder struct {
	config *CreateConfig
}

// NewBuilder creates a new docset builder
func NewBuilder(config *CreateConfig) *Builder {
	return &Builder{config: config}
}

// Create creates a new docset from scratch
func (b *Builder) Create(entries []ExtractedEntry, pages []crawler.Page) error {
	docsetPath := filepath.Join(b.config.DataDir, fmt.Sprintf("%s.docset", b.config.Name))

	// Create directory structure
	contentsPath := filepath.Join(docsetPath, "Contents")
	resourcesPath := filepath.Join(contentsPath, "Resources")
	documentsPath := filepath.Join(resourcesPath, "Documents")

	for _, dir := range []string{contentsPath, resourcesPath, documentsPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Create Info.plist
	if err := b.createInfoPlist(contentsPath); err != nil {
		return fmt.Errorf("creating Info.plist: %w", err)
	}

	// Copy icon if provided
	if b.config.IconPath != "" {
		if err := b.copyIcon(docsetPath); err != nil {
			return fmt.Errorf("copying icon: %w", err)
		}
	}

	// Create SQLite database
	dbPath := filepath.Join(resourcesPath, "docSet.dsidx")
	if err := b.createDatabase(dbPath); err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	// Convert entries to use local paths
	localEntries := b.convertToLocalPaths(entries)
	
	// Insert entries
	if err := b.insertEntries(dbPath, localEntries); err != nil {
		return fmt.Errorf("inserting entries: %w", err)
	}

	// Save markdown pages
	if err := b.savePages(documentsPath, pages); err != nil {
		return fmt.Errorf("saving pages: %w", err)
	}

	return nil
}

// Update performs a differential update on an existing docset
func (b *Builder) Update(entries []ExtractedEntry, pages []crawler.Page) error {
	docsetPath := filepath.Join(b.config.DataDir, fmt.Sprintf("%s.docset", b.config.Name))
	dbPath := filepath.Join(docsetPath, "Contents", "Resources", "docSet.dsidx")
	documentsPath := filepath.Join(docsetPath, "Contents", "Resources", "Documents")

	// Load existing entries
	existingEntries, err := b.loadExistingEntries(dbPath)
	if err != nil {
		return fmt.Errorf("loading existing entries: %w", err)
	}

	// Convert new entries to use local paths
	localEntries := b.convertToLocalPaths(entries)
	
	// Compute differences
	toInsert, toUpdate, toDelete := b.computeDiff(localEntries, existingEntries)

	fmt.Printf("  New entries: %d\n", len(toInsert))
	fmt.Printf("  Updated entries: %d\n", len(toUpdate))
	fmt.Printf("  Orphaned entries: %d\n", len(toDelete))

	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert new entries
	insertStmt, err := tx.Prepare("INSERT INTO searchIndex(name, type, path) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("preparing insert statement: %w", err)
	}
	defer insertStmt.Close()

	for _, entry := range toInsert {
		_, err := insertStmt.Exec(entry.Name, entry.Type, entry.Path)
		if err != nil {
			return fmt.Errorf("inserting entry %s: %w", entry.Name, err)
		}
	}

	// Update changed entries
	updateStmt, err := tx.Prepare("UPDATE searchIndex SET type = ?, path = ? WHERE name = ?")
	if err != nil {
		return fmt.Errorf("preparing update statement: %w", err)
	}
	defer updateStmt.Close()

	for _, entry := range toUpdate {
		_, err := updateStmt.Exec(entry.Type, entry.Path, entry.Name)
		if err != nil {
			return fmt.Errorf("updating entry %s: %w", entry.Name, err)
		}
	}

	// Delete orphaned entries
	deleteStmt, err := tx.Prepare("DELETE FROM searchIndex WHERE name = ?")
	if err != nil {
		return fmt.Errorf("preparing delete statement: %w", err)
	}
	defer deleteStmt.Close()

	for _, entry := range toDelete {
		_, err := deleteStmt.Exec(entry.Name)
		if err != nil {
			return fmt.Errorf("deleting entry %s: %w", entry.Name, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	// Update pages (save new, update changed)
	if err := b.savePages(documentsPath, pages); err != nil {
		return fmt.Errorf("updating pages: %w", err)
	}

	// Update version in Info.plist
	if err := b.updateVersion(docsetPath); err != nil {
		return fmt.Errorf("updating version: %w", err)
	}

	return nil
}

// createInfoPlist creates the Info.plist file
func (b *Builder) createInfoPlist(contentsPath string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>DocSetPlatformFamily</key>
	<string>%s</string>
	<key>isDashDocset</key>
	<true/>
	<key>dashIndexFilePath</key>
	<string>index.md</string>
</dict>
</plist>`, b.config.Name, b.config.Name, b.config.Name)

	plistPath := filepath.Join(contentsPath, "Info.plist")
	return os.WriteFile(plistPath, []byte(plist), 0644)
}

// copyIcon copies the icon file to the docset
func (b *Builder) copyIcon(docsetPath string) error {
	data, err := os.ReadFile(b.config.IconPath)
	if err != nil {
		return err
	}

	iconPath := filepath.Join(docsetPath, "icon.png")
	return os.WriteFile(iconPath, data, 0644)
}

// createDatabase creates the SQLite database with proper schema
func (b *Builder) createDatabase(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE searchIndex(
			id INTEGER PRIMARY KEY,
			name TEXT,
			type TEXT,
			path TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Create unique index
	_, err = db.Exec(`CREATE UNIQUE INDEX anchor ON searchIndex (name, type, path)`)
	if err != nil {
		return err
	}

	return nil
}

// insertEntries inserts entries into the database
func (b *Builder) insertEntries(dbPath string, entries []ExtractedEntry) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT OR IGNORE INTO searchIndex(name, type, path) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		_, err := stmt.Exec(entry.Name, entry.Type, entry.Path)
		if err != nil {
			return fmt.Errorf("inserting entry %s: %w", entry.Name, err)
		}
	}

	return nil
}

// savePages saves markdown pages to the Documents folder
func (b *Builder) savePages(documentsPath string, pages []crawler.Page) error {
	for _, page := range pages {
		// Generate filename from URL
		filename := urlToFilename(page.URL) + ".md"
		filePath := filepath.Join(documentsPath, filename)

		// Ensure parent directories exist
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(page.Markdown), 0644); err != nil {
			return fmt.Errorf("writing file %s: %w", filePath, err)
		}
	}

	return nil
}

// loadExistingEntries loads entries from an existing docset
func (b *Builder) loadExistingEntries(dbPath string) ([]ExtractedEntry, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, type, path FROM searchIndex")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ExtractedEntry
	for rows.Next() {
		var entry ExtractedEntry
		err := rows.Scan(&entry.ID, &entry.Name, &entry.Type, &entry.Path)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// computeDiff computes the differences between new and existing entries
func (b *Builder) computeDiff(newEntries, existingEntries []ExtractedEntry) (toInsert, toUpdate, toDelete []ExtractedEntry) {
	// Build maps for comparison
	existingMap := make(map[string]ExtractedEntry)
	for _, e := range existingEntries {
		key := fmt.Sprintf("%s:%s:%s", e.Name, e.Type, e.Path)
		existingMap[key] = e
	}

	newMap := make(map[string]ExtractedEntry)
	for _, e := range newEntries {
		key := fmt.Sprintf("%s:%s:%s", e.Name, e.Type, e.Path)
		newMap[key] = e
	}

	// Find entries to insert and update
	for key, newEntry := range newMap {
		if existingEntry, exists := existingMap[key]; !exists {
			// New entry
			toInsert = append(toInsert, newEntry)
		} else if existingEntry.Hash != newEntry.Hash {
			// Changed entry
			toUpdate = append(toUpdate, newEntry)
		}
	}

	// Find entries to delete (orphaned)
	for key, existingEntry := range existingMap {
		if _, exists := newMap[key]; !exists {
			toDelete = append(toDelete, existingEntry)
		}
	}

	return
}

// updateVersion updates the version in Info.plist
func (b *Builder) updateVersion(docsetPath string) error {
	// For simplicity, we'll just recreate Info.plist with the new version
	contentsPath := filepath.Join(docsetPath, "Contents")
	return b.createInfoPlist(contentsPath)
}

// convertToLocalPaths converts entry paths from URLs to local filenames
func (b *Builder) convertToLocalPaths(entries []ExtractedEntry) []ExtractedEntry {
	var localEntries []ExtractedEntry
	
	for _, entry := range entries {
		localEntry := entry
		localEntry.Path = urlToFilename(entry.Path) + ".md"
		localEntries = append(localEntries, localEntry)
	}
	
	return localEntries
}

// Helper functions

func urlToFilename(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Replace special characters
	url = strings.ReplaceAll(url, "/", "_")
	url = strings.ReplaceAll(url, "?", "_")
	url = strings.ReplaceAll(url, "&", "_")
	url = strings.ReplaceAll(url, "=", "_")

	// Limit length
	if len(url) > 200 {
		// Create hash of the full URL
		hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))
		url = url[:150] + "_" + hash[:8]
	}

	return url
}
