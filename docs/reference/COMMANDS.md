# VaultEnv Command Reference

This document provides a comprehensive reference for all VaultEnv CLI commands and options.

## Table of Contents

- [Global Flags](#global-flags)
- [Core Commands](#core-commands)
  - [vaultenv init](#vaultenv-init)
  - [vaultenv set](#vaultenv-set)
  - [vaultenv get](#vaultenv-get)
  - [vaultenv list](#vaultenv-list)
  - [vaultenv export](#vaultenv-export)
  - [vaultenv load](#vaultenv-load)
  - [vaultenv execute](#vaultenv-execute)
- [Environment Commands](#environment-commands)
  - [vaultenv env](#vaultenv-env)
- [Git Integration Commands](#git-integration-commands)
  - [vaultenv git](#vaultenv-git)
- [Configuration Commands](#configuration-commands)
  - [vaultenv config](#vaultenv-config)
- [Utility Commands](#utility-commands)
  - [vaultenv shell](#vaultenv-shell)
  - [vaultenv completion](#vaultenv-completion)
  - [vaultenv version](#vaultenv-version)
  - [vaultenv migrate](#vaultenv-migrate)
  - [vaultenv history](#vaultenv-history)
  - [vaultenv aliases](#vaultenv-aliases)
  - [vaultenv batch](#vaultenv-batch)
  - [vaultenv security](#vaultenv-security)

## Global Flags

These flags can be used with any VaultEnv command:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--env` | `-e` | Target environment name | Current environment |
| `--config` | `-c` | Config file path | `~/.vaultenv/config.yaml` |
| `--verbose` | `-v` | Enable verbose output | `false` |
| `--quiet` | `-q` | Suppress non-error output | `false` |
| `--no-color` | | Disable colored output | `false` |
| `--help` | `-h` | Show help for command | |

## Core Commands

### vaultenv init

Initialize a new VaultEnv environment or reinitialize an existing one.

#### Synopsis
```bash
vaultenv init [ENVIRONMENT] [flags]
```

#### Examples
```bash
# Initialize default environment
vaultenv init

# Initialize specific environment
vaultenv init production

# Initialize with custom storage path
vaultenv init --path /custom/path/secrets.db

# Initialize from existing .env file
vaultenv init --from .env.example
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--path` | `-p` | Custom storage path |
| `--from` | `-f` | Import from .env file |
| `--force` | | Overwrite existing environment |

### vaultenv set

Set one or more environment variables in the current or specified environment.

#### Synopsis
```bash
vaultenv set KEY=VALUE [KEY=VALUE...] [flags]
```

#### Examples
```bash
# Set a single variable
vaultenv set DATABASE_URL=postgres://localhost/mydb

# Set multiple variables
vaultenv set API_KEY=key123 API_SECRET=secret456

# Set in specific environment
vaultenv set NODE_ENV=production --env production

# Set from stdin
echo "SECRET_KEY=mysecret" | vaultenv set -

# Set with force overwrite
vaultenv set API_KEY=newkey --force
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Overwrite without confirmation |
| `--stdin` | | Read from standard input |

### vaultenv get

Retrieve one or more environment variables.

#### Synopsis
```bash
vaultenv get KEY [KEY...] [flags]
```

#### Examples
```bash
# Get single variable
vaultenv get DATABASE_URL

# Get multiple variables
vaultenv get API_KEY API_SECRET

# Get with export format
vaultenv get DATABASE_URL --export

# Get from specific environment
vaultenv get NODE_ENV --env production

# Get and decrypt (show actual value)
vaultenv get API_KEY --decrypt
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--export` | | Output in export format |
| `--decrypt` | `-d` | Show decrypted values |
| `--json` | `-j` | Output as JSON |

### vaultenv list

List all environment variables in the current or specified environment.

#### Synopsis
```bash
vaultenv list [flags]
```

#### Examples
```bash
# List all variables
vaultenv list

# List with values
vaultenv list --decrypt

# List in JSON format
vaultenv list --json

# List for specific environment
vaultenv list --env production

# List matching pattern
vaultenv list --filter "API_*"
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--decrypt` | `-d` | Show decrypted values |
| `--json` | `-j` | Output as JSON |
| `--filter` | | Filter by pattern |
| `--keys-only` | `-k` | Show only keys |

### vaultenv export

Export environment variables in various formats.

#### Synopsis
```bash
vaultenv export [flags]
```

#### Examples
```bash
# Export as .env file
vaultenv export > .env

# Export as JSON
vaultenv export --format json > secrets.json

# Export for Docker
vaultenv export --format docker > docker.env

# Export specific environment
vaultenv export --env production --format yaml

# Export with prefix
vaultenv export --prefix "MYAPP_"
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--format` | `-f` | Output format (dotenv, json, yaml, docker, shell) |
| `--output` | `-o` | Output file path |
| `--prefix` | | Add prefix to all keys |
| `--exclude` | | Exclude patterns |

### vaultenv load

Load environment variables from a file.

#### Synopsis
```bash
vaultenv load FILE [flags]
```

#### Examples
```bash
# Load from .env file
vaultenv load .env

# Load with merge strategy
vaultenv load .env.new --merge

# Load and overwrite all
vaultenv load .env.production --force

# Load into specific environment
vaultenv load staging.env --env staging

# Load from URL
vaultenv load https://config.example.com/env
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--merge` | `-m` | Merge with existing variables |
| `--force` | `-f` | Overwrite all variables |
| `--format` | | Input format (auto-detect by default) |

### vaultenv execute

Execute a command with environment variables loaded.

#### Synopsis
```bash
vaultenv execute [--] COMMAND [ARGS...] [flags]
```

#### Examples
```bash
# Run Node.js app
vaultenv execute node app.js

# Run with specific environment
vaultenv execute --env production npm start

# Run with additional variables
vaultenv execute --set PORT=3000 python app.py

# Complex command with flags
vaultenv execute -- docker run -p 8080:8080 myapp
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--set` | `-s` | Set additional variables |
| `--inherit` | `-i` | Inherit system environment |

## Environment Commands

### vaultenv env

Manage VaultEnv environments.

#### Synopsis
```bash
vaultenv env SUBCOMMAND [flags]
```

#### Subcommands

##### env list
List all available environments.

```bash
# List environments
vaultenv env list

# List with details
vaultenv env list --details
```

##### env create
Create a new environment.

```bash
# Create environment
vaultenv env create staging

# Create with description
vaultenv env create production --description "Production environment"
```

##### env copy
Copy an environment.

```bash
# Copy environment
vaultenv env copy development staging

# Copy with new password
vaultenv env copy production production-backup --new-password
```

##### env remove
Remove an environment.

```bash
# Remove environment
vaultenv env remove old-staging

# Remove without confirmation
vaultenv env remove temp --force
```

##### env switch
Switch to a different environment.

```bash
# Switch environment
vaultenv env switch production

# Switch with auto-create
vaultenv env switch new-env --create
```

##### env current
Show current environment.

```bash
vaultenv env current
```

##### env change-password
Change environment password.

```bash
# Change password interactively
vaultenv env change-password production

# Change with password file
vaultenv env change-password staging --password-file new-pass.txt
```

## Git Integration Commands

### vaultenv git

Git integration commands for team synchronization.

#### Synopsis
```bash
vaultenv git SUBCOMMAND [flags]
```

#### Subcommands

##### git init
Initialize Git integration.

```bash
# Initialize in current repo
vaultenv git init

# Initialize with remote
vaultenv git init --remote origin
```

##### git sync
Synchronize with Git repository.

```bash
# Sync all environments
vaultenv git sync

# Sync specific environment
vaultenv git sync --env production

# Force sync (overwrite local)
vaultenv git sync --force
```

##### git push
Push encrypted files to Git.

```bash
# Push current environment
vaultenv git push

# Push all environments
vaultenv git push --all
```

##### git pull
Pull encrypted files from Git.

```bash
# Pull and merge
vaultenv git pull

# Pull specific branch
vaultenv git pull --branch feature/new-secrets
```

##### git status
Show Git synchronization status.

```bash
vaultenv git status
```

## Configuration Commands

### vaultenv config

Manage VaultEnv configuration.

#### Synopsis
```bash
vaultenv config SUBCOMMAND [flags]
```

#### Subcommands

##### config get
Get configuration value.

```bash
# Get specific value
vaultenv config get encryption.algorithm

# Get all values
vaultenv config get
```

##### config set
Set configuration value.

```bash
# Set encryption algorithm
vaultenv config set encryption.algorithm chacha20poly1305

# Set with validation
vaultenv config set audit.enabled true --validate
```

##### config list
List all configuration options.

```bash
vaultenv config list
```

##### config reset
Reset configuration to defaults.

```bash
# Reset all
vaultenv config reset

# Reset specific section
vaultenv config reset encryption
```

## Utility Commands

### vaultenv shell

Start an interactive shell with environment variables loaded.

#### Synopsis
```bash
vaultenv shell [flags]
```

#### Examples
```bash
# Start default shell
vaultenv shell

# Start specific shell
vaultenv shell --shell zsh

# Start with specific environment
vaultenv shell --env production
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--shell` | `-s` | Shell to use (bash, zsh, fish) |
| `--inherit` | `-i` | Inherit system environment |

### vaultenv completion

Generate shell completion scripts.

#### Synopsis
```bash
vaultenv completion SHELL [flags]
```

#### Examples
```bash
# Bash completion
vaultenv completion bash > ~/.vaultenv-completion.bash
echo "source ~/.vaultenv-completion.bash" >> ~/.bashrc

# Zsh completion
vaultenv completion zsh > ~/.vaultenv-completion.zsh
echo "source ~/.vaultenv-completion.zsh" >> ~/.zshrc

# Fish completion
vaultenv completion fish > ~/.config/fish/completions/vaultenv.fish
```

### vaultenv version

Display version information.

#### Synopsis
```bash
vaultenv version [flags]
```

#### Examples
```bash
# Show version
vaultenv version

# Show detailed version info
vaultenv version --verbose
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Show detailed version info |
| `--check-update` | | Check for updates |

### vaultenv migrate

Migrate from other secret management tools.

#### Synopsis
```bash
vaultenv migrate SOURCE [flags]
```

#### Examples
```bash
# Migrate from .env file
vaultenv migrate dotenv --file .env

# Migrate from HashiCorp Vault
vaultenv migrate vault --path secret/myapp

# Migrate from AWS Secrets Manager
vaultenv migrate aws --secret-id prod/myapp

# Migrate with mapping
vaultenv migrate dotenv --map old_key=new_key
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--file` | `-f` | Source file path |
| `--map` | `-m` | Key mapping (old=new) |
| `--dry-run` | | Preview migration |

### vaultenv history

View operation history and audit logs.

#### Synopsis
```bash
vaultenv history [flags]
```

#### Examples
```bash
# View recent history
vaultenv history

# View specific environment history
vaultenv history --env production

# Filter by operation type
vaultenv history --operation set

# Export as JSON
vaultenv history --format json --since "2024-01-01"
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--limit` | `-l` | Number of entries |
| `--since` | | Start date |
| `--until` | | End date |
| `--operation` | `-o` | Filter by operation |
| `--format` | `-f` | Output format |

### vaultenv aliases

Manage command aliases.

#### Synopsis
```bash
vaultenv aliases SUBCOMMAND [flags]
```

#### Examples
```bash
# List aliases
vaultenv aliases list

# Create alias
vaultenv aliases set prod "env switch production"

# Remove alias
vaultenv aliases remove prod

# Execute alias
vaultenv prod  # Switches to production
```

### vaultenv batch

Execute batch operations from file.

#### Synopsis
```bash
vaultenv batch FILE [flags]
```

#### Examples
```bash
# Execute batch file
vaultenv batch operations.txt

# Dry run
vaultenv batch operations.txt --dry-run

# With transaction
vaultenv batch updates.txt --transaction
```

#### Flags
| Flag | Short | Description |
|------|-------|-------------|
| `--dry-run` | | Preview operations |
| `--transaction` | `-t` | Run as transaction |
| `--stop-on-error` | | Stop on first error |

### vaultenv security

Security utilities and checks.

#### Synopsis
```bash
vaultenv security SUBCOMMAND [flags]
```

#### Subcommands

##### security scan
Scan for security issues.

```bash
# Scan current environment
vaultenv security scan

# Scan all environments
vaultenv security scan --all

# Scan with fixes
vaultenv security scan --fix
```

##### security rotate
Rotate encryption keys.

```bash
# Rotate current environment
vaultenv security rotate

# Rotate all environments
vaultenv security rotate --all
```

##### security audit
Generate security audit report.

```bash
# Generate audit report
vaultenv security audit

# Export as PDF
vaultenv security audit --format pdf --output audit.pdf
```

## See Also

- [Configuration Reference](./CONFIGURATION.md) - Detailed configuration options
- [File Formats](./FILE_FORMATS.md) - File format specifications
- [Security Best Practices](../guides/SECURITY_BEST_PRACTICES.md) - Security guidelines