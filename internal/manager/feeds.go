// Package manager handles downloading and managing docsets.
package manager

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// Feed represents a Kapeli docset feed.
type Feed struct {
	XMLName  xml.Name `xml:"entry"`
	Name     string   `xml:"name"`
	URL      string   `xml:"url"`
	Version  string   `xml:"version"`
	OtherURL string   `xml:"otherVersionsURL,omitempty"`
}

// AvailableFeeds fetches and returns all available docset feeds.
// It uses the GitHub API to get the list of feed files.
func AvailableFeeds() ([]Feed, error) {
	// Get Kapeli feeds directory listing from GitHub API
	resp, err := http.Get("https://api.github.com/repos/Kapeli/feeds/contents")
	if err != nil {
		return nil, fmt.Errorf("fetching feeds directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Parse JSON to get feed names
	type GitHubFile struct {
		Name string `json:"name"`
	}
	var files []GitHubFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	// Filter .xml files and build feed list
	var feeds []Feed
	for _, file := range files {
		if !strings.HasSuffix(file.Name, ".xml") {
			continue
		}
		name := strings.TrimSuffix(file.Name, ".xml")
		feeds = append(feeds, Feed{
			Name: name,
			URL:  fmt.Sprintf("https://raw.githubusercontent.com/Kapeli/feeds/master/%s", file.Name),
		})
	}

	return feeds, nil
}

// Install downloads and installs a docset from its feed.
func Install(name, destDir string) error {
	// Find the feed
	feeds, err := AvailableFeeds()
	if err != nil {
		return err
	}

	var feed *Feed
	for i := range feeds {
		if strings.EqualFold(feeds[i].Name, name) {
			feed = &feeds[i]
			break
		}
	}

	if feed == nil {
		return fmt.Errorf("docset %q not found", name)
	}

	// Fetch the feed to get the download URL
	downloadURL, err := getFeedDownloadURL(feed.URL)
	if err != nil {
		return fmt.Errorf("fetching feed details: %w", err)
	}

	// Check if already installed
	docsetPath := filepath.Join(destDir, name+".docset")
	if _, err := os.Stat(docsetPath); err == nil {
		return fmt.Errorf("docset already installed at %s", docsetPath)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Download the docset to a temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, name+".tgz")
	defer os.Remove(tempFile)

	fmt.Printf("Downloading %s from %s\n", name, downloadURL)

	// Download with progress bar
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create progress bar
	totalSize := resp.ContentLength
	bar := progressbar.NewOptions64(
		totalSize,
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprintln(os.Stderr)
		}),
	)

	// Create the temporary file
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	// Download and write to file with progress
	multiWriter := io.MultiWriter(out, bar)
	if _, err := io.Copy(multiWriter, resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	// Extract the archive
	fmt.Printf("Extracting to %s\n", destDir)

	if err := extractArchive(tempFile, destDir); err != nil {
		// Clean up partial extraction on error
		os.RemoveAll(docsetPath)
		return fmt.Errorf("extracting: %w", err)
	}

	fmt.Printf("Successfully installed %s\n", name)
	return nil
}

// extractArchive extracts a .tgz archive to the destination directory.
func extractArchive(archivePath, destDir string) error {
	// Open the gzip reader
	gzFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer gzFile.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract each file
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("reading tar header: %w", err)
		}

		// Skip directories (they'll be created automatically)
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Create the target path
		targetPath := filepath.Join(destDir, header.Name)

		// Create directory structure if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Create the file (convert tar mode to os.FileMode)
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return fmt.Errorf("creating file: %w", err)
		}

		// Copy file content
		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return fmt.Errorf("copying file: %w", err)
		}
		outFile.Close()
	}

	return nil
}

// getFeedDownloadURL fetches the feed XML and extracts the download URL.
func getFeedDownloadURL(feedURL string) (string, error) {
	resp, err := http.Get(feedURL)
	if err != nil {
		return "", fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return "", fmt.Errorf("parsing feed XML: %w", err)
	}

	return feed.URL, nil
}
