# BWH CLI Makefile
# Simple and efficient build automation for the BandwagonHost management tool

# Project metadata
BINARY_NAME := bwh
PACKAGE := github.com/strahe/bwh
MAIN_PACKAGE := ./cmd/bwh

# Build information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X github.com/strahe/bwh/internal/version.Version=$(VERSION) -X github.com/strahe/bwh/internal/version.BuildTime=$(BUILD_TIME) -X github.com/strahe/bwh/internal/version.CommitHash=$(COMMIT_HASH) -w -s"
BUILD_FLAGS := -trimpath

# Cross-compilation targets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Build completed: $(BINARY_NAME)"

# Install the binary to $GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(BUILD_FLAGS) $(LDFLAGS) $(MAIN_PACKAGE)
	@echo "✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Lint and format code with golangci-lint
.PHONY: lint
lint:
	@echo "Running linter and formatter..."
	@golangci-lint run --fix
	@echo "✅ Linting and formatting completed"

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "✅ Dependencies tidied"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@rm -rf dist/
	@echo "✅ Cleaned"

# Development workflow: lint, test, build
.PHONY: dev
dev: lint test build
	@echo "✅ Development workflow completed"

# Quick check (lint, test)
.PHONY: check
check: lint test
	@echo "✅ Quick check completed"

# Cross-compile for all platforms
.PHONY: build-all
build-all: clean
	@echo "Cross-compiling for all platforms..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(BINARY_NAME)-$$OS-$$ARCH; \
		if [ $$OS = "windows" ]; then OUTPUT_NAME=$$OUTPUT_NAME.exe; fi; \
		echo "Building $$OUTPUT_NAME..."; \
		GOOS=$$OS GOARCH=$$ARCH go build $(BUILD_FLAGS) $(LDFLAGS) -o dist/$$OUTPUT_NAME $(MAIN_PACKAGE); \
	done
	@echo "✅ Cross-compilation completed. Binaries in dist/"

# Show project information
.PHONY: info
info:
	@echo "Project Information:"
	@echo "  Name: $(BINARY_NAME)"
	@echo "  Package: $(PACKAGE)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Commit: $(COMMIT_HASH)"

# Show help
.PHONY: help
help:
	@echo "BWH CLI Makefile Commands:"
	@echo ""
	@echo "Building:"
	@echo "  build      - Build the binary"
	@echo "  install    - Install binary to \$$GOPATH/bin"
	@echo "  build-all  - Cross-compile for all platforms"
	@echo ""
	@echo "Development:"
	@echo "  dev        - Full development workflow (lint + test + build)"
	@echo "  check      - Quick check (lint + test)"
	@echo "  lint       - Run linter and formatter (golangci-lint)"
	@echo "  tidy       - Tidy dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  test       - Run tests"
	@echo ""
	@echo "Utilities:"
	@echo "  clean      - Clean build artifacts"
	@echo "  info       - Show project information"
	@echo "  help       - Show this help"
