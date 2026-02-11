# dsearch - Development Status

## Completed Features ✅

### Core Infrastructure
- [x] Project scaffolding (Go modules, Makefile)
- [x] CLI framework (cobra)
- [x] XDG-compliant directory structure
- [x] Configuration management

### Docset Management
- [x] Docset discovery (finds `.docset` folders)
- [x] Info.plist parsing (extracts docset metadata)
- [x] SQLite database access (reads `docSet.dsidx`)
- [x] Available docsets listing (fetches from Kapeli feeds)
- [x] Docset installation (downloads from Kapeli feeds)

### Search Engine
- [x] Basic SQLite queries (LIKE pattern matching)
- [x] Fuzzy matching with scoring (using `sahilm/fuzzy`)
- [x] Multi-docset search with filtering
- [x] Result ranking by match score

### Content Rendering
- [x] HTML to plain text conversion
- [x] HTML to markdown conversion
- [x] Output format selection (`--format text|md`)

### CLI Commands
- [x] `dsearch <query>` - Direct search with best match display
- [x] `dsearch list` - List installed docsets
- [x] `dsearch available [query]` - List downloadable docsets
- [x] `dsearch install <docset>...` - Install docsets
- [x] `dsearch version` - Show version info
- [x] `--list` flag - Show results without content
- [x] `--docset/-d` - Filter by docset
- [x] `--type/-t` - Filter by entry type
- [x] `--limit/-l` - Max results
- [x] `--format/-f` - Output format

## In Progress 🚧

### Interactive TUI
- [ ] Bubbletea application structure
- [ ] Fuzzy search input component
- [ ] Results list with keyboard navigation
- [ ] Live preview pane
- [ ] Docset filtering in TUI

## Pending Features 📋

### Docset Management
- [ ] `dsearch update [docset]...` - Update docsets
- [ ] `dsearch remove <docset>...` - Uninstall docsets
- [ ] Proper tarball extraction
- [ ] Progress bar for downloads

### Content Enhancement
- [ ] Better HTML cleaning (remove nav, footer, ads)
- [ ] Code syntax highlighting in preview
- [ ] Cross-reference link handling
- [ ] Support for multiple result display

### UX Polish
- [ ] Shell completions (bash, zsh, fish)
- [ ] Configuration file support
- [ ] Color themes for TUI
- [ ] Better error messages
- [ ] Man pages

## Directory Structure

```
dsearch/
├── cmd/dsearch/main.go           # Entry point
├── internal/
│   ├── cli/                     # CLI commands
│   │   ├── root.go              # Root command and main logic
│   │   ├── version.go           # Version command
│   │   ├── list.go             # List docsets
│   │   ├── available.go         # List downloadable docsets
│   │   └── install.go          # Install docsets
│   ├── config/                  # Configuration
│   │   └── paths.go            # XDG directory management
│   ├── docset/                  # Docset handling
│   │   └── docset.go           # Docset discovery and reading
│   ├── search/                  # Search engine
│   │   └── engine.go            # Search with fuzzy matching
│   ├── render/                  # Content rendering
│   │   └── render.go            # HTML to text/markdown
│   └── manager/                 # Docset management
│       └── feeds.go             # Kapeli feeds API
├── Makefile                     # Build automation
├── go.mod / go.sum             # Go modules
└── README.md                   # Project documentation
```

## Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| spf13/cobra | CLI framework | 1.10.2 |
| modernc.org/sqlite | SQLite driver | 1.45.0 |
| sahilm/fuzzy | Fuzzy matching | 0.1.1 |
| golang.org/x/net/html | HTML parsing | 0.50.0 |
| schollz/progressbar/v3 | Progress bars | 3.19.0 |

## Usage Examples

```bash
# List installed docsets
dsearch list

# Find available docsets
dsearch available react

# Install docset
dsearch install React

# Search in all docsets
dsearch useState

# Search in specific docset
dsearch useState -d react

# Get results as markdown
dsearch useState --format md

# List results only
dsearch useState --list -l 20

# Filter by type
dsearch useEffect --type Method
```

## Next Steps

1. **Interactive TUI** - Build the bubbletea application for live fuzzy searching
2. **Docset Updates** - Implement update/remove commands
3. **Testing** - Add unit tests for core components
4. **Polish** - Better HTML cleaning, error handling, and UX improvements
