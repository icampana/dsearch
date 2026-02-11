# DSearch Project Context

## Project Overview

**DSearch** is a fast, offline documentation search tool built in Go. It searches through Dash-compatible docsets and displays results in your terminal or via an interactive TUI.

## Tech Stack

- **Language**: Go 1.25.7
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **TUI Framework**: Bubble Tea (v1.3.10) with Bubbles components
- **Database**: SQLite via modernc.org/sqlite
- **Fuzzy Search**: github.com/sahilm/fuzzy
- **Styling**: Lipgloss (github.com/charmbracelet/lipgloss)

## Architecture

```
dsearch/
├── cmd/dsearch/         # Main application entry point
├── internal/
│   ├── cli/            # CLI commands (root, install, list, available, version)
│   ├── config/         # Configuration and path management (XDG spec)
│   ├── docset/         # Docset discovery, loading, and SQLite access
│   ├── manager/        # Feed management for docsets
│   ├── render/         # HTML to text/markdown conversion
│   ├── search/         # Search engine with fuzzy matching
│   └── tui/            # Interactive terminal user interface
└── Makefile            # Build and development commands
```

## Key Components

### search.Engine
- Core search functionality across multiple docsets
- Fuzzy matching with scoring (0-1)
- Supports filtering by docset name and entry type
- Returns sorted results with metadata

### docset.Docset
- Represents a Dash documentation set
- Discovers .docset bundles from directory
- Queries SQLite index (docSet.dsidx) for entries
- Provides access to HTML content files

### tui.Model
- Interactive TUI using Bubble Tea framework
- Features:
  - Search input with debouncing (300ms)
  - Scrollable results list using bubbles/list
  - Content preview panel (toggleable with Tab)
  - Keyboard navigation (arrow keys, Enter, Ctrl+C)
  - Handles window resize events

### cli.Commands
- `dsearch [query]` - Direct search or launch TUI
- `dsearch list` - List installed docsets
- `dsearch available` - List available docsets to install
- `dsearch install <docset>` - Install a docset
- `dsearch version` - Show version info

## Design Patterns

- **Hexagonal-ish**: Clean separation between domain logic (search, docset) and interfaces (CLI, TUI)
- **Interface-based**: Accept interfaces, return structs (e.g., search.Engine works with docset.Docset)
- **Error Wrapping**: Errors wrapped with context using `fmt.Errorf("...: %w", err)`
- **Resource Management**: Database connections deferred close, file operations with proper error checking

## Code Conventions

- **Naming**: Exported names start with capital letter, short receivers (1-2 letters)
- **Comments**: Godoc comments on all exported functions/types
- **Error Handling**: Always check errors, never ignore them
- **Formatting**: Standard Go formatting via `gofmt`
- **Testing**: Use table-driven tests with `t.Parallel()`
- **Imports**: Grouped as stdlib, blank line, third-party

## Development Commands

### Build & Run
```bash
make build        # Build binary to bin/dsearch
make run          # Run without building
make dev          # Build and run with ARGS variable
```

### Testing
```bash
make test         # Run all tests
make test-cover   # Run tests with coverage
```

### Code Quality
```bash
make fmt          # Format code
make lint         # Run golangci-lint
```

### Deployment
```bash
make install      # Install to GOPATH/bin
make release      # Build for multiple platforms
```

## XDG Directories

- Data: `~/.local/share/dsearch/` (docsets storage)
- Config: `~/.config/dsearch/` (configuration files)
- Cache: `~/.cache/dsearch/` (downloaded feeds)

## Key Features

1. **Offline Search**: No network required after docset installation
2. **Fuzzy Matching**: Intelligent ranking of results
3. **Multiple Formats**: Output as text or markdown
4. **Interactive TUI**: Browse and search interactively
5. **Cross-Platform**: Builds for Darwin, Linux (amd64/arm64)

## Entry Points

- **Main**: `cmd/dsearch/main.go` → `cli.Execute()`
- **CLI**: `internal/cli/root.go` → Cobra command tree
- **TUI**: `internal/tui/model.go` → Bubble Tea program
