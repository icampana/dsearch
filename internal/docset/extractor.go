package docset

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"

	"github.com/icampana/dsearch/internal/crawler"
)

// ExtractedEntry represents a searchable entry during docset creation
type ExtractedEntry struct {
	Entry
	Content string
	Hash    string // Content hash for differential updates
}

// Extractor handles extraction of searchable entries from documentation pages
type Extractor struct {
	typeMap map[string]string
}

// NewExtractor creates a new entry extractor with the given type mappings
func NewExtractor(typeMap map[string]string) *Extractor {
	return &Extractor{
		typeMap: typeMap,
	}
}

// ExtractFromPages extracts searchable entries from crawled pages
func (e *Extractor) ExtractFromPages(pages []crawler.Page) []ExtractedEntry {
	var entries []ExtractedEntry

	for _, page := range pages {
		pageEntries := e.extractFromPage(page)
		entries = append(entries, pageEntries...)
	}

	return entries
}

// extractFromPage extracts entries from a single page
func (e *Extractor) extractFromPage(page crawler.Page) []ExtractedEntry {
	var entries []ExtractedEntry
	lines := strings.Split(page.Markdown, "\n")

	for i, line := range lines {
		// Try to extract entry from this line
		entry := e.extractEntry(line, page.URL, i, lines)
		if entry != nil {
			// Compute content hash
			content := e.extractContent(lines, i)
			entry.Hash = computeHash(content)
			entry.Content = content
			entries = append(entries, *entry)
		}
	}

	return entries
}

// extractEntry attempts to extract a searchable entry from a line
func (e *Extractor) extractEntry(line, url string, lineNum int, allLines []string) *ExtractedEntry {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Check custom type mappings first (highest priority)
	for pattern, entryType := range e.typeMap {
		if matchesPattern(trimmed, pattern) {
			name := extractName(trimmed, pattern)
			return &ExtractedEntry{
				Entry: Entry{
					Name: name,
					Type: entryType,
					Path: url,
				},
			}
		}
	}

	// Detect heading-based entries
	if heading := detectHeading(trimmed); heading != nil {
		entryType := e.inferTypeFromHeading(heading, lineNum, allLines)
		return &ExtractedEntry{
			Entry: Entry{
				Name: heading.Text,
				Type: entryType,
				Path: url,
			},
		}
	}

	// Detect code pattern entries
	if codeEntry := detectCodePattern(trimmed); codeEntry != nil {
		return &ExtractedEntry{
			Entry: Entry{
				Name: codeEntry.Name,
				Type: codeEntry.Type,
				Path: url,
			},
		}
	}

	return nil
}

// Heading represents a markdown heading
type Heading struct {
	Level int
	Text  string
}

// detectHeading detects if a line is a markdown heading
func detectHeading(line string) *Heading {
	// Match ATX headings: # Heading
	if match := regexp.MustCompile(`^(#{1,6})\s+(.+)$`).FindStringSubmatch(line); match != nil {
		return &Heading{
			Level: len(match[1]),
			Text:  strings.TrimSpace(match[2]),
		}
	}

	// Match Setext headings (only H1 and H2)
	// H1: Heading
	//     ===
	// H2: Heading
	//     ---

	return nil
}

