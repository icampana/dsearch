# DSearch Code Style & Conventions

## Naming Conventions

### Packages
- All lowercase, single word when possible (e.g., `search`, `docset`, `render`)
- No underscores or mixed case
- Descriptive but concise

### Variables & Constants
- **Exported**: PascalCase (e.g., `Engine`, `NewModel`, `Docsets`)
- **Unexported**: camelCase (e.g., `docsets`, `limit`, `input`)
- **Constants**: UPPER_SNAKE_CASE for exported, camelCase for unexported
- **Interface receivers**: Single letter (e.g., `m *Model`, `e *Engine`)

### Functions
- **Exported**: PascalCase with descriptive verbs (e.g., `Search`, `Load`, `Discover`)
- **Unexported**: camelCase (e.g., `parseInfoPlist`, `countEntries`)

### Types
- **Structs**: PascalCase, descriptive names (e.g., `Docset`, `Entry`, `Result`)
- **Interfaces**: PascalCase, often single abstract noun (e.g., `Reader`, `Writer`)

## Error Handling

### Always Check Errors
```go
// GOOD
data, err := os.ReadFile(path)
if err != nil {
    return nil, fmt.Errorf("reading file: %w", err)
}

// BAD
data, _ := os.ReadFile(path) // Never ignore errors
```

### Wrap Errors with Context
```go
// GOOD - wraps error with context
if err != nil {
    return fmt.Errorf("searching %s: %w", ds.Name, err)
}

// BAD - loses context
if err != nil {
    return err
}
```

### Use Error Sentinel Values When Appropriate
```go
// For expected, recoverable errors
var (
    ErrNoResults = errors.New("no results found")
    ErrNotFound  = errors.New("not found")
)
```

## Comments & Documentation

### Exported Functions Must Have Godoc
```go
// Search performs a search across all docsets with fuzzy matching.
// It accepts a query string and optional docset name filters.
// Returns a slice of results sorted by relevance score.
func (e *Engine) Search(query string, docsetNames []string) ([]Result, error) {
    // ...
}
```

### Inline Comments for Non-Obvious Logic
```go
// Normalize score to 0-1 range
score := float64(match.Score) / 100.0
```

## Code Organization

### File Structure (single responsibility)
- One main type per file when large (e.g., `engine.go`, `docset.go`)
- Related helper functions in same file
- Tests in `*_test.go` adjacent to implementation

### Import Order
```go
import (
    // 1. Standard library
    "fmt"
    "os"
    "strings"

    // 2. Third-party packages
    "github.com/spf13/cobra"
    "github.com/sahilm/fuzzy"

    // 3. Internal packages
    "github.com/icampana/dsearch/internal/docset"
    "github.com/icampana/dsearch/internal/search"
)
```

## Interfaces

### Accept Interfaces, Return Structs
```go
// GOOD - function accepts interface
func ProcessItems(r docset.Reader) error {
    // ...
}

// GOOD - constructor returns concrete type
func NewEngine() *Engine {
    // ...
}
```

### Keep Interfaces Small and Focused
```go
// GOOD - single responsibility
type Loader interface {
    Load(path string) (Docset, error)
}

type Searcher interface {
    Search(query string) ([]Result, error)
}

// BAD - too many responsibilities
type Manager interface {
    Load(path string) (Docset, error)
    Search(query string) ([]Result, error)
    Save(docset Docset) error
    Delete(id string) error
}
```

## Concurrency

### Use Context for Cancellation
```go
func (e *Engine) Search(ctx context.Context, query string) ([]Result, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Perform search
    }
}
```

### Prefer Channels Over Shared Memory
```go
// GOOD - channel for communication
results := make(chan Result, 10)
go func() {
    for _, item := range items {
        results <- process(item)
    }
    close(results)
}()

// AVOID - shared mutable state
var mu sync.Mutex
var sharedState []Result
```

### Always Clean Up Goroutines
```go
// GOOD - proper cleanup
func Worker(ctx context.Context, jobs <-chan Job) {
    for {
        select {
        case <-ctx.Done():
            return // Clean shutdown
        case job, ok := <-jobs:
            if !ok {
                return // Channel closed
            }
            process(job)
        }
    }
}
```

