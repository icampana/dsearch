// Package render handles converting HTML documentation to text and markdown.
package render

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// Format represents the output format.
type Format string

const (
	FormatText Format = "text"
	FormatMD   Format = "md"
)

// Renderer converts HTML to the specified format.
type Renderer struct {
	format Format
}

// New creates a new renderer.
func New(format Format) *Renderer {
	return &Renderer{format: format}
}

// Render converts HTML to the configured format.
func (r *Renderer) Render(htmlContent []byte) (string, error) {
	switch r.format {
	case FormatMD:
		return r.renderMarkdown(htmlContent)
	case FormatText:
		return r.renderText(htmlContent)
	default:
		return r.renderText(htmlContent)
	}
}

// renderText converts HTML to plain text.
func (r *Renderer) renderText(htmlContent []byte) (string, error) {
	// Clean the HTML first
	cleanContent := cleanHTML(htmlContent)

	doc, err := html.Parse(bytes.NewReader(cleanContent))
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	var buf strings.Builder
	r.extractText(doc, &buf)
	return buf.String(), nil
}

// extractText recursively extracts text from HTML nodes.
func (r *Renderer) extractText(n *html.Node, buf *strings.Builder) {
	if n == nil {
		return
	}

	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			buf.WriteString(text)
			buf.WriteString(" ")
		}

	case html.ElementNode:
		// Handle block elements
		switch n.Data {
		case "p", "br", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li":
			buf.WriteString("\n")
		case "pre", "code":
			// Keep whitespace for code blocks
			buf.WriteString("\n```\n")
		case "a":
			// Extract href for links
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					buf.WriteString("[")
					break
				}
			}
		}

		// Recursively process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.extractText(c, buf)
		}

		switch n.Data {
		case "a":
			// Close link reference
			buf.WriteString("]")
		case "pre", "code":
			buf.WriteString("\n```\n")
		}

	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.extractText(c, buf)
		}
	}
}

// renderMarkdown converts HTML to markdown.
func (r *Renderer) renderMarkdown(htmlContent []byte) (string, error) {
	// Clean the HTML first
	cleanContent := cleanHTML(htmlContent)

	doc, err := html.Parse(bytes.NewReader(cleanContent))
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	var buf strings.Builder
	r.extractMarkdown(doc, &buf)
	return buf.String(), nil
}

// extractMarkdown recursively extracts markdown from HTML nodes.
func (r *Renderer) extractMarkdown(n *html.Node, buf *strings.Builder) {
	if n == nil {
		return
	}

	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			buf.WriteString(text)
		}

	case html.ElementNode:
		// Handle block elements with markdown formatting
		switch n.Data {
		case "p":
			buf.WriteString("\n\n")
		case "br":
			buf.WriteString("\n")
		case "h1":
			buf.WriteString("\n# ")
		case "h2":
			buf.WriteString("\n## ")
		case "h3":
			buf.WriteString("\n### ")
		case "h4":
			buf.WriteString("\n#### ")
		case "code":
			buf.WriteString("`")
		case "pre":
			buf.WriteString("\n```\n")
		case "strong", "b":
			buf.WriteString("**")
		case "em", "i":
			buf.WriteString("*")
		case "a":
			// Extract href for links
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					buf.WriteString("[")
					break
				}
			}
		case "ul", "ol":
			buf.WriteString("\n")
		case "li":
			buf.WriteString("- ")
		}

		// Recursively process children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.extractMarkdown(c, buf)
		}

		// Close formatting
		switch n.Data {
		case "a":
			// Close link
			buf.WriteString("]")
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					buf.WriteString("(" + attr.Val + ")")
					break
				}
			}
		case "strong", "b":
			buf.WriteString("**")
		case "em", "i":
			buf.WriteString("*")
		case "code":
			buf.WriteString("`")
		case "pre":
			buf.WriteString("\n```\n")
		case "h1", "h2", "h3", "h4", "h5", "h6":
			buf.WriteString("\n")
		}

	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			r.extractMarkdown(c, buf)
		}
	}
}

// cleanHTML removes specific navigation/cruft elements from HTML.
func cleanHTML(htmlContent []byte) []byte {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return htmlContent // Return original if parsing fails
	}

	var buf bytes.Buffer

	// Elements to completely remove
	removeTags := map[string]bool{
		"script":   true,
		"style":    true,
		"link":     true,
		"meta":     true,
		"noscript": true,
		"iframe":   true,
		"head":     true, // Remove entire head section
	}

	// ID patterns that indicate navigation elements (case-insensitive substring match)
	navIDs := []string{
		"search",
		"nav",
		"menu",
		"sidebar",
		"top",
		"header",
		"footer",
		"breadcrumbs",
		"skip",
		"related",
		"social",
	}

	// Href patterns that indicate navigation links
	navHrefs := []string{
		"/learn/index.html",
		"/community/index.html",
		"/blog",
		"facebook.com/react",
		"twitter.com",
		"github.com",
	}

	var shouldRemoveNode func(*html.Node) bool
	shouldRemoveNode = func(n *html.Node) bool {
		if n == nil {
			return false
		}

		// Remove by tag name
		if n.Type == html.ElementNode && removeTags[n.Data] {
			return true
		}

		// Remove by ID patterns
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "id" {
					idVal := strings.ToLower(attr.Val)
					for _, pattern := range navIDs {
						if strings.Contains(idVal, pattern) {
							return true
						}
					}
				}
				// Remove navigation links by href
				if attr.Key == "href" {
					hrefVal := strings.ToLower(attr.Val)
					for _, pattern := range navHrefs {
						if strings.Contains(hrefVal, pattern) {
							return true
						}
					}
				}
				// Also remove links with certain class patterns
				if attr.Key == "class" {
					classVal := strings.ToLower(attr.Val)
					if strings.Contains(classVal, "skip") ||
						strings.Contains(classVal, "nav") ||
						strings.Contains(classVal, "menu") ||
						strings.Contains(classVal, "sidebar") {
						return true
					}
				}
			}
		}

		return false
	}

	// Traverses the tree and removes unwanted nodes
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n == nil {
			return
		}

		// Skip nodes that should be removed
		if shouldRemoveNode(n) {
			return
		}

		// Render this node
		html.Render(&buf, n)

		// Continue to children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	// Start traversal from document root
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		traverse(c)
	}

	return buf.Bytes()
}

// CleanContent is a no-op - cleaning happens during parsing.
// Kept for backwards compatibility.
func CleanContent(htmlContent []byte) []byte {
	return htmlContent
}
