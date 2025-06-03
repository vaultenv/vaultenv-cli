# VaultEnv Development Guide

Welcome to the VaultEnv development guide! This document will help you contribute to VaultEnv, understand its internals, and maintain code quality.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)
- [Project Structure](#project-structure)
- [Build System](#build-system)
- [Testing Strategy](#testing-strategy)
- [Debugging Techniques](#debugging-techniques)
- [Development Workflow](#development-workflow)
- [Release Process](#release-process)
- [Common Development Tasks](#common-development-tasks)

## Development Environment Setup

### Prerequisites

- **Go 1.22 or later** - [Install Go](https://golang.org/dl/)
- **Git** - For version control
- **Make** - Build automation (usually pre-installed on Unix)
- **SQLite3** - For SQLite storage backend

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/vaultenv/vaultenv-cli.git
cd vaultenv-cli

# Install dependencies
go mod download

# Run tests to verify setup
make test

# Build the binary
make build

# Run the built binary
./build/vaultenv version
```

### Recommended Tools

```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

## Project Structure

```
vaultenv-cli/
├── cmd/
│   └── vaultenv-cli/
│       ├── main.go              # Application entry point
│       └── main_test.go         # Entry point tests
├── internal/                    # Private packages (not importable)
│   ├── auth/                    # Authentication and password handling
│   │   ├── password.go          # Password operations
│   │   └── password_test.go
│   ├── cmd/                     # CLI command implementations
│   │   ├── aliases.go           # Command aliases
│   │   ├── batch.go            # Batch operations
│   │   ├── completion.go       # Shell completions
│   │   ├── config.go           # Configuration management
│   │   ├── env.go              # Environment management
│   │   ├── execute.go          # Command execution
│   │   ├── export.go           # Export functionality
│   │   ├── get.go              # Get variables
│   │   ├── git.go              # Git integration
│   │   ├── history.go          # History tracking
│   │   ├── init.go             # Initialization
│   │   ├── list.go             # List variables
│   │   ├── load.go             # Load from files
│   │   ├── migrate.go          # Migration tools
│   │   ├── security.go         # Security operations
│   │   ├── set.go              # Set variables
│   │   ├── shell.go            # Shell integration
│   │   └── version.go          # Version info
│   ├── config/                 # Configuration management
│   │   ├── config.go
│   │   └── migration.go
│   ├── keystore/               # Key management
│   │   ├── keystore.go
│   │   └── environment_keys.go
│   └── ui/                     # Terminal UI utilities
│       ├── output.go           # Output formatting
│       └── errors.go           # Error handling
├── pkg/                        # Public packages (importable)
│   ├── encryption/             # Encryption implementations
│   │   ├── interface.go        # Encryption interface
│   │   ├── aes_gcm.go         # AES-256-GCM
│   │   ├── chacha.go          # ChaCha20-Poly1305
│   │   └── deterministic.go   # Deterministic encryption
│   └── storage/                # Storage backends
│       ├── interface.go        # Storage interface
│       ├── file.go            # File backend
│       ├── sqlite.go          # SQLite backend
│       ├── memory.go          # In-memory backend
│       ├── git.go             # Git backend
│       └── encrypted.go       # Encryption wrapper
├── scripts/                    # Build and test scripts
│   ├── test-completions.sh    # Test shell completions
│   └── test-coverage.sh       # Generate coverage
├── docs/                      # Documentation
├── Makefile                   # Build automation
├── go.mod                     # Go module definition
└── go.sum                     # Dependency checksums
```

## Build System

### Makefile Targets

```bash
# Build
make build          # Build for current platform
make build-all      # Build for all platforms
make install        # Install to $GOPATH/bin

# Testing
make test           # Run all tests
make test-unit      # Unit tests only
make test-integration # Integration tests
make coverage       # Generate coverage report
make test-race      # Test with race detector

# Code Quality
make lint           # Run linters
make fmt            # Format code
make vet            # Run go vet

# Documentation
make docs           # Generate documentation

# Release
make release        # Create release binaries
make clean          # Clean build artifacts
```

### Build Tags

```bash
# Build with specific features
go build -tags "sqlite" ./cmd/vaultenv-cli

# Build without SQLite support
go build -tags "nosqlite" ./cmd/vaultenv-cli
```

## Testing Strategy

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/encryption/...

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestEncryption ./pkg/encryption

# Run with race detection
go test -race ./...

# Generate coverage
make coverage
```

### Writing Tests

#### Unit Test Example

```go
// internal/auth/password_test.go
func TestDeriveKey(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantLen  int
        wantErr  bool
    }{
        {"valid password", "testpass123", 32, false},
        {"empty password", "", 0, true},
        {"unicode password", "test🔐pass", 32, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            key, err := DeriveKey(tt.password)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Len(t, key, tt.wantLen)
        })
    }
}
```

#### Integration Test Example

```go
// internal/cmd/integration_test.go
func TestSetGetFlow(t *testing.T) {
    // Setup test environment
    testDir := t.TempDir()
    t.Setenv("VAULTENV_HOME", testDir)
    
    // Initialize
    require.NoError(t, runCommand("init"))
    
    // Set variable
    require.NoError(t, runCommand("set", "TEST_KEY=test_value"))
    
    // Get variable
    output, err := captureOutput(func() error {
        return runCommand("get", "TEST_KEY")
    })
    require.NoError(t, err)
    assert.Contains(t, output, "test_value")
}
```

### Test Helpers

```go
// internal/test/helpers.go

// Create temporary .env file
func TempEnvFile(t *testing.T, content string) string {
    t.Helper()
    file := filepath.Join(t.TempDir(), ".env")
    require.NoError(t, os.WriteFile(file, []byte(content), 0600))
    return file
}

// Capture command output
func CaptureOutput(f func() error) (string, error) {
    old := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w
    
    err := f()
    
    w.Close()
    os.Stdout = old
    
    out, _ := io.ReadAll(r)
    return string(out), err
}
```

## Debugging Techniques

### Debug Logging

```bash
# Enable debug logging
export VAULTENV_DEBUG=true
vaultenv set KEY=value

# Enable trace logging
export VAULTENV_TRACE=true
vaultenv list
```

### Using Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a command
dlv debug ./cmd/vaultenv-cli -- set KEY=value

# Debug a test
dlv test ./internal/auth -- -test.run TestDeriveKey
```

### Common Debug Commands

```go
// Add debug logging
if os.Getenv("VAULTENV_DEBUG") == "true" {
    log.Printf("[DEBUG] Setting key: %s", key)
}

// Add breakpoint for delve
runtime.Breakpoint()
```

## Development Workflow

### 1. Create Feature Branch

```bash
git checkout -b feature/your-feature
```

### 2. Make Changes

Follow the coding standards:
- Use `gofmt` for formatting
- Follow Go idioms
- Add comments for exported functions
- Keep functions small and focused

### 3. Add Tests

- Write tests for new functionality
- Ensure existing tests pass
- Aim for >80% coverage

### 4. Run Checks

```bash
# Format code
make fmt

# Run linters
make lint

# Run tests
make test

# Check coverage
make coverage
```

### 5. Commit Changes

```bash
git add .
git commit -m "feat: add new feature

- Detailed description
- Closes #123"
```

Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `test:` Testing
- `refactor:` Code refactoring
- `chore:` Maintenance

### 6. Push and Create PR

```bash
git push origin feature/your-feature
```

Create a pull request with:
- Clear description
- Link to related issues
- Test results
- Screenshots if UI changes

## Release Process

### 1. Update Version

```go
// internal/cmd/version.go
const Version = "0.2.0"
```

### 2. Update Changelog

```markdown
## [0.2.0] - 2024-02-01

### Added
- New feature X
- Enhancement Y

### Fixed
- Bug fix Z

### Changed
- Breaking change A
```

### 3. Create Tag

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

### 4. Build Release

```bash
make release
```

This creates binaries for all platforms in `dist/`.

## Common Development Tasks

### Adding a New Command

1. Create command file:

```go
// internal/cmd/newcmd.go
package cmd

import (
    "github.com/spf13/cobra"
)

func newNewCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "newcmd [args]",
        Short: "Brief description",
        Long:  `Detailed description`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
}
```

2. Register in root command:

```go
// internal/cmd/root.go
rootCmd.AddCommand(newNewCommand())
```

3. Add tests:

```go
// internal/cmd/newcmd_test.go
func TestNewCommand(t *testing.T) {
    // Test implementation
}
```

### Adding a Storage Backend

1. Implement the interface:

```go
// pkg/storage/new_backend.go
type NewBackend struct {
    // fields
}

func (b *NewBackend) Set(key, value string, encrypt bool) error {
    // Implementation
}

// Implement other Backend methods...
```

2. Add to factory:

```go
// pkg/storage/interface.go
case "new":
    baseBackend, err = NewNewBackend(opts.BasePath, opts.Environment)
```

3. Add tests following the pattern in `storage_test.go`

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkEncryption ./pkg/encryption

# With memory allocation stats
go test -bench=. -benchmem ./...

# Compare results
go test -bench=. ./... > old.txt
# make changes
go test -bench=. ./... > new.txt
benchcmp old.txt new.txt
```

## Code Style Guidelines

### General

- Use meaningful variable names
- Keep line length under 100 characters
- Group related functionality
- Prefer early returns

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to set key %s: %w", key, err)
}

// Bad
if err != nil {
    return err
}
```

### Comments

```go
// Package storage provides backend implementations for storing
// environment variables with optional encryption.
package storage

// Backend defines the interface for storage implementations.
// All implementations must be thread-safe.
type Backend interface {
    // Set stores a variable with optional encryption.
    // Returns ErrInvalidName if the key contains invalid characters.
    Set(key, value string, encrypt bool) error
}
```

## Security Guidelines

1. **Never log sensitive data**
   ```go
   log.Printf("Setting variable: %s", key) // Don't log value
   ```

2. **Clear sensitive memory**
   ```go
   defer func() {
       for i := range password {
           password[i] = 0
       }
   }()
   ```

3. **Use crypto/rand for random data**
   ```go
   import "crypto/rand"
   // Not math/rand
   ```

4. **Validate all input**
   ```go
   if !isValidKey(key) {
       return ErrInvalidName
   }
   ```

## Getting Help

- **GitHub Issues**: [Report bugs or request features](https://github.com/vaultenv/vaultenv-cli/issues)
- **Discord**: [Join our community](https://discord.gg/vaultenv)
- **Documentation**: Check the [docs](./docs) directory

## Next Steps

1. Read [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines
2. Check [open issues](https://github.com/vaultenv/vaultenv-cli/issues) for tasks
3. Join Discord to discuss ideas
4. Start with "good first issue" labeled tasks

Thank you for contributing to VaultEnv! 🚀