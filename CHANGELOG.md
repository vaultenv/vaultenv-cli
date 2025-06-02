# Changelog

All notable changes to vaultenv-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-beta.1] - 2025-01-06

### Added
- Initial release of vaultenv-cli
- Core environment variable management commands:
  - `init` - Initialize a new vaultenv project
  - `set` - Set environment variables
  - `get` - Retrieve environment variable values
  - `list` - List all environment variables
  - `delete` - Remove environment variables
  - `export` - Export variables in various formats
- Multiple storage backends:
  - File-based storage (default)
  - SQLite storage for advanced features
  - Git storage for version control integration
- Strong encryption support:
  - AES-256-GCM encryption
  - ChaCha20-Poly1305 as alternative
  - Per-environment password protection
  - OS keychain integration for password storage
- Import/Export functionality:
  - .env file format
  - JSON format
  - YAML format
  - TOML format
  - Shell export format
- Advanced features:
  - Environment management (dev, staging, production, etc.)
  - Command aliases for common operations
  - Batch operations for multiple variables
  - Git synchronization for team collaboration
  - Audit history tracking (SQLite backend)
  - Migration tools for backend switching
- Shell integration:
  - Bash completion
  - Zsh completion
  - Fish completion
  - PowerShell completion
- Security features:
  - Zero-knowledge encryption (server never sees plaintext)
  - Secure password handling
  - Environment isolation
  - Access control per environment
- Developer experience:
  - Intuitive CLI with helpful error messages
  - Comprehensive help documentation
  - Quick start guide in README
  - Example usage for common scenarios

### Security
- All encryption happens client-side
- Passwords never stored in plaintext
- Git-friendly encrypted file format

[Unreleased]: https://github.com/vaultenv/vaultenv-cli/compare/v0.1.0-beta.1...HEAD
[0.1.0-beta.1]: https://github.com/vaultenv/vaultenv-cli/releases/tag/v0.1.0-beta.1