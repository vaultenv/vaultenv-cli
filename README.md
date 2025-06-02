# VaultEnv CLI - Secure Environment Variable Management 🔐

[![Beta Version](https://img.shields.io/badge/version-v0.1.0--beta.1-orange)](https://github.com/vaultenv/vaultenv-cli/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vaultenv/vaultenv-cli)](https://go.dev)
[![Test Coverage](https://img.shields.io/badge/coverage-56.5%25-yellow)](./coverage.out)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

> ⚠️ **Beta Release**: This is a beta version. While core features are stable, some functionality may change before the 1.0 release.

VaultEnv CLI is a secure, developer-friendly command-line tool for managing environment variables across different environments. Built with zero-knowledge encryption, it ensures your secrets remain secure while providing a seamless development experience.

## 🚀 Quick Start

### Installation

```bash
# macOS
brew install vaultenv/tap/vaultenv-cli

# Linux/Windows
curl -sSL https://install.vaultenv.io | bash

# Go developers
go install github.com/vaultenv/vaultenv-cli/cmd/vaultenv-cli@latest
```

### First Steps

```bash
# Initialize a new project
vaultenv-cli init

# Set a variable
vaultenv-cli set DATABASE_URL=postgres://localhost/myapp

# Get a variable
vaultenv-cli get DATABASE_URL

# List all variables
vaultenv-cli list

# Switch environments
vaultenv-cli set API_KEY=prod-key --env production
```

## ✨ Key Features

- **🔐 Zero-Knowledge Encryption**: Client-side AES-256-GCM encryption with Argon2id key derivation
- **🔑 Per-Environment Passwords**: Separate encryption keys for each environment
- **📁 Multiple Storage Backends**: File, SQLite, and Git-friendly storage options
- **🔄 Git Integration**: Deterministic encryption mode for clean diffs
- **📋 Import/Export**: Support for .env, JSON, YAML, and Docker formats
- **🔍 Audit Trail**: Complete history tracking with SQLite backend
- **🔒 OS Keychain Integration**: Secure key storage using system keychains
- **🎯 Developer Friendly**: Intuitive commands and helpful error messages

## 🛠️ Development

### Prerequisites

- Go 1.22 or higher
- Make

### Building from Source

```bash
# Clone the repository
git clone https://github.com/vaultenv/vaultenv-cli.git
cd vaultenv-cli

# Build the binary
go build -o vaultenv ./cmd/vaultenv-cli

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Install locally
go install ./cmd/vaultenv-cli
```

### Project Structure

```
vaultenv-cli/
├── cmd/vaultenv-cli/    # Application entry point
├── internal/            # Private application code
│   ├── auth/           # Password management and authentication
│   ├── cmd/            # Command implementations
│   ├── config/         # Configuration management
│   ├── keystore/       # Encryption key storage
│   ├── ui/             # Terminal UI components
│   └── test/           # Test helpers
├── pkg/                # Public packages
│   ├── access/         # Access control
│   ├── dotenv/         # .env file parsing
│   ├── encryption/     # Encryption implementations
│   ├── export/         # Export format handlers
│   ├── keystore/       # OS keychain integration
│   └── storage/        # Storage backends
├── docs/               # Documentation
└── scripts/            # Build and test scripts
```

## 🤝 Contributing

We love contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

Quick ways to contribute:

- 🐛 Report bugs
- 💡 Suggest features
- 📖 Improve documentation
- 🔧 Submit pull requests

## 📚 Documentation

- [Getting Started Guide](https://docs.vaultenv.io/getting-started)
- [Security Architecture](https://docs.vaultenv.io/security)
- [API Reference](https://docs.vaultenv.io/api)
- [Examples](https://github.com/vaultenv/vaultenv-cli/tree/main/examples)

## 🏢 Commercial Support

Need enterprise features? Check out [vaultenv Cloud](https://vaultenv.io) for:

- Team management and collaboration
- Audit logging and compliance
- SSO/SAML integration
- Priority support

## 📄 License

vaultenv-cli is MIT licensed. See [LICENSE](LICENSE) file for details.

---

Built with ❤️ by developers, for developers.