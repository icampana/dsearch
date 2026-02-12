// Package devdocs tests for HTTP client
package devdocs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchManifest(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/docs.json" {
			t.Errorf("Expected path /docs.json, got %s", r.URL.Path)
		}

		// Return mock docs.json
		mockManifest := []Doc{
			{
				Name:    "Angular",
				Slug:    "angular",
				Type:    "angular",
				Release: "21.0.3",
				Mtime:   1765113132,
				DBSize:  8830513,
			},
			{
				Name:    "React",
				Slug:    "react",
				Type:    "react",
				Release: "18.3.1",
				Mtime:   1765113200,
				DBSize:  5242880,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockManifest)
	}))
	defer ts.Close()

	// Test client
	client := NewClient(WithBaseURL(ts.URL))
	docs, err := client.FetchManifest()
	if err != nil {
		t.Fatalf("FetchManifest() error = %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 docs, got %d", len(docs))
	}
	if docs[0].Name != "Angular" {
		t.Errorf("First doc Name = %q, want Angular", docs[0].Name)
	}
	if docs[1].Name != "React" {
		t.Errorf("Second doc Name = %q, want React", docs[1].Name)
	}
}

func TestFetchIndex(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/react/index.json" {
			t.Errorf("Expected path /react/index.json, got %s", r.URL.Path)
		}

		mockIndex := Index{
			Entries: []Entry{
				{Name: "useState", Path: "reference/react/hooks/usestate", Type: "hooks"},
			},
			Types: []Type{
				{Name: "hooks", Count: 1, Slug: "hooks"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockIndex)
	}))
	defer ts.Close()

	client := NewClient(WithBaseURL(ts.URL))
	index, err := client.FetchIndex("react")
	if err != nil {
		t.Fatalf("FetchIndex() error = %v", err)
	}

	if len(index.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(index.Entries))
	}
	if index.Entries[0].Name != "useState" {
		t.Errorf("Entry Name = %q, want useState", index.Entries[0].Name)
	}
}

func TestFetchDB(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/react/db.json" {
			t.Errorf("Expected path /react/db.json, got %s", r.URL.Path)
		}

		// Return mock db.json (key-value map of path -> HTML content)
		mockDB := map[string]string{
			"index":                          "<h1>React Documentation</h1>",
			"reference/react/hooks/usestate": "<h1>useState</h1><p>State hook</p>",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockDB)
	}))
	defer ts.Close()

	client := NewClient(WithBaseURL(ts.URL))
	db, err := client.FetchDB("react")
	if err != nil {
		t.Fatalf("FetchDB() error = %v", err)
	}

	if len(db) != 2 {
		t.Errorf("Expected 2 entries in DB, got %d", len(db))
	}
	content, ok := db["reference/react/hooks/usestate"]
	if !ok {
		t.Fatal("Expected key 'reference/react/hooks/usestate' not found")
	}
	expectedContent := "<h1>useState</h1><p>State hook</p>"
	if content != expectedContent {
		t.Errorf("Content = %q, want %q", content, expectedContent)
	}
}

func TestFetchManifestHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient(WithBaseURL(ts.URL))
	_, err := client.FetchManifest()
	if err == nil {
		t.Error("Expected error for 500 response, got nil")
	}
}

func TestFetchIndexInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer ts.Close()

	client := NewClient(WithBaseURL(ts.URL))
	_, err := client.FetchIndex("react")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
