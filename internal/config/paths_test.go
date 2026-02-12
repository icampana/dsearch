package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPaths(t *testing.T) {
	t.Parallel()

	// Save original env vars
	origDataHome := os.Getenv("XDG_DATA_HOME")
	origCacheHome := os.Getenv("XDG_CACHE_HOME")
	origConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origDataHome)
		os.Setenv("XDG_CACHE_HOME", origCacheHome)
		os.Setenv("XDG_CONFIG_HOME", origConfigHome)
	}()

	tests := []struct {
		name             string
		dataHome         string
		cacheHome        string
		configHome       string
		wantDataSuffix   string
		wantCacheSuffix  string
		wantConfigSuffix string
	}{
		{
			name:             "Default paths (no XDG env vars)",
			dataHome:         "",
			cacheHome:        "",
			configHome:       "",
			wantDataSuffix:   filepath.Join(".local", "share", "dsearch"),
			wantCacheSuffix:  filepath.Join(".cache", "dsearch"),
			wantConfigSuffix: filepath.Join(".config", "dsearch"),
		},
		{
			name:             "Custom XDG paths",
			dataHome:         "/custom/data",
			cacheHome:        "/custom/cache",
			configHome:       "/custom/config",
			wantDataSuffix:   filepath.Join("dsearch"),
			wantCacheSuffix:  filepath.Join("dsearch"),
			wantConfigSuffix: filepath.Join("dsearch"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			if tt.dataHome != "" {
				os.Setenv("XDG_DATA_HOME", tt.dataHome)
			} else {
				os.Unsetenv("XDG_DATA_HOME")
			}
			if tt.cacheHome != "" {
				os.Setenv("XDG_CACHE_HOME", tt.cacheHome)
			} else {
				os.Unsetenv("XDG_CACHE_HOME")
			}
			if tt.configHome != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.configHome)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}

			paths := DefaultPaths()

			// Verify DataDir
			if tt.dataHome != "" {
				if paths.DataDir != filepath.Join(tt.dataHome, tt.wantDataSuffix) {
					t.Errorf("DataDir = %v, want %v", paths.DataDir, filepath.Join(tt.dataHome, tt.wantDataSuffix))
				}
			} else {
				if !filepath.IsAbs(paths.DataDir) {
					t.Errorf("DataDir should be absolute, got %v", paths.DataDir)
				}
				if filepath.Base(filepath.Dir(paths.DataDir)) != "share" {
					t.Errorf("DataDir should be under .local/share, got %v", paths.DataDir)
				}
			}

			// Verify CacheDir
			if tt.cacheHome != "" {
				if paths.CacheDir != filepath.Join(tt.cacheHome, tt.wantCacheSuffix) {
					t.Errorf("CacheDir = %v, want %v", paths.CacheDir, filepath.Join(tt.cacheHome, tt.wantCacheSuffix))
				}
			} else {
				if !filepath.IsAbs(paths.CacheDir) {
					t.Errorf("CacheDir should be absolute, got %v", paths.CacheDir)
				}
			}

			// Verify ConfigDir
			if tt.configHome != "" {
				if paths.ConfigDir != filepath.Join(tt.configHome, tt.wantConfigSuffix) {
					t.Errorf("ConfigDir = %v, want %v", paths.ConfigDir, filepath.Join(tt.configHome, tt.wantConfigSuffix))
				}
			} else {
				if !filepath.IsAbs(paths.ConfigDir) {
					t.Errorf("ConfigDir should be absolute, got %v", paths.ConfigDir)
				}
			}
		})
	}
}

func TestPaths_EnsureDirs(t *testing.T) {
	t.Parallel()

	// Create temp dir for testing
	tmpDir := t.TempDir()

	paths := Paths{
		DataDir:   filepath.Join(tmpDir, "data"),
		CacheDir:  filepath.Join(tmpDir, "cache"),
		ConfigDir: filepath.Join(tmpDir, "config"),
	}

	// Ensure directories are created
	if err := paths.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}

	// Verify directories exist
	for _, dir := range []string{paths.DataDir, paths.CacheDir, paths.ConfigDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory %s should exist, got error: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("%s should be a directory", dir)
		}
	}

	// Calling EnsureDirs again should be idempotent
	if err := paths.EnsureDirs(); err != nil {
		t.Errorf("EnsureDirs() should be idempotent, got error: %v", err)
	}
}
