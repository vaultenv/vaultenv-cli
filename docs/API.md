# VaultEnv Go API Documentation

## Overview

The VaultEnv Go API provides programmatic access to VaultEnv's core functionality, enabling developers to integrate secure environment variable management into their applications. This document covers the public API interfaces available in the `pkg/` directory.

## Design Philosophy

VaultEnv's API follows these core principles:

1. **Security by Default**: All sensitive data is encrypted automatically
2. **Interface-Driven**: Clean interfaces allow for multiple implementations
3. **Error Transparency**: Clear error types for proper handling
4. **Zero Dependencies**: Minimal external dependencies for security
5. **Backward Compatibility**: Stable interfaces within major versions

## Installation

```bash
go get github.com/vaultenv/vaultenv-cli/pkg/storage
go get github.com/vaultenv/vaultenv-cli/pkg/encryption
```

## Core Interfaces

### Storage Interface

The storage package provides the backend abstraction for storing environment variables:

```go
package storage

// Backend defines the interface for storage implementations
type Backend interface {
    // Set stores a variable with optional encryption
    Set(key, value string, encrypt bool) error
    
    // Get retrieves a variable value
    Get(key string) (string, error)
    
    // Exists checks if a variable exists
    Exists(key string) (bool, error)
    
    // Delete removes a variable
    Delete(key string) error
    
    // List returns all variable names
    List() ([]string, error)
    
    // Close closes the storage backend
    Close() error
}
```

### Encryption Interface

The encryption package provides cryptographic operations:

```go
package encryption

// Encryptor defines the interface for encryption implementations
type Encryptor interface {
    // Encrypt encrypts plaintext using the provided key
    Encrypt(plaintext []byte, key []byte) ([]byte, error)
    
    // Decrypt decrypts ciphertext using the provided key
    Decrypt(ciphertext []byte, key []byte) ([]byte, error)
    
    // EncryptString encrypts a string and returns base64-encoded result
    EncryptString(plaintext string, key []byte) (string, error)
    
    // DecryptString decrypts a base64-encoded string
    DecryptString(ciphertext string, key []byte) (string, error)
    
    // GenerateKey derives an encryption key from a password
    GenerateKey(password string, salt []byte) []byte
    
    // GenerateSalt creates a new random salt
    GenerateSalt() ([]byte, error)
    
    // Algorithm returns the name of the encryption algorithm
    Algorithm() string
}
```

## Usage Examples

### Basic Storage Operations

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func main() {
    // Create a storage backend
    backend, err := storage.GetBackend("development")
    if err != nil {
        log.Fatal(err)
    }
    defer backend.Close()
    
    // Set a variable
    err = backend.Set("API_KEY", "sk-123456", true) // encrypt = true
    if err != nil {
        log.Fatal(err)
    }
    
    // Get a variable
    value, err := backend.Get("API_KEY")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("API_KEY: %s\n", value)
}
```

### Using Different Storage Backends

```go
// File-based storage (default)
fileBackend, err := storage.GetBackendWithOptions(storage.BackendOptions{
    Environment: "production",
    Type:        "file",
    BasePath:    ".vaultenv",
})

// SQLite storage for better performance with many variables
sqliteBackend, err := storage.GetBackendWithOptions(storage.BackendOptions{
    Environment: "production", 
    Type:        "sqlite",
    BasePath:    ".vaultenv",
})

// Git-backed storage for version control
gitBackend, err := storage.GetBackendWithOptions(storage.BackendOptions{
    Environment: "production",
    Type:        "git", 
    BasePath:    ".vaultenv",
})
```

### Encrypted Storage

```go
// Create an encrypted backend with password
encryptedBackend, err := storage.GetBackendWithOptions(storage.BackendOptions{
    Environment: "production",
    Password:    "my-secure-password", // Enables encryption
    Type:        "sqlite",
})

// All operations are transparently encrypted/decrypted
err = encryptedBackend.Set("SECRET_KEY", "super-secret-value", true)
value, err := encryptedBackend.Get("SECRET_KEY") // Returns decrypted value
```

### Direct Encryption Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/vaultenv/vaultenv-cli/pkg/encryption"
)

func main() {
    // Create an encryptor
    encryptor := encryption.DefaultEncryptor() // AES-GCM-256
    
    // Generate a salt for key derivation
    salt, err := encryptor.GenerateSalt()
    if err != nil {
        log.Fatal(err)
    }
    
    // Derive key from password
    key := encryptor.GenerateKey("my-password", salt)
    
    // Encrypt data
    encrypted, err := encryptor.EncryptString("sensitive data", key)
    if err != nil {
        log.Fatal(err)
    }
    
    // Decrypt data
    decrypted, err := encryptor.DecryptString(encrypted, key)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Original: %s\n", decrypted)
}
```

### Using Different Encryption Algorithms

```go
// Standard AES-GCM (default)
aesEncryptor, err := encryption.NewEncryptor("aes-gcm-256")

// Deterministic encryption for git-friendly storage
deterministicEncryptor, err := encryption.NewEncryptor("aes-gcm-256-deterministic")

// ChaCha20-Poly1305 for better performance on some systems
chachaEncryptor, err := encryption.NewEncryptor("chacha20-poly1305")

// Check algorithm
fmt.Println("Algorithm:", encryptor.Algorithm())
```

## Error Handling

VaultEnv uses specific error types for different scenarios:

