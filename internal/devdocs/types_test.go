// Package devdocs tests for DevDocs data types
package devdocs

import (
	"encoding/json"
	"testing"
)

func TestDocUnmarshal(t *testing.T) {
	// Actual DevDocs docs.json entry format
	jsonDoc := `{
		"name": "Angular",
		"slug": "angular",
		"type": "angular",
		"version": "",
		"release": "21.0.3",
		"mtime": 1765113132,
		"db_size": 8830513,
		"attribution": "Angular documentation is maintained by...",
		"alias": "ng"
	}`

	var doc Doc
	err := json.Unmarshal([]byte(jsonDoc), &doc)
	if err != nil {
		t.Fatalf("Failed to unmarshal Doc: %v", err)
	}

	expected := Doc{
		Name:        "Angular",
		Slug:        "angular",
		Type:        "angular",
		Version:     "",
		Release:     "21.0.3",
		Mtime:       1765113132,
		DBSize:      8830513,
		Attribution: "Angular documentation is maintained by...",
		Alias:       "ng",
	}

	if doc.Name != expected.Name {
		t.Errorf("Name = %q, want %q", doc.Name, expected.Name)
	}
	if doc.Slug != expected.Slug {
		t.Errorf("Slug = %q, want %q", doc.Slug, expected.Slug)
	}
	if doc.Release != expected.Release {
		t.Errorf("Release = %q, want %q", doc.Release, expected.Release)
	}
	if doc.Mtime != expected.Mtime {
		t.Errorf("Mtime = %d, want %d", doc.Mtime, expected.Mtime)
	}
	if doc.DBSize != expected.DBSize {
		t.Errorf("DBSize = %d, want %d", doc.DBSize, expected.DBSize)
	}
}

func TestIndexUnmarshal(t *testing.T) {
	// Actual DevDocs index.json format (simplified)
	jsonIndex := `{
		"entries": [
			{"name": "$localize", "path": "api/localize/init/$localize", "type": "localize"},
			{"name": "HttpClient", "path": "api/common/http", "type": "http"}
		],
		"types": [
			{"name": "animations", "count": 36, "slug": "animations"},
			{"name": "http", "count": 12, "slug": "http"}
		]
	}`

	var index Index
	err := json.Unmarshal([]byte(jsonIndex), &index)
	if err != nil {
		t.Fatalf("Failed to unmarshal Index: %v", err)
	}

	if len(index.Entries) != 2 {
		t.Errorf("Entries count = %d, want 2", len(index.Entries))
	}
	if index.Entries[0].Name != "$localize" {
		t.Errorf("First entry Name = %q, want $localize", index.Entries[0].Name)
	}
	if index.Entries[0].Path != "api/localize/init/$localize" {
		t.Errorf("First entry Path = %q, want api/localize/init/$localize", index.Entries[0].Path)
	}

	if len(index.Types) != 2 {
		t.Errorf("Types count = %d, want 2", len(index.Types))
	}
	if index.Types[0].Name != "animations" {
		t.Errorf("First type Name = %q, want animations", index.Types[0].Name)
	}
}

func TestEntryUnmarshal(t *testing.T) {
	jsonEntry := `{"name": "useState", "path": "reference/react/hooks/usestate", "type": "hooks"}`

	var entry Entry
	err := json.Unmarshal([]byte(jsonEntry), &entry)
	if err != nil {
		t.Fatalf("Failed to unmarshal Entry: %v", err)
	}

	if entry.Name != "useState" {
		t.Errorf("Name = %q, want useState", entry.Name)
	}
	if entry.Path != "reference/react/hooks/usestate" {
		t.Errorf("Path = %q, want reference/react/hooks/usestate", entry.Path)
	}
	if entry.Type != "hooks" {
		t.Errorf("Type = %q, want hooks", entry.Type)
	}
}

func TestTypeUnmarshal(t *testing.T) {
	jsonType := `{"name": "hooks", "count": 12, "slug": "hooks"}`

	var docType Type
	err := json.Unmarshal([]byte(jsonType), &docType)
	if err != nil {
		t.Fatalf("Failed to unmarshal Type: %v", err)
	}

	if docType.Name != "hooks" {
		t.Errorf("Name = %q, want hooks", docType.Name)
	}
	if docType.Count != 12 {
		t.Errorf("Count = %d, want 12", docType.Count)
	}
	if docType.Slug != "hooks" {
		t.Errorf("Slug = %q, want hooks", docType.Slug)
	}
}
