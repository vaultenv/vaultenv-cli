# Makefile
# This Makefile serves as the primary interface for developers
# Every common task should have a make target

# Use bash for better error handling
SHELL := /bin/bash

# Binary name and paths
BINARY_NAME := vaultenv-cli
BUILD_DIR := ./build
MAIN_PATH := ./cmd/vaultenv-cli
COVERAGE_DIR := ./coverage

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

# Test coverage threshold
COVERAGE_THRESHOLD := 80

.PHONY: help
help: ## Display this help message
	@echo "vaultenv-cli Development Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-20s\033[0m %s\n", "Target", "Description"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Install development dependencies
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/goreleaser/goreleaser@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/vektra/mockery/v2@latest
	@echo "✓ Development tools installed"

.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "✓ Tests complete"

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -v -short -race ./...
	@echo "✓ Unit tests complete"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -tags=integration -timeout=10m ./...
	@echo "✓ Integration tests complete"

.PHONY: test-coverage
test-coverage: ## Run tests with detailed coverage
	@echo "Running test coverage analysis..."
	@./scripts/test-coverage.sh
	@echo "✓ Coverage analysis complete"

.PHONY: test-coverage-html
test-coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Opening coverage report in browser..."
	@open coverage/coverage.html || xdg-open coverage/coverage.html

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -bench=. -benchmem -run=^$ ./... | tee $(COVERAGE_DIR)/benchmark.txt
	@echo "✓ Benchmarks complete. Results: $(COVERAGE_DIR)/benchmark.txt"

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run
	@echo "✓ Linting complete"

.PHONY: lint-fix
lint-fix: ## Run linters with auto-fix
	@echo "Running linters with fixes..."
	@golangci-lint run --fix
	@echo "✓ Linting with fixes complete"

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
	@go list -json -deps ./... | nancy sleuth
	@echo "✓ Security scan complete"

.PHONY: check
check: fmt lint test security ## Run all checks (fmt, lint, test, security)
	@echo "✓ All checks passed!"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR) coverage.* dist/ *.test
	@echo "✓ Clean complete"

.PHONY: install
install: build ## Install binary to GOPATH
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) $(MAIN_PATH)
	@echo "✓ $(BINARY_NAME) installed to $(GOPATH)/bin"

.PHONY: run
run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t vaultenv-cli:$(VERSION) .
	@echo "✓ Docker image built: vaultenv-cli:$(VERSION)"

.PHONY: generate
generate: ## Generate code (mocks, etc.)
	@echo "Generating code..."
	@go generate ./...
	@echo "✓ Code generation complete"

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

.PHONY: update
update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies updated"

.PHONY: release-dry-run
release-dry-run: ## Test release process
	@echo "Running release dry-run..."
	@goreleaser release --snapshot --clean
	@echo "✓ Release dry-run complete"

.PHONY: ci
ci: deps check ## Run CI pipeline locally
	@echo "✓ CI pipeline passed!"

# Default target
.DEFAULT_GOAL := help