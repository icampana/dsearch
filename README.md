# dsearch

A fast, offline documentation search tool for your terminal.

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Search through [Dash](https://kapeli.com/dash)-compatible docsets with fuzzy matching, multiple output formats, and an interactive TUI.

## Features

- 🔍 **Fuzzy search** across all installed docsets
- 📚 **200+ docsets** available (React, Node.js, Python, Go, Rust, etc.)
- 📝 **Multiple output formats**: plain text or markdown (perfect for LLMs)
- 🎨 **Interactive TUI** with live preview (coming soon)
- ⚡ **Fast**: SQLite-based indexing with fuzzy matching
- 🔌 **Offline**: Works completely offline once docsets are installed

## Installation

### From source

```bash
git clone https://github.com/icampana/dsearch.git
cd dsearch
make install
```

### Binary releases

Coming soon! Download from [Releases](https://github.com/icampana/dsearch/releases).

## Quick Start

```bash
# 1. List available docsets
dsearch available

# 2. Install docsets you want
dsearch install React Node.js Go

# 3. Search documentation
dsearch useState

# Or search in specific docset
dsearch useState -d react

# Get markdown output for LLMs
dsearch useState --format md
```

## Usage

### Interactive mode (coming soon)

```bash
dsearch                # Opens interactive fuzzy finder
dsearch -d react       # Interactive mode filtered to React docset
```

### Direct search

```bash
dsearch useState                     # Search all docsets
dsearch useState -d react            # Search only React docset
dsearch useState --format md         # Output as markdown
dsearch useState --type Function     # Filter by entry type
dsearch useState -l 20               # Limit to 20 results
dsearch useState --list              # List results without content
```

### Docset management

```bash
dsearch list                         # Show installed docsets
dsearch available [query]             # Show downloadable docsets
dsearch install react node bash       # Install docsets
```

## Configuration

dsearch follows the [XDG Base Directory specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html):

- **Docsets**: `$XDG_DATA_HOME/dsearch/docsets/` (default: `~/.local/share/dsearch/docsets/`)
- **Cache**: `$XDG_CACHE_HOME/dsearch/` (default: `~/.cache/dsearch/`)
- **Config**: `$XDG_CONFIG_HOME/dsearch/config.yaml` (default: `~/.config/dsearch/config.yaml`)

## Using with existing Dash/Zeal docsets

If you already have docsets from Dash or Zeal, you can symlink them:

```bash
# macOS Dash
ln -s ~/Library/Application\ Support/Dash/DocSets/* ~/.local/share/dsearch/docsets/

# Zeal
ln -s ~/.local/share/Zeal/Zeal/docsets/* ~/.local/share/dsearch/docsets/
```

## Development

### Prerequisites

- Go 1.21 or later

### Build

```bash
# Build binary
make build

# Install to GOPATH/bin
make install

# Run without building
make run

# Run tests
make test

# Build for multiple platforms
make release
```

## Status

This project is under active development. See [STATUS.md](STATUS.md) for detailed progress.

### Completed ✅

- CLI framework with cobra
- Docset discovery and parsing
- SQLite index reading
- Fuzzy search with scoring
- Content rendering (text/markdown)
- Docset installation from Kapeli feeds

### In Progress 🚧

- Interactive TUI with bubbletea

### Planned 📋

- Docset update/remove commands
- Shell completions
- Better HTML cleaning
- Code syntax highlighting

## Contributing

Contributions welcome! Please read the contributing guidelines and submit pull requests.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Dash](https://kapeli.com/dash) for the docset format and feeds
- [Kapeli](https://github.com/Kapeli) for maintaining the feeds repository
- [dasht](https://github.com/sunaku/dasht) for original inspiration

## Alternative Tools

- [dasht](https://github.com/sunaku/dasht) - Shell-based docset searcher (unmaintained)
- [Zeal](https://zealdocs.org/) - Qt-based docset browser
- [DevDocs](https://devdocs.io/) - Web-based documentation browser
