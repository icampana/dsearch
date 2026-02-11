// Package devdocs provides types and client for interacting with the DevDocs API
package devdocs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Meta represents local metadata for an installed doc
type Meta struct {
	Slug      string    `json:"slug"`
	Mtime     int64     `json:"mtime"`
	Installed time.Time `json:"installed"`
	DBSize    int64     `json:"db_size"`
}

// Store handles downloading and storing DevDocs documentation
type Store struct {
	rootDir string
}

// NewStore creates a new Store rooted at the given directory
func NewStore(rootDir string) *Store {
	return &Store{
		rootDir: rootDir,
	}
}

// Install downloads and installs a documentation set
// Returns the local metadata for the installed doc
func (s *Store) Install(slug string, index *Index, db map[string]string, manifest []Doc) (*Meta, error) {
	// Find doc in manifest to get mtime and db_size
	var docInfo *Doc
	for i := range manifest {
		if manifest[i].Slug == slug {
			docInfo = &manifest[i]
			break
		}
	}
	if docInfo == nil {
		return nil, fmt.Errorf("doc %s not found in manifest", slug)
	}

	// Create doc directory
	docDir := filepath.Join(s.rootDir, "docs", slug)
	if err := os.MkdirAll(docDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create doc directory: %w", err)
	}

	// Save index.json
	indexPath := filepath.Join(docDir, "index.json")
	if err := writeJSON(indexPath, index); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	// Create content directory and split db.json into individual files
	contentDir := filepath.Join(docDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create content directory: %w", err)
	}

	for path, content := range db {
		// Ensure path is safe (no directory traversal)
		if filepath.Clean(path) != path {
			continue
		}
		contentFile := filepath.Join(contentDir, path+".html")
		contentDirPath := filepath.Dir(contentFile)
		if err := os.MkdirAll(contentDirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create content subdir: %w", err)
		}
		if err := os.WriteFile(contentFile, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write content file: %w", err)
		}
	}

	// Create and save meta.json
	meta := &Meta{
		Slug:      slug,
		Mtime:     docInfo.Mtime,
		Installed: time.Now(),
		DBSize:    docInfo.DBSize,
	}
	metaPath := filepath.Join(docDir, "meta.json")
	if err := writeJSON(metaPath, meta); err != nil {
		return nil, fmt.Errorf("failed to save meta: %w", err)
	}

	return meta, nil
}

// LoadIndex loads the search index for an installed doc
func (s *Store) LoadIndex(slug string) (*Index, error) {
	indexPath := filepath.Join(s.rootDir, "docs", slug, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}

// LoadContent loads HTML content for a specific path in an installed doc
func (s *Store) LoadContent(slug, path string) (string, error) {
	contentPath := filepath.Join(s.rootDir, "docs", slug, "content", path+".html")
	data, err := os.ReadFile(contentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	return string(data), nil
}

// IsInstalled checks if a doc is installed
func (s *Store) IsInstalled(slug string) bool {
	docDir := filepath.Join(s.rootDir, "docs", slug)
	info, err := os.Stat(docDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ListInstalled returns a list of all installed doc slugs
func (s *Store) ListInstalled() []string {
	docsDir := filepath.Join(s.rootDir, "docs")
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return nil
	}

	var slugs []string
	for _, entry := range entries {
		if entry.IsDir() {
			slugs = append(slugs, entry.Name())
		}
	}

	return slugs
}

// SaveManifest saves the DevDocs manifest to cache
func (s *Store) SaveManifest(manifest []Doc) error {
	cacheDir := filepath.Join(s.rootDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	manifestPath := filepath.Join(cacheDir, "manifest.json")
	return writeJSON(manifestPath, manifest)
}

// LoadManifest loads the cached DevDocs manifest
func (s *Store) LoadManifest() ([]Doc, error) {
	manifestPath := filepath.Join(s.rootDir, "cache", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest []Doc
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return manifest, nil
}

// Uninstall removes an installed doc
func (s *Store) Uninstall(slug string) error {
	docDir := filepath.Join(s.rootDir, "docs", slug)
	return os.RemoveAll(docDir)
}

// writeJSON is a helper to write JSON to a file
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
