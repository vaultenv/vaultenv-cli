# VaultEnv Authentication System

This package provides password management and key derivation functionality for VaultEnv CLI.

## Features

- **Secure Key Derivation**: Uses Argon2id for password-based key derivation
- **Session Caching**: Keys are cached for 15 minutes to improve UX
- **Zero-Knowledge**: Server never has access to passwords or derived keys
- **Keystore Integration**: Secure storage of salt and verification data
- **Environment Variable Support**: CI/CD friendly with `VAULTENV_PASSWORD`
- **Import/Export**: Backup and restore key configurations

## Usage

### Basic Setup

```go
import (
    "github.com/vaultenv/vaultenv-cli/internal/auth"
    "github.com/vaultenv/vaultenv-cli/internal/keystore"
)

// Initialize keystore
ks, err := keystore.NewKeystore("/path/to/data/dir")
if err != nil {
    // handle error
}
defer ks.Close()

// Create password manager
pm := auth.NewPasswordManager(ks)

// Get or create master key for a project
key, err := pm.GetOrCreateMasterKey("my-project")
if err != nil {
    // handle error
}

// Use key for encryption/decryption
// ...
```

### Environment Variable Support

For CI/CD environments, you can set the `VAULTENV_PASSWORD` environment variable:

```bash
export VAULTENV_PASSWORD="your-secure-password"
vaultenv get DATABASE_URL
```

### Password Operations

```go
// Change password for a project
err := pm.ChangePassword("my-project")

// Verify a password
err := pm.VerifyPassword("my-project", password)

// Export key configuration (for backup)
exportData, err := pm.ExportKey("my-project", password)

// Import key configuration (restore from backup)
err := pm.ImportKey("new-project", exportData, password)
```

### Session Management

```go
// Clear all cached keys
pm.ClearSessionCache()

// Clear cache for specific project
pm.ClearProjectCache("my-project")
```

## Security Considerations

1. **Password Requirements**: Minimum 8 characters enforced
2. **Key Derivation**: Uses Argon2id with:
   - Time cost: 3 iterations
   - Memory cost: 64 MB
   - Parallelism: 4 threads
   - Output: 256-bit key

3. **Salt Generation**: 32 bytes of cryptographically secure random data
4. **Verification**: SHA-256 based verification hash stored separately
5. **Session Cache**: Keys expire after 15 minutes of inactivity

## Integration with CLI Commands

See `integration_example.go` for examples of how to integrate the authentication system with Cobra commands.

## Testing

Run tests with:

```bash
go test ./internal/auth -v
```

The test suite includes:
- Key derivation tests
- Salt generation tests
- Session caching tests
- Import/export functionality
- Environment variable handling