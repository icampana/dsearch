// Package render handles converting HTML documentation to text and markdown.
package render

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"golang.org/x/net/html"

	readability "codeberg.org/readeck/go-readability"
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
	// First extract main content using readability to remove navigation/cruft
	cleanContent, err := r.extractMainContent(htmlContent)
	if err != nil {
		// Fallback to original content if extraction fails
		log.Printf("Warning: readability extraction failed: %v", err)
		cleanContent = htmlContent
	}

	// Parse cleaned HTML
	doc, err := html.Parse(bytes.NewReader(cleanContent))
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	var buf strings.Builder
	r.extractText(doc, &buf)
	return buf.String(), nil
}

// renderMarkdown converts HTML to markdown.
func (r *Renderer) renderMarkdown(htmlContent []byte) (string, error) {
	// First extract main content using readability
	cleanContent, err := r.extractMainContent(htmlContent)
	if err != nil {
		// Fallback to original content if extraction fails
		log.Printf("Warning: readability extraction failed: %v", err)
		cleanContent = htmlContent
	}

	// Convert cleaned HTML to markdown using the specialized library
	md, err := htmltomarkdown.ConvertString(string(cleanContent))
	if err != nil {
		return "", fmt.Errorf("converting to markdown: %w", err)
	}

	// Clean up excess whitespace
	md = strings.TrimSpace(md)
	// Replace multiple consecutive newlines with at most 2
	md = strings.ReplaceAll(md, "\n\n\n", "\n\n")

	return md, nil
}

// extractMainContent uses readability to extract the main readable content.
// This removes navigation, sidebar, footer, ads, and other non-content elements.
func (r *Renderer) extractMainContent(htmlContent []byte) ([]byte, error) {
	// Parse the URL for readability (we don't have a real URL for docset files)
	baseURL, _ := url.Parse("http://localhost/docset")

	article, err := readability.FromReader(bytes.NewReader(htmlContent), baseURL)
	if err != nil {
		return nil, fmt.Errorf("readability extraction: %w", err)
	}

	return []byte(article.Content), nil
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
