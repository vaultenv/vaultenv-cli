# vaultenv-cli - Secure Environment Variable Management 🔐

[![CI Status](https://github.com/vaultenv/vaultenv-cli/workflows/CI/badge.svg)](https://github.com/vaultenv/vaultenv-cli/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/vaultenv/vaultenv-cli)](https://goreportcard.com/report/github.com/vaultenv/vaultenv-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

vaultenv-cli makes managing environment variables across different environments as simple as a single command, while maintaining bank-level security.

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

## 🔐 Security First

- **Zero-Knowledge Architecture**: Your secrets are encrypted client-side. We can't read them even if we wanted to.
- **OS Keychain Integration**: Encryption keys are stored in your system's secure keystore.
- **Open Source**: All security-critical code is open source and auditable.

## 🛠️ Development

### Prerequisites

- Go 1.22 or higher
- Make

### Building from Source

```bash
# Clone the repository
git clone https://github.com/vaultenv/vaultenv-cli.git
cd vaultenv-cli

# Install development tools
make setup

# Build the binary
make build

# Run tests
make test

# Run linters
make lint
```

### Project Structure

```
vaultenv-cli/
├── cmd/vaultenv-cli/    # Application entry point
├── internal/            # Private application code
│   ├── cmd/            # Command implementations
│   ├── config/         # Configuration management
│   ├── ui/             # Terminal UI components
│   └── errors/         # Error handling
├── pkg/                # Public packages
│   ├── encryption/     # Encryption implementations
│   ├── storage/        # Storage backends
│   └── types/          # Shared types
├── scripts/            # Build and dev scripts
└── docs/              # Documentation
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