## Resource Management

### Use Defer for Cleanup
```go
func ReadFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close() // Always close

    return io.ReadAll(f)
}
```

### Database Connections
```go
func Query(db *sql.DB, query string) ([]Result, error) {
    rows, err := db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close() // Always close rows

    // Process rows
}
```

## Testing

### Table-Driven Tests
```go
func TestSearch(t *testing.T) {
    tests := []struct {
        name      string
        query     string
        wantCount int
        wantErr   bool
    }{
        {
            name:      "valid query",
            query:     "test",
            wantCount: 5,
            wantErr:   false,
        },
        {
            name:      "empty query",
            query:     "",
            wantCount: 0,
            wantErr:   true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // test implementation
        })
    }
}
```

### Test Both Success and Error Paths
```go
func TestLoad(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        _, err := Load("valid.docset")
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
    })

    t.Run("invalid path", func(t *testing.T) {
        _, err := Load("nonexistent.docset")
        if err == nil {
            t.Fatal("expected error, got nil")
        }
    })
}
```

### Use Race Detection
```bash
go test -race ./internal/tui
```

## CLI-Specific Conventions

### Command Structure
- Use `spf13/cobra` for command hierarchy
- Keep command handlers in `internal/cli`
- Each command in its own file (e.g., `install.go`, `search.go`)

### Error Aggregation in Loop Commands
```go
// GOOD - track and report partial failures
var errors []string
successCount := 0

for _, item := range items {
    if err := process(item); err != nil {
        errors = append(errors, fmt.Sprintf("failed to process %s: %v", item, err))
        continue
    }
    successCount++
}

if len(errors) > 0 {
    fmt.Fprintf(os.Stderr, "\n%d operation(s) failed:\n", len(errors))
    for _, errMsg := range errors {
        fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
    }
    if successCount == 0 {
        return fmt.Errorf("all operations failed")
    }
    return fmt.Errorf("%d operation(s) failed (see above)", len(errors))
}
```

### Output Conventions
- `stdout`: Search results, content, JSON output
- `stderr`: Errors, warnings, progress messages (use `fmt.Fprintf(os.Stderr, ...)`)
- Never use `log.Printf` for CLI output - use `fmt.Fprintf` for consistency



### Model Implementation
```go
type Model struct {
    // Dependencies
    engine *search.Engine

    // UI Components
    input  textinput.Model
    list   list.Model

    // State
    loading bool
    width   int
    height  int
}

func (m Model) Init() tea.Cmd {
    return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyCtrlC:
            return m, tea.Quit
        }
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    return m.list.View()
}
```

### Message Types
```go
// Use type aliasing for custom messages
type searchFinishedMsg struct {
    results []search.Result
    query   string
}

// Send messages as commands
func performSearch(query string) tea.Cmd {
    return func() tea.Msg {
        results, _ := engine.Search(query)
        return searchFinishedMsg{results: results}
    }
}
```

## Formatting

### Standard Go Formatting
```bash
go fmt ./...
# or
gofmt -w .
```

### Line Length
- Prefer 100-120 characters max
- Break long lines logically

### Imports
- Run `go mod tidy` regularly
- Remove unused imports
- Don't use `.` imports except in tests

## Security

### Never Commit Secrets
- Add `.env` to `.gitignore`
- Don't commit API keys, tokens, passwords
- Use environment variables for secrets

### Input Validation
```go
func Search(query string) error {
    if query == "" {
        return fmt.Errorf("query cannot be empty")
    }
    if len(query) > 1000 {
        return fmt.Errorf("query too long")
    }
    // ...
}
```

### SQL Injection Prevention
```go
// GOOD - parameterized queries
query := "SELECT * FROM items WHERE name = ?"
db.Query(query, userInput)

// BAD - string concatenation (vulnerable to injection)
query := fmt.Sprintf("SELECT * FROM items WHERE name = '%s'", userInput)
db.Query(query)
```
