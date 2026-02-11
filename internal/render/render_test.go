// Package render tests for markdown conversion with readability extraction.
package render

import (
	"strings"
	"testing"
)

func TestRenderMarkdownWithReadability(t *testing.T) {
	tests := []struct {
		name           string
		htmlInput      string
		containsOutput []string // strings that should be in output
		notContains    []string // strings that should NOT be in output
	}{
		{
			name: "basic HTML to markdown",
			htmlInput: `<html><body><h1>Test Title</h1>
<p>This is a <strong>test</strong> paragraph.</p></body></html>`,
			containsOutput: []string{
				"# Test Title",
				"**test**",
				"paragraph",
			},
		},
		{
			name: "removes CSS from style tags",
			htmlInput: `<html><head><style>
body { color: red; }
.navigation { display: none; }
</style></head><body><h1>Title</h1></body></html>`,
			containsOutput: []string{"# Title"},
			notContains:    []string{"color: red", "display: none", "body {", "navigation"},
		},
		{
			name: "removes scripts",
			htmlInput: `<html><body>
<h1>Title</h1>
<script>console.log("test");</script>
<p>Content</p>
</body></html>`,
			containsOutput: []string{"# Title", "Content"},
			notContains:    []string{"console.log", "script", "<script"},
		},
		{
			name: "handles code blocks",
			htmlInput: `<html><body>
<h1>Code Example</h1>
<pre><code>function test() {
  return true;
}</code></pre>
</body></html>`,
			containsOutput: []string{
				"# Code Example",
				"```",
				"function test()",
				"return true;",
			},
		},
		{
			name: "removes navigation elements",
			htmlInput: `<html><body>
<nav id="main-nav">
<a href="/home">Home</a>
<a href="/about">About</a>
</nav>
<main>
<h1>Main Content</h1>
<p>This is the main article content.</p>
</main>
<footer>Copyright 2024</footer>
</body></html>`,
			containsOutput: []string{"# Main Content", "article content"},
			notContains:    []string{"Home", "About", "Copyright", "footer"},
		},
		{
			name: "handles links correctly",
			htmlInput: `<html><body>
<p>Check out <a href="https://example.com">this link</a> for more info.</p>
</body></html>`,
			containsOutput: []string{"[this link]", "(https://example.com)"},
		},
		{
			name: "preserves lists",
			htmlInput: `<html><body>
<h1>Todo List</h1>
<ul>
<li>First item</li>
<li>Second item</li>
<li>Third item</li>
</ul>
</body></html>`,
			containsOutput: []string{
				"# Todo List",
				"- First item",
				"- Second item",
				"- Third item",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := New(FormatMD)
			result, err := renderer.Render([]byte(tt.htmlInput))
			if err != nil {
				t.Fatalf("Renderer.Render() error = %v", err)
			}

			// Check expected content
			for _, expected := range tt.containsOutput {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, result)
				}
			}

			// Check unwanted content
			for _, unwanted := range tt.notContains {
				if strings.Contains(result, unwanted) {
					t.Errorf("Output should NOT contain %q, but it does. Got:\n%s", unwanted, result)
				}
			}
		})
	}
}

func TestRenderTextMode(t *testing.T) {
	htmlInput := `<html><head><style>body { color: red; }</style></head><body>
<h1>Title</h1>
<p>This is <strong>bold</strong> text.</p>
</body></html>`

	renderer := New(FormatText)
	result, err := renderer.Render([]byte(htmlInput))
	if err != nil {
		t.Fatalf("Renderer.Render() error = %v", err)
	}

	// Text mode should not contain HTML
	if strings.Contains(result, "<h1>") || strings.Contains(result, "<strong>") {
		t.Errorf("Text mode should not contain HTML tags, got: %s", result)
	}

	// Text mode should not contain CSS
	if strings.Contains(result, "color: red") {
		t.Errorf("Text mode should not contain CSS, got: %s", result)
	}

	// Text mode should have the content
	if !strings.Contains(result, "Title") || !strings.Contains(result, "bold") || !strings.Contains(result, "text") {
		t.Errorf("Text mode should have content, got: %s", result)
	}
}
