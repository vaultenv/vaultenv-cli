# Makefile
# This Makefile serves as the primary interface for developers
# Every common task should have a make target

# Use bash for better error handling
SHELL := /bin/bash

# Binary name and paths
BINARY_NAME := vaultenv-cli
BUILD_DIR := ./build
MAIN_PATH := ./cmd/vaultenv-cli

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')

# Go build flags for optimization and versioning
LDFLAGS := -ldflags " \
    -s -w \
    -X main.version=${VERSION} \
    -X main.commit=${COMMIT} \
    -X main.buildTime=${BUILD_TIME} \
    -X main.builtBy=makefile"

.PHONY: help
help: ## Display this help message
	@echo "vaultenv-cli Development Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-15s\033[0m %s\n", "Target", "Description"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Install development dependencies
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/goreleaser/goreleaser@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "✓ Development tools installed"

.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: test
test: ## Run tests with coverage
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Tests complete. Coverage report: coverage.html"

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run --enable-all --disable=exhaustruct,depguard
	@echo "✓ Linting complete"

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@goimports -w .
	@go fmt ./...
	@echo "✓ Code formatted"

.PHONY: security
security: ## Run security checks
	@echo "Running security scan..."
	@gosec -quiet ./...
	@echo "✓ Security scan complete"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) coverage.* dist/
	@echo "✓ Clean complete"

.PHONY: install
install: build ## Install binary to GOPATH
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) $(MAIN_PATH)
	@echo "✓ $(BINARY_NAME) installed to $(GOPATH)/bin"

# Default target
.DEFAULT_GOAL := help