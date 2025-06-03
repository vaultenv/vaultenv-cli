# VaultEnv File Format Specifications

This document specifies the exact format of all files that VaultEnv creates, reads, or consumes. Understanding these formats enables interoperability and helps with debugging.

## Table of Contents

- [Overview](#overview)
- [Environment Variable Files](#environment-variable-files)
  - [.env File Format](#env-file-format)
  - [JSON Format](#json-format)
  - [YAML Format](#yaml-format)
  - [Docker Format](#docker-format)
- [Encrypted Storage Formats](#encrypted-storage-formats)
  - [VaultEnv Encrypted File (.vaultenv)](#vaultenv-encrypted-file-vaultenv)
  - [SQLite Database Schema](#sqlite-database-schema)
- [Configuration File Format](#configuration-file-format)
- [Export/Import Formats](#exportimport-formats)
  - [Backup Format](#backup-format)
  - [Migration Format](#migration-format)
- [Audit Log Format](#audit-log-format)
- [Batch Operation Format](#batch-operation-format)
- [Format Validation](#format-validation)

## Overview

VaultEnv supports multiple file formats for different use cases:

- **Input formats**: For loading environment variables
- **Output formats**: For exporting environment variables
- **Storage formats**: For persisting encrypted data
- **Configuration formats**: For application settings
- **Exchange formats**: For backup and migration

## Environment Variable Files

### .env File Format

The standard `.env` file format follows these rules:

#### Basic Syntax
```env
# Comments start with #
KEY=value
ANOTHER_KEY=another value

# Empty lines are ignored

# Quotes are preserved
QUOTED="value with spaces"
SINGLE_QUOTED='value with $special chars'

# Multiline values
MULTILINE="first line
second line
third line"

# Export prefix (optional, ignored by VaultEnv)
export EXPORTED_VAR=value
```

#### Parsing Rules

1. **Comments**: Lines starting with `#` are ignored
2. **Empty lines**: Ignored
3. **Key format**: `[A-Za-z_][A-Za-z0-9_]*`
4. **Values**: Everything after first `=` 
5. **Quotes**: Preserved as part of value
6. **Line continuations**: Not supported
7. **Variable expansion**: Not performed

#### Special Characters

```env
# Special characters in values
SPECIAL_CHARS=!@#$%^&*()
SPACES=value with spaces
EQUALS=value=with=equals
NEWLINE="value\nwith\nnewlines"
UNICODE=🚀 Unicode supported
```

#### Edge Cases

```env
# Edge cases handled by VaultEnv
EMPTY=
NO_VALUE=
SPACES_AROUND_EQUALS = value with spaces 
TRAILING_SPACES=value   
=INVALID_KEY
KEY_WITHOUT_VALUE
```

### JSON Format

VaultEnv can import/export JSON formatted environment variables:

#### Structure
```json
{
  "version": "1.0",
  "environment": "production",
  "variables": {
    "DATABASE_URL": "postgres://localhost/mydb",
    "API_KEY": "secret-key-123",
    "DEBUG": "false",
    "PORT": "3000"
  },
  "metadata": {
    "exported_at": "2024-01-15T10:30:00Z",
    "exported_by": "user@example.com",
    "vaultenv_version": "1.0.0"
  }
}
```

#### Schema
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "variables"],
  "properties": {
    "version": {
      "type": "string",
      "enum": ["1.0"]
    },
    "environment": {
      "type": "string"
    },
    "variables": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      }
    },
    "metadata": {
      "type": "object"
    }
  }
}
```

### YAML Format

YAML format for human-friendly editing:

#### Structure
```yaml
version: "1.0"
environment: production
variables:
  DATABASE_URL: postgres://localhost/mydb
  API_KEY: secret-key-123
  DEBUG: false
  PORT: 3000
  MULTILINE: |
    This is a
    multiline value
    with preserved formatting
metadata:
  exported_at: 2024-01-15T10:30:00Z
  exported_by: user@example.com
  vaultenv_version: 1.0.0
```

### Docker Format

Docker-compatible environment file format:

```env
# Docker environment file
DATABASE_URL=postgres://db:5432/myapp
REDIS_URL=redis://cache:6379
API_KEY=secret-key-123
NODE_ENV=production
PORT=3000
```

**Note**: Docker format doesn't support:
- Quotes around values (they become part of the value)
- Multiline values
- Variable expansion

## Encrypted Storage Formats

### VaultEnv Encrypted File (.vaultenv)

Binary format for encrypted environment storage:

#### File Structure
```
[Magic Header (8 bytes)]
[Version (4 bytes)]
[Metadata Length (4 bytes)]
[Metadata (variable)]
[Encrypted Data Length (4 bytes)]
[Encrypted Data (variable)]
[HMAC (32 bytes)]
```

#### Detailed Specification

```go
type VaultEnvFile struct {
    Magic    [8]byte  // "VAULTENV"
    Version  uint32   // Format version (currently 1)
    Metadata Metadata // Unencrypted metadata
    Data     []byte   // Encrypted payload
    HMAC     [32]byte // Authentication tag
}

type Metadata struct {
    Environment     string    `json:"environment"`
    CreatedAt       time.Time `json:"created_at"`
    ModifiedAt      time.Time `json:"modified_at"`
    Algorithm       string    `json:"algorithm"`
    KeyDerivation   string    `json:"key_derivation"`
    Compressed      bool      `json:"compressed"`
    VariableCount   int       `json:"variable_count"`
}
```

#### Encryption Process

1. Serialize variables to JSON
2. Compress if enabled (zstd)
3. Generate random nonce/IV
4. Encrypt with chosen algorithm
5. Calculate HMAC of entire file
6. Write to disk with atomic rename

### SQLite Database Schema

When using SQLite backend:

#### Tables

```sql
-- Environments table
CREATE TABLE environments (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    metadata TEXT -- JSON
);

-- Variables table
CREATE TABLE variables (
    id INTEGER PRIMARY KEY,
    environment_id INTEGER NOT NULL,
    key TEXT NOT NULL,
    value BLOB NOT NULL, -- Encrypted
    nonce BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    metadata TEXT, -- JSON
    FOREIGN KEY (environment_id) REFERENCES environments(id),
    UNIQUE(environment_id, key)
);

-- History table
CREATE TABLE history (
    id INTEGER PRIMARY KEY,
    environment_id INTEGER NOT NULL,
    operation TEXT NOT NULL,
    key TEXT,
    old_value BLOB, -- Encrypted
    new_value BLOB, -- Encrypted
    timestamp INTEGER NOT NULL,
    user TEXT,
    metadata TEXT, -- JSON
    FOREIGN KEY (environment_id) REFERENCES environments(id)
);

-- Indexes
CREATE INDEX idx_variables_env_key ON variables(environment_id, key);
CREATE INDEX idx_history_timestamp ON history(timestamp);
CREATE INDEX idx_history_env_timestamp ON history(environment_id, timestamp);
```

## Configuration File Format

VaultEnv configuration uses YAML:

```yaml
# ~/.vaultenv/config.yaml
version: 1
core:
  default_environment: development
  storage_path: ~/.vaultenv/storage
  auto_lock_timeout: 300

encryption:
  algorithm: aes-256-gcm
  key_derivation: argon2id
  key_derivation_params:
    time: 3
    memory: 65536
    parallelism: 4

storage:
  backend: sqlite
  sqlite:
    path: ${core.storage_path}/vaultenv.db
    wal_mode: true

ui:
  color: true
  timestamp_format: "2006-01-02 15:04:05"

audit:
  enabled: false
  log_file: ${core.storage_path}/audit.log
```

## Export/Import Formats

### Backup Format

Complete environment backup format:

```json
{
  "version": "1.0",
  "backup_format": "full",
  "created_at": "2024-01-15T10:30:00Z",
  "metadata": {
    "vaultenv_version": "1.0.0",
    "host": "workstation-01",
    "user": "john.doe"
  },
  "environments": [
    {
      "name": "production",
      "encrypted_data": "base64-encoded-encrypted-data",
      "encryption_metadata": {
        "algorithm": "aes-256-gcm",
        "key_derivation": "argon2id",
        "nonce": "base64-encoded-nonce"
      },
      "variables_count": 25,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "checksum": "sha256:abcdef1234567890"
}
```

### Migration Format

Format for migrating from other tools:

```yaml
# migration.yaml
version: "1.0"
source:
  type: "dotenv" # or "vault", "aws-secrets", etc.
  version: "1.0"
  
mappings:
  # Old key -> New key mappings
  DATABASE_URL: DB_CONNECTION_STRING
  REDIS_URL: CACHE_URL
  
transforms:
  # Value transformations
  - key: PORT
    transform: "integer"
  - key: DEBUG
    transform: "boolean"
    
data:
  DATABASE_URL: "postgres://old-format"
  REDIS_URL: "redis://old-format"
  PORT: "3000"
  DEBUG: "true"
```

## Audit Log Format

Structured audit logs in JSON:

```json
{
  "timestamp": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "event": "variable.set",
  "environment": "production",
  "user": "john.doe@example.com",
  "details": {
    "key": "API_KEY",
    "old_value_hash": "sha256:oldvaluehash",
    "new_value_hash": "sha256:newvaluehash",
    "source_ip": "192.168.1.100",
    "session_id": "sess_123456"
  },
  "metadata": {
    "vaultenv_version": "1.0.0",
    "command": "vaultenv set API_KEY=***"
  }
}
```

### Audit Event Types

- `environment.created`
- `environment.deleted`
- `environment.password_changed`
- `variable.set`
- `variable.deleted`
- `variable.accessed`
- `export.performed`
- `import.performed`
- `backup.created`
- `restore.performed`

## Batch Operation Format

Format for batch operations file:

```bash
# batch-operations.txt
# VaultEnv Batch Operations File
# Lines starting with # are comments

# Set variables
set DATABASE_URL=postgres://newdb:5432/app
set REDIS_URL=redis://cache:6379
set API_VERSION=v2

# Delete variables
delete OLD_API_KEY
delete DEPRECATED_FLAG

# Copy between environments
copy API_KEY from:development to:staging

# Execute in transaction
transaction begin
set FEATURE_FLAG=enabled
set FEATURE_VERSION=2.0
transaction commit
```

### Batch Commands

- `set KEY=VALUE` - Set variable
- `delete KEY` - Delete variable
- `copy KEY from:ENV to:ENV` - Copy between environments
- `transaction begin/commit/rollback` - Transaction control
- `# comment` - Comments

## Format Validation

### Validation Tools

VaultEnv provides format validation:

```bash
# Validate .env file
vaultenv validate --format env --file .env.example

# Validate JSON export
vaultenv validate --format json --file export.json

# Validate backup file
vaultenv validate --format backup --file backup.vaultenv
```

### Validation Rules

1. **Encoding**: UTF-8 required
2. **Line endings**: LF or CRLF accepted
3. **File size**: Max 100MB for imports
4. **Key format**: Must match `[A-Z_][A-Z0-9_]*`
5. **Value size**: Max 1MB per value

### Format Detection

VaultEnv auto-detects formats based on:

1. File extension
2. Magic headers
3. Content structure
4. Explicit format flag

```go
// Format detection priority
1. --format flag
2. File extension (.env, .json, .yaml, .vaultenv)
3. Content detection (magic bytes, structure)
4. Default to .env format
```

## Best Practices

1. **Use appropriate formats**:
   - `.env` for simple key-value pairs
   - JSON for programmatic access
   - YAML for complex configurations
   - Binary for encrypted storage

2. **Version your formats**:
   - Include version fields
   - Plan for forward compatibility
   - Document breaking changes

3. **Validate on import**:
   - Check encoding
   - Validate structure
   - Verify data types
   - Sanitize inputs

4. **Handle edge cases**:
   - Empty values
   - Special characters
   - Large files
   - Malformed input

## See Also

- [Command Reference](./COMMANDS.md) - Import/export commands
- [Configuration Reference](./CONFIGURATION.md) - Configuration file details
- [Security Best Practices](../guides/SECURITY_BEST_PRACTICES.md) - Secure file handling