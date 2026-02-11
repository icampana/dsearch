// Package devdocs tests for documentation store (download/install logic)
package devdocs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallNewDoc(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Mock data
	mockIndex := &Index{
		Entries: []Entry{
			{Name: "testEntry", Path: "test/path", Type: "test"},
		},
		Types: []Type{{Name: "testType", Count: 1, Slug: "test"}},
	}
	mockDB := map[string]string{
		"test/path": "<h1>Test Content</h1>",
	}

	// Mock manifest
	mockManifest := []Doc{
		{Name: "Test", Slug: "test", Mtime: 12345, DBSize: 100},
	}

	store := NewStore(tmpDir)

	// Install
	meta, err := store.Install("test", mockIndex, mockDB, mockManifest)
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	// Verify meta
	if meta.Slug != "test" {
		t.Errorf("Meta Slug = %q, want test", meta.Slug)
	}
	if meta.Mtime != 12345 {
		t.Errorf("Meta Mtime = %d, want 12345", meta.Mtime)
	}

	// Verify index.json saved
	indexPath := filepath.Join(tmpDir, "docs", "test", "index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("index.json not found at %s", indexPath)
	}

	// Verify content file created
	contentPath := filepath.Join(tmpDir, "docs", "test", "content", "test/path.html")
	if _, err := os.Stat(contentPath); os.IsNotExist(err) {
		t.Errorf("Content file not found at %s", contentPath)
	}

	// Verify meta.json saved
	metaPath := filepath.Join(tmpDir, "docs", "test", "meta.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Errorf("meta.json not found at %s", metaPath)
	}
}

func TestLoadIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock installed doc
	docsDir := filepath.Join(tmpDir, "docs", "test")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write index.json
	indexPath := filepath.Join(docsDir, "index.json")
	mockIndex := Index{
		Entries: []Entry{{Name: "testEntry", Path: "test/path", Type: "test"}},
		Types:   []Type{{Name: "testType", Count: 1, Slug: "test"}},
	}
	if err := writeJSON(indexPath, mockIndex); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tmpDir)

	// Load index
	index, err := store.LoadIndex("test")
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	if len(index.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(index.Entries))
	}
	if index.Entries[0].Name != "testEntry" {
		t.Errorf("Entry Name = %q, want testEntry", index.Entries[0].Name)
	}
}

func TestLoadContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock content file
	contentDir := filepath.Join(tmpDir, "docs", "test", "content")
	contentSubDir := filepath.Join(contentDir, "test")
	if err := os.MkdirAll(contentSubDir, 0755); err != nil {
		t.Fatal(err)
	}

	contentPath := filepath.Join(contentSubDir, "path.html")
	expectedContent := "<h1>Test Content</h1>"
	if err := os.WriteFile(contentPath, []byte(expectedContent), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tmpDir)

	// Load content
	content, err := store.LoadContent("test", "test/path")
	if err != nil {
		t.Fatalf("LoadContent() error = %v", err)
	}

	if content != expectedContent {
		t.Errorf("Content = %q, want %q", content, expectedContent)
	}
}

func TestIsInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock installed doc
	docsDir := filepath.Join(tmpDir, "docs", "test")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tmpDir)

	// Test installed
	if !store.IsInstalled("test") {
		t.Error("Expected test to be installed")
	}

	// Test not installed
	if store.IsInstalled("not-installed") {
		t.Error("Expected not-installed to not be installed")
	}
}

func TestListInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock installed docs
	docsDir := filepath.Join(tmpDir, "docs")
	for _, slug := range []string{"test1", "test2", "test3"} {
		if err := os.MkdirAll(filepath.Join(docsDir, slug), 0755); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStore(tmpDir)

	// List
	installed := store.ListInstalled()
	if len(installed) != 3 {
		t.Errorf("Expected 3 installed docs, got %d", len(installed))
	}
}

func TestSaveAndLoadManifest(t *testing.T) {
	tmpDir := t.TempDir()

	mockManifest := []Doc{
		{Name: "Test", Slug: "test", Mtime: 12345},
	}

	store := NewStore(tmpDir)

	// Save
	err := store.SaveManifest(mockManifest)
	if err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}

	// Verify file exists
	cacheDir := filepath.Join(tmpDir, "cache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("Cache directory not created at %s", cacheDir)
	}

	// Load
	loaded, err := store.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("Expected 1 doc in manifest, got %d", len(loaded))
	}
	if loaded[0].Name != "Test" {
		t.Errorf("Doc Name = %q, want Test", loaded[0].Name)
	}
}

func TestLoadManifest_NotExists(t *testing.T) {
	tmpDir := t.TempDir()

	store := NewStore(tmpDir)

	_, err := store.LoadManifest()
	if err == nil {
		t.Error("Expected error when manifest doesn't exist, got nil")
	}
}

func TestUninstall(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock installed doc
	docsDir := filepath.Join(tmpDir, "docs", "test")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	store := NewStore(tmpDir)

	// Uninstall
	err := store.Uninstall("test")
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}

	// Verify removed
	if store.IsInstalled("test") {
		t.Error("Expected test to be uninstalled")
	}
}