```go
import (
    "errors"
    "github.com/vaultenv/vaultenv-cli/pkg/storage"
    "github.com/vaultenv/vaultenv-cli/pkg/encryption"
)

// Storage errors
value, err := backend.Get("MISSING_KEY")
if err != nil {
    if errors.Is(err, storage.ErrNotFound) {
        // Variable doesn't exist
        fmt.Println("Variable not found")
    } else if errors.Is(err, storage.ErrInvalidName) {
        // Invalid variable name
        fmt.Println("Invalid variable name")
    } else {
        // Other error
        log.Fatal(err)
    }
}

// Encryption errors
encrypted, err := encryptor.Encrypt(data, key)
if err != nil {
    if errors.Is(err, encryption.ErrInvalidKey) {
        // Key is invalid (wrong length, etc.)
        fmt.Println("Invalid encryption key")
    } else if errors.Is(err, encryption.ErrDecryptionFailed) {
        // Decryption failed (wrong key, corrupted data)
        fmt.Println("Failed to decrypt data")
    }
}
```

## Advanced Patterns

### Custom Storage Backend

```go
package mystorage

import "github.com/vaultenv/vaultenv-cli/pkg/storage"

type CustomBackend struct {
    // Your fields
}

func (b *CustomBackend) Set(key, value string, encrypt bool) error {
    // Custom implementation
    return nil
}

func (b *CustomBackend) Get(key string) (string, error) {
    // Custom implementation
    return "", nil
}

// Implement other Backend methods...

// Register and use
func main() {
    backend := &CustomBackend{}
    // Use backend...
}
```

### Batch Operations

```go
// Efficient batch operations
backend, _ := storage.GetBackend("production")

// Get all variables
keys, err := backend.List()
if err != nil {
    log.Fatal(err)
}

// Batch read
values := make(map[string]string)
for _, key := range keys {
    value, err := backend.Get(key)
    if err == nil {
        values[key] = value
    }
}
```

### Migration Between Backends

```go
func migrateBackend(source, target storage.Backend) error {
    // List all variables
    keys, err := source.List()
    if err != nil {
        return err
    }
    
    // Copy each variable
    for _, key := range keys {
        value, err := source.Get(key)
        if err != nil {
            return fmt.Errorf("failed to get %s: %w", key, err)
        }
        
        err = target.Set(key, value, true)
        if err != nil {
            return fmt.Errorf("failed to set %s: %w", key, err)
        }
    }
    
    return nil
}
```

## Best Practices

### 1. Security

```go
// Always handle passwords securely
password := getPasswordFromUser() // Don't hardcode!
defer func() {
    // Clear password from memory
    for i := range password {
        password[i] = 0
    }
}()

// Use encryption for sensitive data
backend.Set("API_KEY", secretValue, true) // encrypt = true

// Don't log sensitive values
value, _ := backend.Get("SECRET")
log.Printf("Got value for SECRET: [REDACTED]")
```

### 2. Error Handling

```go
// Always check for specific errors
value, err := backend.Get(key)
if err != nil {
    switch {
    case errors.Is(err, storage.ErrNotFound):
        // Handle missing variable
        return defaultValue
    case errors.Is(err, encryption.ErrDecryptionFailed):
        // Handle decryption failure
        return "", fmt.Errorf("failed to decrypt %s: %w", key, err)
    default:
        // Handle other errors
        return "", err
    }
}
```

### 3. Resource Management

```go
// Always close backends
backend, err := storage.GetBackend("production")
if err != nil {
    return err
}
defer backend.Close()

// Use defer for cleanup
func processVariables() error {
    backend, err := storage.GetBackend("prod")
    if err != nil {
        return err
    }
    defer backend.Close()
    
    // Do work...
    return nil
}
```

### 4. Testing

```go
// Use test backend for unit tests
func TestMyFunction(t *testing.T) {
    // Create in-memory backend for testing
    testBackend := storage.NewMemoryBackend()
    storage.SetTestBackend(testBackend)
    defer storage.ResetTestBackend()
    
    // Your test code...
}
```

## Version Compatibility

VaultEnv follows semantic versioning:

- **Major versions** (1.x.x → 2.x.x): May include breaking changes
- **Minor versions** (0.1.x → 0.2.x): Add functionality, maintain compatibility  
- **Patch versions** (0.1.1 → 0.1.2): Bug fixes only

### Current Version: v0.1.0-beta.1

The API is in beta and stabilizing. We expect the following guarantees:

- Storage `Backend` interface: Stable
- Encryption `Encryptor` interface: Stable
- Error types: Stable
- Function signatures in `pkg/`: Stable

Internal packages (`internal/`) are not part of the public API and may change.

## Performance Considerations

```go
// Use SQLite for better performance with many variables
backend, _ := storage.GetBackendWithOptions(storage.BackendOptions{
    Type: "sqlite", // Better for >100 variables
})

// Reuse backends when possible
var globalBackend storage.Backend

func init() {
    globalBackend, _ = storage.GetBackend("default")
}

// Batch operations when possible
func setMany(vars map[string]string) error {
    backend, _ := storage.GetBackend("prod")
    defer backend.Close()
    
    for k, v := range vars {
        if err := backend.Set(k, v, true); err != nil {
            return err
        }
    }
    return nil
}
```

## Support and Resources

- **Source Code**: [github.com/vaultenv/vaultenv-cli](https://github.com/vaultenv/vaultenv-cli)
- **Issues**: [github.com/vaultenv/vaultenv-cli/issues](https://github.com/vaultenv/vaultenv-cli/issues)
- **Examples**: See `pkg/storage/example_test.go` and `pkg/encryption/example_test.go`
- **Discord**: [discord.gg/vaultenv](https://discord.gg/vaultenv)

## License

The VaultEnv Go API is available under the MIT License. See [LICENSE](../LICENSE) for details.