// inferTypeFromHeading infers the entry type based on heading context
func (e *Extractor) inferTypeFromHeading(heading *Heading, lineNum int, allLines []string) string {
	// Check parent heading for context
	parentContext := ""
	for i := lineNum - 1; i >= 0; i-- {
		if parent := detectHeading(allLines[i]); parent != nil && parent.Level < heading.Level {
			parentContext = strings.ToLower(parent.Text)
			break
		}
	}

	// Apply heuristics based on heading level and context
	switch heading.Level {
	case 1:
		// H1 is usually package name or main guide
		if strings.Contains(parentContext, "api") || strings.Contains(parentContext, "reference") {
			return "Guide"
		}
		return "Package"

	case 2:
		// H2: Could be Function, Class, or Module
		text := strings.ToLower(heading.Text)

		// Check for class indicators
		if strings.HasPrefix(text, "class ") || isCapitalized(heading.Text) {
			return "Class"
		}

		// Check for function indicators
		if strings.Contains(text, "()") || strings.Contains(parentContext, "function") {
			return "Function"
		}

		// Default for H2
		if strings.Contains(parentContext, "api") {
			return "Function"
		}
		return "Guide"

	case 3:
		// H3: Usually Method, Property, or Section
		text := strings.ToLower(heading.Text)

		// Check for method indicators
		if strings.Contains(text, "()") || strings.Contains(text, ".") {
			return "Method"
		}

		// Check parent context
		if strings.Contains(parentContext, "class") {
			return "Property"
		}
		if strings.Contains(parentContext, "function") {
			return "Parameter"
		}

		return "Section"

	case 4, 5, 6:
		// Deeper headings are usually sections or parameters
		return "Section"
	}

	return "Guide"
}

// CodeEntry represents an entry detected from code patterns
type CodeEntry struct {
	Name string
	Type string
}

// detectCodePattern detects entries from inline code patterns
func detectCodePattern(line string) *CodeEntry {
	// Look for inline code: `something()`
	codePattern := regexp.MustCompile("`([^`]+)`")
	matches := codePattern.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		code := strings.TrimSpace(match[1])

		// Function pattern: name() or name(args)
		if strings.HasSuffix(code, "()") || regexp.MustCompile(`\w+\([^)]*\)$`).MatchString(code) {
			name := strings.TrimSuffix(code, "()")
			if idx := strings.Index(name, "("); idx != -1 {
				name = name[:idx]
			}
			return &CodeEntry{Name: name, Type: "Function"}
		}

		// Method pattern: Class.method()
		if strings.Contains(code, ".") && strings.HasSuffix(code, "()") {
			return &CodeEntry{Name: code, Type: "Method"}
		}

		// Constant pattern: ALL_CAPS
		if regexp.MustCompile(`^[A-Z][A-Z_0-9]+$`).MatchString(code) {
			return &CodeEntry{Name: code, Type: "Constant"}
		}
	}

	return nil
}

// extractContent extracts the content following an entry for hashing
func (e *Extractor) extractContent(lines []string, startLine int) string {
	var content []string

	// Get the entry line
	if startLine < len(lines) {
		content = append(content, lines[startLine])
	}

	// Get content until next same-level or higher heading
	if startLine+1 < len(lines) {
		currentLevel := 6
		if h := detectHeading(lines[startLine]); h != nil {
			currentLevel = h.Level
		}

		for i := startLine + 1; i < len(lines); i++ {
			line := lines[i]

			// Stop at same or higher level heading
			if h := detectHeading(line); h != nil && h.Level <= currentLevel {
				break
			}

			content = append(content, line)

			// Limit content size
			if len(content) > 50 {
				break
			}
		}
	}

	return strings.Join(content, "\n")
}

// Helper functions

func matchesPattern(line, pattern string) bool {
	// Support simple glob patterns and regex
	if strings.HasPrefix(pattern, "H") && len(pattern) == 2 {
		// Heading level pattern: H1, H2, etc.
		level := int(pattern[1] - '0')
		if level >= 1 && level <= 6 {
			h := detectHeading(line)
			return h != nil && h.Level == level
		}
	}

	// Code pattern: `regex`
	if strings.HasPrefix(pattern, "`") && strings.HasSuffix(pattern, "`") {
		inner := pattern[1 : len(pattern)-1]
		matched, _ := regexp.MatchString(inner, line)
		return matched
	}

	// Simple substring match
	return strings.Contains(line, pattern)
}

func extractName(line, pattern string) string {
	// Extract name based on pattern type
	if strings.HasPrefix(pattern, "H") && len(pattern) == 2 {
		// Extract from heading
		if h := detectHeading(line); h != nil {
			return h.Text
		}
	}

	// Default: return trimmed line
	return strings.TrimSpace(line)
}

func isCapitalized(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Check if first character is uppercase
	first := s[0]
	return first >= 'A' && first <= 'Z'
}

func computeHash(content string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(content)))
}
