// Package config provides XDG-compliant configuration and path management.
package config

import (
	"os"
	"path/filepath"
)

// Paths holds the XDG-compliant directory paths for dsearch.
type Paths struct {
	DataDir   string // Where docsets are stored
	CacheDir  string // For downloads and temporary files
	ConfigDir string // For configuration files
}

// DefaultPaths returns XDG-compliant paths for dsearch.
// Falls back to ~/.local/share, ~/.cache, and ~/.config if XDG vars are unset.
func DefaultPaths() Paths {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(home, ".cache")
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(home, ".config")
	}

	return Paths{
		DataDir:   filepath.Join(dataDir, "dsearch", "docsets"),
		CacheDir:  filepath.Join(cacheDir, "dsearch"),
		ConfigDir: filepath.Join(configDir, "dsearch"),
	}
}

// EnsureDirs creates all necessary directories if they don't exist.
func (p Paths) EnsureDirs() error {
	for _, dir := range []string{p.DataDir, p.CacheDir, p.ConfigDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// DocsetDir returns the path where a specific docset would be stored.
func (p Paths) DocsetDir(name string) string {
	return filepath.Join(p.DataDir, name+".docset")
}
