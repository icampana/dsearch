// Package config provides XDG-compliant configuration and path management.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// MigrateDataDir moves docs from the old double-nested path to the correct path.
// Old: ~/.local/share/dsearch/docs/docs/<slug>
// New: ~/.local/share/dsearch/docs/<slug>
//
// This migration is idempotent and safe to run multiple times.
func MigrateDataDir(dataDir string) error {
	oldPath := filepath.Join(dataDir, "docs", "docs")

	// Check if old path exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		// Already migrated or fresh install
		return nil
	}

	// List all directories in the old path
	entries, err := os.ReadDir(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old docs directory: %w", err)
	}

	migratedCount := 0
	var migrationErrors []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		slug := entry.Name()
		oldSlugPath := filepath.Join(oldPath, slug)
		newSlugPath := filepath.Join(dataDir, "docs", slug)

		// Check if destination already exists (shouldn't happen, but be safe)
		if _, err := os.Stat(newSlugPath); err == nil {
			fmt.Fprintf(os.Stderr, "Migration: skipping %s (already exists at new location)\n", slug)
			continue
		}

		// Move the directory
		if err := os.Rename(oldSlugPath, newSlugPath); err != nil {
			migrationErrors = append(migrationErrors, fmt.Errorf("failed to migrate %s: %w", slug, err))
			continue
		}

		migratedCount++
	}

	// Remove the now-empty old directory
	if migratedCount > 0 {
		if err := os.Remove(oldPath); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Migration: could not remove old directory: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Migration: successfully migrated %d documentation set(s)\n", migratedCount)
		}
	}

	// Return aggregated errors if any
	if len(migrationErrors) > 0 {
		return fmt.Errorf("migration completed with %d error(s): %v", len(migrationErrors), migrationErrors[0])
	}

	return nil
}
