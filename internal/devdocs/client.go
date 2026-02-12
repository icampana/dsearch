// Package devdocs provides types and client for interacting with the DevDocs API
package devdocs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout     = 60 * time.Second
	userAgent          = "github.com/icampana/dsearch"
	defaultManifestURL = "https://devdocs.io"
	defaultContentURL  = "https://documents.devdocs.io"
)

// Client is an HTTP client for fetching DevDocs data
type Client struct {
	manifestURL string
	contentURL  string
	httpClient  *http.Client
}

// NewClient creates a new DevDocs API client.
// If baseURL is non-empty, it overrides both manifest and content URLs (for testing).
// Otherwise, uses the default DevDocs URLs.
func NewClient(baseURL string) *Client {
	if baseURL != "" {
		return &Client{
			manifestURL: baseURL,
			contentURL:  baseURL,
			httpClient: &http.Client{
				Timeout: defaultTimeout,
			},
		}
	}
	return &Client{
		manifestURL: defaultManifestURL,
		contentURL:  defaultContentURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// FetchManifest fetches the docs.json manifest from DevDocs
// Returns a list of all available documentation
func (c *Client) FetchManifest() ([]Doc, error) {
	url := fmt.Sprintf("%s/docs.json", c.manifestURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest body: %w", err)
	}

	var docs []Doc
	if err := json.Unmarshal(body, &docs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return docs, nil
}

// FetchIndex fetches the index.json for a specific documentation slug
// Returns the search index containing entries and types
func (c *Client) FetchIndex(slug string) (*Index, error) {
	url := fmt.Sprintf("%s/%s/index.json", c.contentURL, slug)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index for %s: %w", slug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch index for %s failed with status %d", slug, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read index body: %w", err)
	}

	var index Index
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}

// FetchDB fetches the db.json for a specific documentation slug
// Returns a map of content paths to HTML strings
func (c *Client) FetchDB(slug string) (map[string]string, error) {
	url := fmt.Sprintf("%s/%s/db.json", c.contentURL, slug)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch db for %s: %w", slug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch db for %s failed with status %d", slug, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read db body: %w", err)
	}

	var db map[string]string
	if err := json.Unmarshal(body, &db); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db: %w", err)
	}

	return db, nil
}
