// Package tui tests for the interactive UI model.
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/icampana/dsearch/internal/docset"
	"github.com/icampana/dsearch/internal/render"
	"github.com/icampana/dsearch/internal/search"
)

func TestNewModel(t *testing.T) {
	docsets := []docset.Docset{
		{Name: "Test Docset", Path: "/test", IndexPath: "/test.idx", DocsPath: "/test/docs"},
	}

	engine := search.New(docsets, "", 10)

	model := NewModel(engine, docsets)

	// Verify initial state
	if model.engine != engine {
		t.Error("Engine not set correctly")
	}

	if len(model.docsets) != len(docsets) {
		t.Error("Docsets not set correctly")
	}

	if model.showPreview != true {
		t.Error("Initial showPreview should be true")
	}

	if model.outputFormat != render.FormatText {
		t.Error("Initial output format should be text")
	}
}

func TestSetOutputFormat(t *testing.T) {
	docsets := []docset.Docset{
		{Name: "Test", Path: "/test", IndexPath: "/test.idx", DocsPath: "/test/docs"},
	}

	model := NewModel(search.New(docsets, "", 10), docsets)

	// Test setting to markdown
	model.SetOutputFormat(render.FormatMD)
	if model.outputFormat != render.FormatMD {
		t.Error("Output format not set to MD")
	}

	// Test setting to text
	model.SetOutputFormat(render.FormatText)
	if model.outputFormat != render.FormatText {
		t.Error("Output format not set to Text")
	}
}

func TestViewportScrolling(t *testing.T) {
	docsets := []docset.Docset{
		{Name: "Test", Path: "/test", IndexPath: "/test.idx", DocsPath: "/test/docs"},
	}

	model := NewModel(search.New(docsets, "", 10), docsets)
	model.previewText = strings.Repeat("Line\n", 50) // 50 lines
	model.showPreview = true

	// Simulate window resize to set viewport size
	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 50,
	}
	_, _ = model.Update(msg)

	// Test page down
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})

	// Test page up
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})

	// Test Ctrl+U (top)
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Test Ctrl+D (bottom)
	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})

	// If no panic, viewport scrolling works
}

func TestLoadContentCmd(t *testing.T) {
	docsets := []docset.Docset{
		{Name: "Test", Path: "/test", IndexPath: "/test.idx", DocsPath: "/test/docs"},
	}

	engine := search.New(docsets, "", 10)
	model := NewModel(engine, docsets)

	// Create a mock result
	result := search.Result{
		Entry: docset.Entry{
			Name:     "test",
			Type:     "Function",
			Path:     "test.html",
			FullPath: "/test/docs/test.html",
		},
		Score: 0.8,
	}

	// Load content command
	cmd := model.loadContentCmd(result)

	// The command is a function that returns a tea.Msg
	// We can't execute it directly in tests, but we can verify it compiles
	if cmd == nil {
		t.Error("loadContentCmd returned nil")
	}
}
