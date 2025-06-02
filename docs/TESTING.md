# Testing Guide for VaultEnv CLI

This document describes the testing strategy, conventions, and procedures for the VaultEnv CLI project.

## Table of Contents

- [Overview](#overview)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Writing Tests](#writing-tests)
- [Test Coverage](#test-coverage)
- [CI/CD Integration](#cicd-integration)
- [Benchmarking](#benchmarking)

## Overview

The VaultEnv CLI project uses a comprehensive testing strategy that includes:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test interactions between components
- **End-to-End Tests**: Test complete workflows
- **Benchmark Tests**: Measure performance characteristics
- **Security Tests**: Validate security properties

## Test Structure

```
vaultenv-cli/
├── pkg/                      # Public packages
│   ├── encryption/
│   │   ├── aes_gcm_test.go
│   │   ├── deterministic_test.go
│   │   ├── chacha_test.go
│   │   └── integration_test.go
│   ├── storage/
│   │   ├── memory_test.go
│   │   ├── file_test.go
│   │   ├── encrypted_test.go
│   │   ├── sqlite_test.go
│   │   ├── git_test.go
│   │   └── interface_test.go
│   ├── keystore/
│   │   ├── keystore_test.go
│   │   └── mock.go
│   ├── export/
│   │   └── formats_test.go
│   ├── access/
│   │   └── control_test.go
│   └── dotenv/
│       └── parser_test.go
├── internal/                 # Private packages
│   ├── auth/
│   │   └── password_test.go
│   ├── ui/
│   │   ├── output_test.go
│   │   └── errors_test.go
│   ├── config/
│   │   ├── config_test.go
│   │   └── migration_test.go
│   ├── keystore/
│   │   └── keystore_test.go
│   └── cmd/                  # Command tests
│       ├── init_test.go
│       ├── get_test.go
│       ├── set_test.go
│       ├── list_test.go
│       ├── export_test.go
│       ├── load_test.go
│       ├── env_test.go
│       ├── batch_test.go
│       └── integration_test.go
└── internal/test/           # Test helpers
    └── helpers.go
```

## Running Tests

### Quick Start

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### All Tests
```bash
go test ./...
```

### With Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Using the Test Script
```bash
# Run all tests with coverage report
./scripts/test-coverage.sh

# Include integration tests
./scripts/test-coverage.sh --integration

# Include benchmarks
./scripts/test-coverage.sh --bench
```

### Specific Package
```bash
go test -v github.com/vaultenv/vaultenv-cli/pkg/encryption
```

### Integration Tests
```bash
go test -tags=integration -timeout=10m ./...
```

### Benchmarks
```bash
go test -bench=. -benchmem ./...
```

### Race Detection
```bash
go test -race ./...
```

### Generate detailed coverage report
```bash
# Generate coverage data
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser (macOS)
open coverage.html
```

## Writing Tests

### Unit Test Example

```go
func TestEncryptor_EncryptDecrypt(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {
            name:    "valid_data",
            input:   "secret data",
            wantErr: false,
        },
        {
            name:    "empty_data",
            input:   "",
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            enc := NewAESGCMEncryptor()
            key := GenerateKey()
            
            encrypted, err := enc.Encrypt([]byte(tt.input), key)
            if (err != nil) != tt.wantErr {
                t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !tt.wantErr {
                decrypted, err := enc.Decrypt(encrypted, key)
                if err != nil {
                    t.Errorf("Decrypt() error = %v", err)
                    return
                }
                
                if string(decrypted) != tt.input {
                    t.Errorf("Decrypted = %v, want %v", string(decrypted), tt.input)
                }
            }
        })
    }
}
```

### Integration Test Example

```go
// +build integration

func TestIntegration_CompleteWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Create test environment
    tmpDir := t.TempDir()
    
    // Initialize project
    cmd := exec.Command("vaultenv", "init", "--name", "test")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Init failed: %v", err)
    }
    
    // Set values
    cmd = exec.Command("vaultenv", "set", "KEY", "value")
    cmd.Dir = tmpDir
    if err := cmd.Run(); err != nil {
        t.Fatalf("Set failed: %v", err)
    }
    
    // Verify
    cmd = exec.Command("vaultenv", "get", "KEY")
    cmd.Dir = tmpDir
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("Get failed: %v", err)
    }
    
    if string(output) != "value\n" {
        t.Errorf("Got %q, want %q", string(output), "value\n")
    }
}
```

### Benchmark Example

```go
func BenchmarkEncrypt(b *testing.B) {
    enc := NewAESGCMEncryptor()
    key := GenerateKey()
    data := []byte("benchmark test data")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := enc.Encrypt(data, key)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Test Helpers

```go
// internal/test/helpers.go
func CreateTestStorage(t *testing.T) storage.Backend {
    t.Helper()
    tmpDir := t.TempDir()
    store, err := storage.NewFileBackend(tmpDir, "test")
    if err != nil {
        t.Fatal(err)
    }
    return store
}

func SetupTestEnvironment(t *testing.T) string {
    t.Helper()
    tmpDir := t.TempDir()
    
    // Initialize config
    cfg := &config.Config{
        Project: config.ProjectConfig{
            Name: "test-project",
        },
    }
    
    // Save config
    configPath := filepath.Join(tmpDir, ".vaultenv", "config.yaml")
    os.MkdirAll(filepath.Dir(configPath), 0755)
    
    data, _ := yaml.Marshal(cfg)
    os.WriteFile(configPath, data, 0644)
    
    return tmpDir
}
```

## Test Coverage

### Current Coverage Status

The VaultEnv CLI maintains high test coverage across critical packages:

| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| pkg/encryption | 85.3% | >85% | ✅ Pass |
| pkg/export | 94.2% | >90% | ✅ Pass |
| pkg/storage | 77.4% | >80% | ⚠️ Near Target |
| pkg/access | 84.8% | >80% | ✅ Pass |
| pkg/keystore | 58.7% | >70% | ❌ Below Target |
| internal/auth | 32.6% | >70% | ❌ Below Target |
| internal/config | 80.4% | >80% | ✅ Pass |
| internal/cmd | 1.8% | >60% | ❌ Below Target |
| internal/ui | 90.3% | >60% | ✅ Pass |

**Overall Project Coverage: ~30.1%**

### Recent Test Fixes

The following test issues have been fixed to prepare for open source release:

1. **Storage Package**: Fixed function naming from `NewFileStorage` to `NewFileBackend`
2. **SQLite Backend**: Fixed Delete() to not error on non-existent keys
3. **Audit Logging**: Added proper logging for GET, SET, DELETE, and LIST operations
4. **Test Compilation**: Fixed undefined references to cobra errors and missing imports
5. **Version Command**: Updated tests to use `newVersionCommand()` instead of undefined `versionCmd`

### Coverage Goals

- **Overall**: 80% minimum
- **Critical packages** (encryption, storage): 90% minimum
- **Commands**: 75% minimum
- **UI/Output**: 60% minimum

### Viewing Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# View coverage by package
go tool cover -func=coverage.out | grep -E "^github.com/vaultenv"
```

### Coverage Badges

The project uses Codecov for coverage tracking. Coverage badges are displayed in the README:

```markdown
[![codecov](https://codecov.io/gh/vaultenv/vaultenv-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/vaultenv/vaultenv-cli)
```

## CI/CD Integration

### GitHub Actions

Tests run automatically on:
- Push to `main` or `develop` branches
- Pull requests to `main`

The CI pipeline includes:
1. **Test Matrix**: Multiple OS (Linux, Windows, macOS) and Go versions
2. **Linting**: golangci-lint with custom configuration
3. **Security Scanning**: gosec and Trivy
4. **Coverage Upload**: To Codecov
5. **Benchmark Results**: Stored as artifacts

### GitHub Actions Workflow

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.20', '1.21']
        os: [ubuntu-latest, windows-latest, macos-latest]
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}
    
    - name: Run tests
      run: go test -v -race -coverprofile=coverage.out ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

### Pre-commit Hooks

Install pre-commit hooks:

```bash
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
set -e

# Run tests
go test ./...

# Run linter
golangci-lint run

# Check formatting
gofmt -l . | grep -v vendor
if [ $? -eq 0 ]; then
    echo "Code is not formatted. Run 'go fmt ./...'"
    exit 1
fi
EOF

chmod +x .git/hooks/pre-commit
```

## Benchmarking

### Running Benchmarks

```bash
# All benchmarks
go test -bench=. ./...

# Specific package
go test -bench=. github.com/vaultenv/vaultenv-cli/pkg/encryption

# With memory allocation stats
go test -bench=. -benchmem ./...

# Compare results
go test -bench=. ./... > new.txt
benchcmp old.txt new.txt
```

### Benchmark Guidelines

1. **Naming**: Use `BenchmarkFunctionName` format
2. **Reset Timer**: Use `b.ResetTimer()` after setup
3. **Parallel**: Use `b.RunParallel()` for concurrent benchmarks
4. **Sub-benchmarks**: Use `b.Run()` for variations

Example:
```go
func BenchmarkStorage(b *testing.B) {
    b.Run("Memory", func(b *testing.B) {
        benchmarkStorage(b, storage.NewMemoryBackend())
    })
    
    b.Run("File", func(b *testing.B) {
        store, _ := storage.NewFileBackend("bench", "test")
        benchmarkStorage(b, store)
    })
}
```

## Debugging Tests

### Verbose Output
```bash
go test -v ./...
```

### Specific Test
```bash
go test -run TestEncryptor_EncryptDecrypt ./pkg/encryption
```

### Debug with Delve
```bash
dlv test github.com/vaultenv/vaultenv-cli/pkg/encryption
```

### Test Timeout
```bash
go test -timeout 30s ./...
```

### Print debugging
```go
func TestDebug(t *testing.T) {
    t.Logf("Debug value: %+v", complexStruct)
    // Only shown with -v flag
}
```

## Best Practices

1. **Table-Driven Tests**: Use test tables for multiple scenarios
2. **Subtests**: Use `t.Run()` for better organization
3. **Cleanup**: Use `t.Cleanup()` or `defer` for resource cleanup
4. **Helpers**: Mark helper functions with `t.Helper()`
5. **Parallel Tests**: Use `t.Parallel()` where safe
6. **Error Messages**: Include context in error messages
7. **Mock External Dependencies**: Use interfaces and mocks
8. **Test Data**: Use realistic test data
9. **Edge Cases**: Test boundaries and error conditions
10. **Documentation**: Document complex test scenarios
11. **Use t.TempDir()**: For temporary files (auto-cleanup)
12. **Keep tests fast**: Use t.Short() for slow tests
13. **Test concurrency**: Always run with -race flag
14. **Name tests clearly**: TestFunction_Scenario

## Troubleshooting

### Tests Hanging
- Check for deadlocks in concurrent code
- Add timeout: `go test -timeout 30s`
- Use race detector: `go test -race`
- Use context.WithTimeout for network operations

### Flaky Tests
- Avoid time-dependent tests
- Use deterministic random seeds
- Mock external dependencies
- Check for race conditions
- Don't rely on timing - use synchronization
- Avoid hardcoded ports - use :0 for random ports

### Coverage Issues
- Ensure all packages have tests
- Check for untested error paths
- Use build tags for conditional code
- Exclude generated code from coverage

### Permission errors
- Use t.TempDir() instead of system directories
- Check file permissions in test setup
- Run tests as non-root user in CI