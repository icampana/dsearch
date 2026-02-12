.PHONY: build install clean test run

# Build variables
BINARY_NAME=dsearch
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/icampana/dsearch/internal/cli.Version=$(VERSION) \
                  -X github.com/icampana/dsearch/internal/cli.Commit=$(COMMIT) \
                  -X github.com/icampana/dsearch/internal/cli.BuildDate=$(BUILD_DATE)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/dsearch

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/dsearch

# Run without building
run:
	go run ./cmd/dsearch $(ARGS)

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for multiple platforms
release:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/dsearch
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/dsearch
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/dsearch
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/dsearch

# Development: build and run with args
dev: build
	./bin/$(BINARY_NAME) $(ARGS)
