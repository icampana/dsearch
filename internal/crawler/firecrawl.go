package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Page represents a scraped documentation page
type Page struct {
	URL      string
	Markdown string
	Title    string
	Links    []string
}

// Crawler handles Firecrawl operations
type Crawler struct {
	startURL string
	include  []string
	exclude  []string
	depth    int
	selector string
	tempDir  string
}

// New creates a new Crawler instance
func New(startURL string, include, exclude []string, depth int, selector string) (*Crawler, error) {
	// Create temp directory for Firecrawl outputs
	tempDir, err := os.MkdirTemp("", "dsearch-crawl-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp directory: %w", err)
	}

	return &Crawler{
		startURL: startURL,
		include:  include,
		exclude:  exclude,
		depth:    depth,
		selector: selector,
		tempDir:  tempDir,
	}, nil
}

// Cleanup removes temporary files
func (c *Crawler) Cleanup() error {
	return os.RemoveAll(c.tempDir)
}

// MapURLs discovers all URLs on the documentation site
func (c *Crawler) MapURLs() ([]string, error) {
	outputFile := filepath.Join(c.tempDir, "urls.json")

	// Build Firecrawl map command
	args := []string{"map", c.startURL, "--json", "-o", outputFile}

	// Add search pattern if include patterns specified
	if len(c.include) > 0 {
		// Use the first include pattern as search filter
		args = append(args, "--search", c.include[0])
	}

	// Execute Firecrawl map
	cmd := exec.Command("firecrawl", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("firecrawl map failed: %w\nOutput: %s", err, string(output))
	}

	// Parse output
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("reading map output: %w", err)
	}

	// Firecrawl map returns JSON with links array
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Links []struct {
				URL string `json:"url"`
			} `json:"links"`
		} `json:"data"`
	}

	var urls []string
	if err := json.Unmarshal(data, &result); err == nil && result.Success {
		for _, link := range result.Data.Links {
			urls = append(urls, link.URL)
		}
	} else {
		// Fallback: try to extract URLs from text
		urls = extractURLs(string(data))
	}

	// Filter URLs
	urls = c.filterURLs(urls)

	// Limit by depth (approximate by path segments)
	urls = c.limitByDepth(urls)

	return urls, nil
}

// ScrapePages downloads and scrapes pages in parallel (max 2 concurrent)
func (c *Crawler) ScrapePages(urls []string) ([]Page, error) {
	var (
		pages []Page
		mu    sync.Mutex
		wg    sync.WaitGroup
	)

	// Semaphore to limit concurrent scrapes to 2
	semaphore := make(chan struct{}, 2)

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			page, err := c.scrapePage(u)
			if err != nil {
				// Log error but continue with other pages
				fmt.Fprintf(os.Stderr, "Warning: failed to scrape %s: %v\n", u, err)
				return
			}

			mu.Lock()
			pages = append(pages, page)
			mu.Unlock()
		}(url)
	}

	wg.Wait()

	return pages, nil
}

// scrapePage scrapes a single page using Firecrawl
func (c *Crawler) scrapePage(url string) (Page, error) {
	// Generate safe filename from URL
	filename := sanitizeFilename(url) + ".json"
	outputFile := filepath.Join(c.tempDir, filename)

	// Build Firecrawl scrape command
	args := []string{"scrape", url, "--format", "markdown,links", "-o", outputFile}

	if c.selector != "" {
		args = append(args, "--include-tags", c.selector)
	}

	// Always get main content only for cleaner docs
	args = append(args, "--only-main-content")

	// Execute Firecrawl scrape
	cmd := exec.Command("firecrawl", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return Page{}, fmt.Errorf("firecrawl scrape failed: %w\nOutput: %s", err, string(output))
	}

	// Parse output
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return Page{}, fmt.Errorf("reading scrape output: %w", err)
	}

	// Parse Firecrawl JSON output
	// Try wrapped format first (with data field)
	var wrappedResult struct {
		Data struct {
			Markdown string `json:"markdown"`
			Title    string `json:"title"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &wrappedResult); err == nil && wrappedResult.Data.Markdown != "" {
		return Page{
			URL:      url,
			Markdown: wrappedResult.Data.Markdown,
			Title:    wrappedResult.Data.Title,
		}, nil
	}

	// Try direct format
	var directResult struct {
		Markdown string `json:"markdown"`
		Title    string `json:"title"`
	}

	if err := json.Unmarshal(data, &directResult); err != nil {
		return Page{}, fmt.Errorf("parsing scrape output: %w", err)
	}

	return Page{
		URL:      url,
		Markdown: directResult.Markdown,
		Title:    directResult.Title,
	}, nil
}

// filterURLs applies include/exclude patterns
func (c *Crawler) filterURLs(urls []string) []string {
	var filtered []string

	for _, url := range urls {
		// Check exclude patterns first
		if matchesAny(url, c.exclude) {
			continue
		}

		// Check include patterns (if any specified)
		if len(c.include) > 0 && !matchesAny(url, c.include) {
			continue
		}

		filtered = append(filtered, url)
	}

	return filtered
}

// limitByDepth limits URLs based on path depth
func (c *Crawler) limitByDepth(urls []string) []string {
	if c.depth <= 0 {
		return urls
	}

	// Parse start URL to get base path
	startPath := extractPath(c.startURL)
	startSegments := len(strings.Split(startPath, "/"))

	var limited []string
	for _, url := range urls {
		path := extractPath(url)
		segments := len(strings.Split(path, "/"))

		// Allow URLs within depth limit
		if segments-startSegments <= c.depth {
			limited = append(limited, url)
		}
	}

	return limited
}

// Helper functions

func sanitizeFilename(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Replace special characters
	url = strings.ReplaceAll(url, "/", "_")
	url = strings.ReplaceAll(url, "?", "_")
	url = strings.ReplaceAll(url, "&", "_")
	url = strings.ReplaceAll(url, "=", "_")

	// Limit length
	if len(url) > 100 {
		url = url[:100]
	}

	return url
}

func extractURLs(text string) []string {
	// Simple regex to extract URLs from text
	re := regexp.MustCompile(`https?://[^\s\"<>]+`)
	return re.FindAllString(text, -1)
}

func extractPath(url string) string {
	// Remove protocol
	if idx := strings.Index(url, "://"); idx != -1 {
		url = url[idx+3:]
	}

	// Remove domain, keep only path
	if idx := strings.Index(url, "/"); idx != -1 {
		return url[idx:]
	}

	return "/"
}

func matchesAny(s string, patterns []string) bool {
	for _, pattern := range patterns {
		// Support glob-style patterns
		if matched, _ := filepath.Match(pattern, s); matched {
			return true
		}
		// Also check simple substring match
		if strings.Contains(s, pattern) {
			return true
		}
	}
	return false
}
