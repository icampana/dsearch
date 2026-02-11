# Suggested Commands for DSearch Development

## Essential Commands

### Build & Run
```bash
# Build the binary
make build

# Run without building
make run

# Build and run with arguments
make dev ARGS="useState"

# Install to GOPATH/bin
make install
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage
make test-cover

# Run tests in a specific package
go test -v ./internal/search

# Run tests with race detection
go test -race ./...
```

### Code Quality
```bash
# Format code
make fmt
# or
go fmt ./...

# Lint code
make lint
# or
golangci-lint run

# Static analysis
go vet ./...
staticcheck ./...
```

### Development
```bash
# Run the TUI (interactive mode)
./bin/dsearch

# Search directly
./bin/dsearch useState

# Search in specific docset
./bin/dsearch useState -d react

# List installed docsets
./bin/dsearch list

# List available docsets
./bin/dsearch available

# Install a docset
./bin/dsearch install react
```

### Dependency Management
```bash
# Add a new dependency
go get github.com/example/package

# Update dependencies
go get -u ./...
go mod tidy

# Verify dependencies
go mod verify

# Download dependencies
go mod download
```

### Debugging
```bash
# Build with debug symbols
go build -gcflags="all=-N -l" ./cmd/dsearch

# Run tests with verbose output
go test -v -run TestFunctionName ./internal/search

# Check for race conditions
go test -race ./internal/tui
```

### CI/CD
```bash
# Build for multiple platforms
make release

# Build specific platform
GOOS=linux GOARCH=amd64 go build -o bin/dsearch-linux-amd64 ./cmd/dsearch
```

## Useful Aliases

```bash
# Quick format and lint
alias ql='go fmt ./... && golangci-lint run'

# Quick test
alias qt='go test -v ./...'

# Quick build
alias qb='make build'
```

## Testing TUI

Since TUI testing requires terminal interaction, use these strategies:

```bash
# Test TUI model logic (not rendering)
go test -v ./internal/tui -run TestModel

# Run TUI with timeout for manual testing
timeout 5 ./bin/dsearch || true

# Use script command to automate TUI input
echo -e "useState\n\nc" | ./bin/dsearch
```

## Performance Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof ./internal/search
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof ./internal/search
go tool pprof mem.prof

# Benchmarking
go test -bench=. -benchmem ./internal/search
```
