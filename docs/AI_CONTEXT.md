# AI CONTEXT & ARCHITECTURAL MAP
> Last Updated: 2026-02-12 (Post-Production Cleanup)

## 1. Tech Stack & Versions
- **Language:** Go 1.25.7
- **CLI Framework:** `spf13/cobra` (v1.10.2)
- **Search Engine:** In-memory fuzzy search using `sahilm/fuzzy`
- **Documentation Source:** DevDocs API (devdocs.io)
- **Rendering:**
  - HTML parsing: `codeberg.org/readeck/go-readability`
  - Markdown conversion: `github.com/JohannesKaufmann/html-to-markdown/v2`
- **Storage:** XDG-compliant filesystem storage (JSON indices, HTML content)
- **Testing:** Go's built-in testing with race detector enabled

## 2. High-Level Architecture
`dsearch` is a CLI-based offline documentation browser inspired by Zeal/Dash but tailored for the terminal. It follows a "fetch-index-search" architecture:
1.  **Fetcher:** Downloads documentation sets (manifest, indices, content) from DevDocs.
2.  **Indexer:** Stores metadata and search indices locally in XDG data directories.
3.  **Search Engine:** Loads indices into memory and performs fuzzy matching on query with O(1) slug lookup.
4.  **Renderer:** Transforms stored HTML content into readable terminal text or Markdown on demand.

The application follows a standard Go project layout with a clear separation between CLI interface (`cmd`, `internal/cli`) and core logic (`internal/devdocs`, `internal/search`, `internal/render`).

## 3. Critical Data Flows

### Installation Flow (`dsearch install <doc>`)
1.  **Manifest Fetch:** `devdocs.Client` fetches `docs.json` from `devdocs.io`.
2.  **Selection:** User input (e.g., "react@18") is parsed and matched against manifest.
3.  **Download:**
    *   Index (`index.json`): Contains all searchable entries.
    *   Content (`db.json`): Contains compiled HTML blobs.
4.  **Storage:** `devdocs.Store` unpacks `db.json` into individual HTML files in `~/.local/share/dsearch/docs/<slug>/content/`.
5.  **Error Handling:** Partial failures are tracked and reported with aggregated error messages.

### Search Flow (`dsearch <query>`)
1.  **Initialization:** `loadSearchEngine` loads selected (or all) indices from disk.
2.  **Execution:** `search.Engine` aggregates entries and performs fuzzy matching with O(1) slug-to-index reverse lookup.
3.  **Optimization:** If `--doc` is passed, only relevant indices are loaded/searched.
4.  **Result:** Matches are ranked by score. Empty results return cleanly (no error).

### Render Flow (View Result)
1.  **Load:** `devdocs.Store` reads the specific HTML file for the selected entry.
2.  **Process:** `render.Renderer` uses `readability` to strip navigation/ads.
3.  **Format:** Content is converted to requested format (Text/Markdown) and printed to stdout.

## 4. Key Directory Map
- `cmd/dsearch`: Application entry point (`main.go`).
- `internal/cli`: Cobra command definitions and flag handling.
- `internal/config`:
    - `paths.go`: XDG path configuration and management.
    - `migrate.go`: One-time migration from old double-nested paths.
- `internal/devdocs`:
    - `client.go`: HTTP client for DevDocs API.
    - `store.go`: Local filesystem storage with path traversal protection.
    - `types.go`: Core data models (Doc, Index, Entry).
- `internal/render`: HTML-to-Text/Markdown conversion logic.
- `internal/search`: In-memory fuzzy search engine with optimized slug lookups.

## 5. Developer Guide / Conventions
- **Error Handling:** Go 1.13+ style wrapping (`fmt.Errorf("...: %w", err)`). Loop-based commands aggregate errors.
- **Configuration:** Strictly adheres to XDG Base Directory specification.
- **Dependency Injection:** Explicit constructors (`New...`) used for testability.
- **Output:**
    - `stdout`: Search results and content.
    - `stderr`: Logs, warnings, and error messages (using `fmt.Fprintf(os.Stderr)`).
    - JSON output supported via `--json` flag for integration.
- **Testing:**
    - All tests use `t.Parallel()` for isolation.
    - Race detector enabled in CI/Makefile (`-race` flag).
    - Comprehensive coverage for config, devdocs, render, and search packages.
- **Modern Go:** Uses `any` instead of `interface{}` (Go 1.18+).

## 6. Recent Production Cleanup (2026-02-12)
- **Path Migration:** Fixed double-nesting bug (`docs/docs/<slug>` → `docs/<slug>`) with automatic migration.
- **Dead Code Removal:** Removed unused functions and dependencies (progressbar).
- **Search Optimization:** Added reverse slug map for O(1) lookups (was O(n)).
- **Security:** Strengthened path traversal checks (rejects absolute paths and `..`).
- **Error Handling:** Install/uninstall commands now aggregate and report partial failures.
- **Testing:** Added config tests, enabled parallelization across all test suites.

## 7. Known Technical Debt / Watchlist
- **Memory Usage:** `search.Engine` loads indices into memory. With many large docsets installed, this could bloat memory usage.
- **Rendering Fragility:** `render.go` uses `readability` with a dummy base URL (`http://localhost/docset`). Complex relative links or assets might break.
- **HTML Parsing:** `extractText` in `render.go` manually traverses HTML nodes, which can be brittle compared to using a robust policy-based sanitizer/extractor.
- **Lack of Concurrency:** Installation downloads `db.json` and extracts sequentially. Large docs could benefit from concurrent processing (though `db.json` is a single file).
- **Global State (deferred):** CLI commands use package-level `paths` variable. Refactoring to pass config through function parameters would improve testability.
- **Context Propagation (deferred):** HTTP client methods don't accept `context.Context` for cancellation. Adding this would require threading context through all CLI command handlers.
