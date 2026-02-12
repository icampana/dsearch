package search

import (
	"testing"

	"github.com/icampana/dsearch/internal/devdocs"
)

func TestEngine_Search(t *testing.T) {
	// Setup test data
	index1 := &devdocs.Index{
		Entries: []devdocs.Entry{
			{Name: "useState", Path: "react/hooks", Type: "Hook"},
			{Name: "useEffect", Path: "react/hooks", Type: "Hook"},
		},
	}
	index2 := &devdocs.Index{
		Entries: []devdocs.Entry{
			{Name: "User", Path: "models/user", Type: "Model"},
		},
	}

	indices := []*devdocs.Index{index1, index2}
	indicesBySlug := map[string]*devdocs.Index{
		"react":  index1,
		"django": index2,
	}

	engine := New(indices, indicesBySlug, 10)

	tests := []struct {
		name      string
		query     string
		slugs     []string
		wantCount int
		wantName  string
		wantErr   bool
	}{
		{
			name:      "Exact match in all docs",
			query:     "useState",
			slugs:     nil,
			wantCount: 1,
			wantName:  "useState",
			wantErr:   false,
		},
		{
			name:      "Fuzzy match",
			query:     "useSt",
			slugs:     nil,
			wantCount: 1,
			wantName:  "useState",
			wantErr:   false,
		},
		{
			name:      "Filter by specific slug",
			query:     "useState",
			slugs:     []string{"react"},
			wantCount: 1,
			wantName:  "useState",
			wantErr:   false,
		},
		{
			name:      "Filter by wrong slug",
			query:     "useState",
			slugs:     []string{"django"},
			wantCount: 0,
			wantName:  "",
			wantErr:   true, // Should return error if no results found
		},
		{
			name:      "No results",
			query:     "nonexistent",
			slugs:     nil,
			wantCount: 0,
			wantName:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, warning, err := engine.Search(tt.query, tt.slugs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Engine.Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(results) != tt.wantCount {
					t.Errorf("Engine.Search() got %d results, want %d", len(results), tt.wantCount)
				}
				if len(results) > 0 && results[0].Name != tt.wantName {
					t.Errorf("Engine.Search() first result = %s, want %s", results[0].Name, tt.wantName)
				}
				// Verify warning logic (simple check)
				if len(indices) > 10 && len(tt.slugs) == 0 && warning == "" {
					t.Errorf("Engine.Search() expected warning for many docs")
				}
			}
		})
	}
}

func TestEngine_Limit(t *testing.T) {
	// Create larger index
	entries := make([]devdocs.Entry, 20)
	for i := 0; i < 20; i++ {
		entries[i] = devdocs.Entry{Name: "Test", Path: "test", Type: "Type"}
	}
	index := &devdocs.Index{Entries: entries}
	indices := []*devdocs.Index{index}
	indicesBySlug := map[string]*devdocs.Index{"test": index}

	// Limit to 5
	engine := New(indices, indicesBySlug, 5)
	results, _, err := engine.Search("Test", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}
}
