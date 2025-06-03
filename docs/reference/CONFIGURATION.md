# VaultEnv Configuration Reference

This document details all configuration options available in VaultEnv, their default values, and how they affect application behavior.

## Table of Contents

- [Configuration Overview](#configuration-overview)
- [Configuration File Locations](#configuration-file-locations)
- [Configuration Precedence](#configuration-precedence)
- [Configuration Options](#configuration-options)
  - [Core Settings](#core-settings)
  - [Encryption Settings](#encryption-settings)
  - [Storage Settings](#storage-settings)
  - [Git Integration](#git-integration)
  - [Security Settings](#security-settings)
  - [UI and Output](#ui-and-output)
  - [Performance](#performance)
  - [Audit and Logging](#audit-and-logging)
- [Environment Variable Overrides](#environment-variable-overrides)
- [Example Configurations](#example-configurations)
- [Configuration Migration](#configuration-migration)

## Configuration Overview

VaultEnv uses a hierarchical configuration system that allows for flexible customization while maintaining secure defaults. Configuration can be set at multiple levels and through various methods.

## Configuration File Locations

VaultEnv looks for configuration files in the following order:

1. **Command-line specified**: `--config /path/to/config.yaml`
2. **Project-specific**: `./.vaultenv/config.yaml`
3. **User-specific**: `~/.vaultenv/config.yaml`
4. **System-wide**: `/etc/vaultenv/config.yaml`
5. **Built-in defaults**

### File Format

Configuration files use YAML format:

```yaml
# ~/.vaultenv/config.yaml
version: 1
core:
  default_environment: development
  auto_lock_timeout: 300
encryption:
  algorithm: aes-256-gcm
  key_derivation: argon2id
```

## Configuration Precedence

Configuration values are resolved in the following order (highest to lowest precedence):

1. Command-line flags
2. Environment variables
3. Project-specific config file
4. User-specific config file
5. System-wide config file
6. Built-in defaults

## Configuration Options

### Core Settings

Core settings control fundamental VaultEnv behavior.

#### core.default_environment
- **Type**: `string`
- **Default**: `"default"`
- **Description**: Default environment when none is specified
- **Example**: 
  ```yaml
  core:
    default_environment: development
  ```

#### core.auto_lock_timeout
- **Type**: `integer` (seconds)
- **Default**: `300` (5 minutes)
- **Description**: Time before automatic session lock
- **Example**: 
  ```yaml
  core:
    auto_lock_timeout: 600  # 10 minutes
  ```

#### core.storage_path
- **Type**: `string`
- **Default**: `"~/.vaultenv/storage"`
- **Description**: Base path for encrypted storage files
- **Example**: 
  ```yaml
  core:
    storage_path: /opt/vaultenv/data
  ```

#### core.workspace_detection
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Auto-detect project workspace
- **Example**: 
  ```yaml
  core:
    workspace_detection: false
  ```

### Encryption Settings

Control encryption algorithms and parameters.

#### encryption.algorithm
- **Type**: `string`
- **Default**: `"aes-256-gcm"`
- **Options**: `"aes-256-gcm"`, `"chacha20poly1305"`
- **Description**: Encryption algorithm for secret storage
- **Example**: 
  ```yaml
  encryption:
    algorithm: chacha20poly1305
  ```

#### encryption.key_derivation
- **Type**: `string`
- **Default**: `"argon2id"`
- **Options**: `"argon2id"`, `"scrypt"`, `"pbkdf2"`
- **Description**: Key derivation function
- **Example**: 
  ```yaml
  encryption:
    key_derivation: scrypt
  ```

#### encryption.key_derivation_params
- **Type**: `object`
- **Default**: Algorithm-specific
- **Description**: Parameters for key derivation
- **Example**: 
  ```yaml
  encryption:
    key_derivation_params:
      time: 3
      memory: 65536
      parallelism: 4
      salt_length: 32
  ```

#### encryption.deterministic_fields
- **Type**: `array[string]`
- **Default**: `[]`
- **Description**: Fields to encrypt deterministically for searching
- **Example**: 
  ```yaml
  encryption:
    deterministic_fields:
      - API_KEY
      - DATABASE_NAME
  ```

### Storage Settings

Configure how VaultEnv stores encrypted data.

#### storage.backend
- **Type**: `string`
- **Default**: `"sqlite"`
- **Options**: `"sqlite"`, `"file"`, `"memory"`
- **Description**: Storage backend type
- **Example**: 
  ```yaml
  storage:
    backend: file
  ```

#### storage.sqlite.path
- **Type**: `string`
- **Default**: `"${core.storage_path}/vaultenv.db"`
- **Description**: SQLite database path
- **Example**: 
  ```yaml
  storage:
    sqlite:
      path: /var/lib/vaultenv/data.db
  ```

#### storage.sqlite.wal_mode
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable Write-Ahead Logging
- **Example**: 
  ```yaml
  storage:
    sqlite:
      wal_mode: false
  ```

#### storage.file.extension
- **Type**: `string`
- **Default**: `".vaultenv"`
- **Description**: File extension for encrypted files
- **Example**: 
  ```yaml
  storage:
    file:
      extension: .secrets
  ```

#### storage.compression
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable compression before encryption
- **Example**: 
  ```yaml
  storage:
    compression: false
  ```

### Git Integration

Settings for Git synchronization features.

#### git.enabled
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable Git integration features
- **Example**: 
  ```yaml
  git:
    enabled: false
  ```

#### git.auto_commit
- **Type**: `boolean`
- **Default**: `false`
- **Description**: Automatically commit changes
- **Example**: 
  ```yaml
  git:
    auto_commit: true
  ```

#### git.commit_message_template
- **Type**: `string`
- **Default**: `"VaultEnv: Update {environment} secrets"`
- **Description**: Template for commit messages
- **Example**: 
  ```yaml
  git:
    commit_message_template: "[vaultenv] {operation} in {environment}"
  ```

#### git.sync_on_change
- **Type**: `boolean`
- **Default**: `false`
- **Description**: Auto-sync with remote on changes
- **Example**: 
  ```yaml
  git:
    sync_on_change: true
  ```

#### git.ignored_environments
- **Type**: `array[string]`
- **Default**: `["local", "temp"]`
- **Description**: Environments to exclude from Git
- **Example**: 
  ```yaml
  git:
    ignored_environments:
      - development
      - test
  ```

### Security Settings

Security-related configuration options.

#### security.require_password_change
- **Type**: `integer` (days)
- **Default**: `0` (disabled)
- **Description**: Force password change after N days
- **Example**: 
  ```yaml
  security:
    require_password_change: 90
  ```

#### security.min_password_length
- **Type**: `integer`
- **Default**: `12`
- **Description**: Minimum password length
- **Example**: 
  ```yaml
  security:
    min_password_length: 16
  ```

#### security.password_complexity
- **Type**: `object`
- **Default**: All true
- **Description**: Password complexity requirements
- **Example**: 
  ```yaml
  security:
    password_complexity:
      require_uppercase: true
      require_lowercase: true
      require_numbers: true
      require_special: true
  ```

#### security.lockout_policy
- **Type**: `object`
- **Default**: 5 attempts, 15 min lockout
- **Description**: Failed authentication lockout
- **Example**: 
  ```yaml
  security:
    lockout_policy:
      max_attempts: 3
      lockout_duration: 1800  # 30 minutes
  ```

#### security.secure_delete
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Overwrite data before deletion
- **Example**: 
  ```yaml
  security:
    secure_delete: false
  ```

### UI and Output

Control display and output formatting.

#### ui.color
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable colored output
- **Example**: 
  ```yaml
  ui:
    color: false
  ```

#### ui.timestamp_format
- **Type**: `string`
- **Default**: `"2006-01-02 15:04:05"`
- **Description**: Timestamp format (Go format)
- **Example**: 
  ```yaml
  ui:
    timestamp_format: "Jan 2 15:04:05"
  ```

#### ui.table_style
- **Type**: `string`
- **Default**: `"simple"`
- **Options**: `"simple"`, `"rounded"`, `"heavy"`, `"none"`
- **Description**: Table border style
- **Example**: 
  ```yaml
  ui:
    table_style: rounded
  ```

#### ui.confirm_destructive
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Confirm destructive operations
- **Example**: 
  ```yaml
  ui:
    confirm_destructive: false
  ```

#### ui.output_format
- **Type**: `string`
- **Default**: `"text"`
- **Options**: `"text"`, `"json"`, `"yaml"`
- **Description**: Default output format
- **Example**: 
  ```yaml
  ui:
    output_format: json
  ```

### Performance

Performance tuning options.

#### performance.cache_enabled
- **Type**: `boolean`
- **Default**: `true`
- **Description**: Enable in-memory caching
- **Example**: 
  ```yaml
  performance:
    cache_enabled: false
  ```

#### performance.cache_ttl
- **Type**: `integer` (seconds)
- **Default**: `300`
- **Description**: Cache time-to-live
- **Example**: 
  ```yaml
  performance:
    cache_ttl: 600
  ```

#### performance.batch_size
- **Type**: `integer`
- **Default**: `100`
- **Description**: Batch operation size
- **Example**: 
  ```yaml
  performance:
    batch_size: 50
  ```

#### performance.parallel_operations
- **Type**: `integer`
- **Default**: `4`
- **Description**: Max parallel operations
- **Example**: 
  ```yaml
  performance:
    parallel_operations: 8
  ```

### Audit and Logging

Audit trail and logging configuration.

#### audit.enabled
- **Type**: `boolean`
- **Default**: `false`
- **Description**: Enable audit logging
- **Example**: 
  ```yaml
  audit:
    enabled: true
  ```

#### audit.log_file
- **Type**: `string`
- **Default**: `"${core.storage_path}/audit.log"`
- **Description**: Audit log file path
- **Example**: 
  ```yaml
  audit:
    log_file: /var/log/vaultenv/audit.log
  ```

#### audit.log_level
- **Type**: `string`
- **Default**: `"info"`
- **Options**: `"debug"`, `"info"`, `"warn"`, `"error"`
- **Description**: Logging verbosity
- **Example**: 
  ```yaml
  audit:
    log_level: debug
  ```

#### audit.log_format
- **Type**: `string`
- **Default**: `"json"`
- **Options**: `"json"`, `"text"`
- **Description**: Log entry format
- **Example**: 
  ```yaml
  audit:
    log_format: text
  ```

#### audit.retention_days
- **Type**: `integer`
- **Default**: `90`
- **Description**: Log retention period
- **Example**: 
  ```yaml
  audit:
    retention_days: 365
  ```

#### audit.include_values
- **Type**: `boolean`
- **Default**: `false`
- **Description**: Include secret values in logs
- **Example**: 
  ```yaml
  audit:
    include_values: true  # CAUTION: Security risk
  ```

## Environment Variable Overrides

Any configuration option can be overridden using environment variables. The format is:

```
VAULTENV_<SECTION>_<KEY>=value
```

### Examples

```bash
# Override default environment
export VAULTENV_CORE_DEFAULT_ENVIRONMENT=production

# Override encryption algorithm
export VAULTENV_ENCRYPTION_ALGORITHM=chacha20poly1305

# Override UI color setting
export VAULTENV_UI_COLOR=false

# Override audit logging
export VAULTENV_AUDIT_ENABLED=true
```

### Nested Configuration

For nested configuration, use double underscores:

```bash
# Override SQLite path
export VAULTENV_STORAGE_SQLITE__PATH=/custom/path.db

# Override password complexity
export VAULTENV_SECURITY_PASSWORD_COMPLEXITY__REQUIRE_SPECIAL=false
```

## Example Configurations

### Development Configuration

```yaml
# Development-focused configuration
version: 1
core:
  default_environment: development
  auto_lock_timeout: 3600  # 1 hour
encryption:
  algorithm: aes-256-gcm  # Faster for development
ui:
  confirm_destructive: false
  output_format: text
audit:
  enabled: false
```

### Production Configuration

```yaml
# Production-ready configuration
version: 1
core:
  default_environment: production
  auto_lock_timeout: 300  # 5 minutes
encryption:
  algorithm: chacha20poly1305
  key_derivation: argon2id
  key_derivation_params:
    time: 4
    memory: 131072  # 128MB
    parallelism: 8
storage:
  backend: sqlite
  compression: true
  sqlite:
    wal_mode: true
security:
  require_password_change: 90
  min_password_length: 16
  secure_delete: true
  lockout_policy:
    max_attempts: 3
    lockout_duration: 1800
git:
  enabled: true
  auto_commit: true
  sync_on_change: true
audit:
  enabled: true
  log_level: info
  retention_days: 365
ui:
  confirm_destructive: true
  output_format: json
```

### CI/CD Configuration

```yaml
# CI/CD optimized configuration
version: 1
core:
  auto_lock_timeout: 0  # No timeout
ui:
  color: false
  confirm_destructive: false
  output_format: json
performance:
  cache_enabled: false  # Ensure fresh data
audit:
  enabled: true
  log_level: debug
```

### High-Security Configuration

```yaml
# Maximum security configuration
version: 1
encryption:
  algorithm: chacha20poly1305
  key_derivation: argon2id
  key_derivation_params:
    time: 8
    memory: 262144  # 256MB
    parallelism: 16
security:
  require_password_change: 30
  min_password_length: 20
  password_complexity:
    require_uppercase: true
    require_lowercase: true
    require_numbers: true
    require_special: true
  secure_delete: true
  lockout_policy:
    max_attempts: 2
    lockout_duration: 3600  # 1 hour
audit:
  enabled: true
  log_level: debug
  include_values: false
  retention_days: 730  # 2 years
```

## Configuration Migration

When upgrading VaultEnv, configuration may need migration. VaultEnv handles this automatically but backs up the original.

### Automatic Migration

```bash
# Check if migration is needed
vaultenv config migrate --check

# Perform migration
vaultenv config migrate

# Migration with custom backup location
vaultenv config migrate --backup /safe/location/
```

### Manual Migration

For major version upgrades, manual migration might be required:

```yaml
# Old format (v1)
encryption:
  algorithm: aes256gcm

# New format (v2)
encryption:
  algorithm: aes-256-gcm
  version: 2
```

### Migration Best Practices

1. **Always backup** before migration
2. **Test in development** first
3. **Review changes** in migration log
4. **Validate** configuration after migration

```bash
# Full migration workflow
vaultenv config migrate --dry-run
vaultenv config backup
vaultenv config migrate
vaultenv config validate
```

## Configuration Validation

VaultEnv provides built-in configuration validation:

```bash
# Validate current configuration
vaultenv config validate

# Validate specific file
vaultenv config validate --file custom-config.yaml

# Validate with verbose output
vaultenv config validate --verbose
```

### Validation Rules

- Required fields must be present
- Values must be within acceptable ranges
- Paths must be accessible
- Algorithms must be supported
- Dependencies must be satisfied

## See Also

- [Command Reference](./COMMANDS.md) - Complete command documentation
- [Security Best Practices](../guides/SECURITY_BEST_PRACTICES.md) - Security configuration guide
- [File Formats](./FILE_FORMATS.md) - Configuration file format details