# VaultEnv CLI - Zero-Knowledge Secrets Management for Modern Development Teams 🔐

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue)](https://go.dev)
[![Beta Version](https://img.shields.io/badge/version-v0.1.0--beta.1-orange)](https://github.com/vaultenv/vaultenv-cli/releases)
[![Test Coverage](https://img.shields.io/badge/coverage-60%25-yellow)](./docs/TEST_COVERAGE_REPORT.md)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/vaultenv/vaultenv-cli)](https://goreportcard.com/report/github.com/vaultenv/vaultenv-cli)
[![Security: Zero-Knowledge](https://img.shields.io/badge/Security-Zero--Knowledge-brightgreen)](./SECURITY.md)

> 🚀 **The secure, Git-friendly environment variable manager that developers actually love using**

VaultEnv revolutionizes how development teams manage secrets and environment variables. With military-grade encryption, seamless Git integration, and a delightful developer experience, it's the tool you've been waiting for since EnvKey's shutdown.

## ✨ Why VaultEnv?

**🔐 True Zero-Knowledge Architecture** - Your secrets are encrypted client-side. We can't read them, even if we wanted to.

**⚡ 5-Minute Setup** - From install to your first encrypted secret in under 5 minutes. No complex configurations, no steep learning curve.

**🔄 Git-Native Workflow** - Encrypted .vaultenv files live in your repo. Branch, merge, and collaborate without fear.

**🎯 Developer-First Design** - Intuitive commands, helpful error messages, and shortcuts for common tasks. Built by developers, for developers.

## 🚀 Quick Start

### Installation

```bash
# macOS/Linux (via Homebrew)
brew tap vaultenv/tap
brew install vaultenv-cli

# macOS/Linux (via curl)
curl -sSL https://install.vaultenv.dev | bash

# Windows (via Scoop)
scoop bucket add vaultenv https://github.com/vaultenv/scoop-bucket
scoop install vaultenv-cli

# Go developers
go install github.com/vaultenv/vaultenv-cli/cmd/vaultenv-cli@latest

# Docker
docker run -it vaultenv/cli:latest
```

### Your First Secret (< 60 seconds)

```bash
# Initialize your project
vaultenv init

# Set your first secret
vaultenv set DATABASE_URL="postgres://localhost/myapp"

# Use it in your application
vaultenv run -- npm start

# Share with your team (secrets are encrypted!)
git add .vaultenv
git commit -m "Add encrypted environment config"
git push
```

## 🎯 Core Features

### 🔐 **Bank-Grade Security**

- **AES-256-GCM** encryption with **Argon2id** key derivation
- **Per-environment** encryption keys
- **OS keychain** integration for secure key storage
- **Zero-knowledge** - your secrets never leave your machine unencrypted

### 🚀 **Developer Experience**

```bash
# Intuitive commands that just make sense
vaultenv set API_KEY="sk-123" --env production
vaultenv get API_KEY
vaultenv list --env staging
vaultenv run -- cargo test

# Import existing .env files in seconds
vaultenv import .env --env development

# Export for CI/CD
vaultenv export --format dotenv > .env.production
```

### 🔄 **Team Collaboration**

```bash
# Git-friendly deterministic encryption
vaultenv set --deterministic SHARED_KEY="value"

# Sync with your team
git pull
vaultenv sync

# Handle conflicts like a pro
vaultenv conflicts resolve
```

### 📊 **Enterprise Ready**

- **Audit trails** with SQLite backend
- **Access control** per environment
- **Compliance-friendly** export formats
- **Migration tools** for easy adoption

## 🆚 Why Choose VaultEnv?

| Feature           | VaultEnv           | .env Files    | direnv        | dotenv-vault         |
| ----------------- | ------------------ | ------------- | ------------- | -------------------- |
| **Encryption**    | ✅ Zero-knowledge  | ❌ Plaintext  | ❌ Plaintext  | ✅ Basic             |
| **Git-Safe**      | ✅ Encrypted files | ❌ .gitignore | ❌ .gitignore | ⚠️ Separate          |
| **Team Sync**     | ✅ Built-in        | ❌ Manual     | ❌ Manual     | ✅ Cloud-only        |
| **Audit Trail**   | ✅ Complete        | ❌ None       | ❌ None       | ⚠️ Limited           |
| **Offline-First** | ✅ Always works    | ✅ Yes        | ✅ Yes        | ❌ Requires internet |
| **Free & Open**   | ✅ MIT License     | ✅ N/A        | ✅ MIT        | ⚠️ Freemium          |

## 📚 Documentation

- 📖 **[Getting Started Guide](./docs/guides/GETTING_STARTED.md)** - Your journey begins here
- 🔐 **[Security Architecture](./docs/ARCHITECTURE.md#security)** - How we keep your secrets safe
- 🔧 **[CLI Reference](./docs/reference/COMMANDS.md)** - Complete command documentation
- 🚀 **[Migration Guide](./docs/guides/MIGRATION_GUIDE.md)** - Switch from .env files or other tools
- 👥 **[Team Collaboration](./docs/guides/TEAM_COLLABORATION.md)** - Work together securely
- 🛠️ **[API Documentation](./docs/API.md)** - Build on top of VaultEnv

## 🤝 Contributing

VaultEnv is open source and we love contributions! Whether you're fixing bugs, adding features, improving docs, or spreading the word - we appreciate your help.

Check out our **[Contributing Guide](CONTRIBUTING.md)** to get started. We welcome:

- 🐛 Bug reports and fixes
- ✨ Feature suggestions and implementations
- 📖 Documentation improvements
- 🧪 Test coverage improvements
- 🌐 Translations
- 💡 Ideas and feedback

## 🔒 Security

Security is our top priority. VaultEnv uses industry-standard encryption and follows security best practices.

- 🔍 **[Security Policy](SECURITY.md)** - How to report vulnerabilities
- 🛡️ **[Security Best Practices](./docs/guides/SECURITY_BEST_PRACTICES.md)** - Keep your secrets safe
- 🔐 **[Encryption Details](./docs/ARCHITECTURE.md#encryption)** - Technical implementation

Found a security issue? Please check our **[Security Policy](SECURITY.md)** for responsible disclosure.

## 🚀 What's Next?

### Coming Soon (v0.2.0)

- 🔌 Plugin system for custom integrations
- 🌍 Multi-region support
- 📱 Mobile companion app
- 🤖 GitHub Actions integration

### VaultEnv Cloud (Coming Q4 2025)

Need enterprise features? **[Join the waitlist](https://vaultenv.dev/cloud)** for:

- ☁️ Automatic team synchronization
- 📊 Advanced audit logging
- 🔐 SSO/SAML integration
- 👥 Role-based access control
- 📞 Priority support

## 📄 License

VaultEnv CLI is open source under the **[MIT License](LICENSE)**. Use it freely in personal and commercial projects.

## 🙏 Acknowledgments

Built with amazing open source projects:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [SQLite](https://sqlite.org) - Embedded database
- [x/crypto](https://pkg.go.dev/golang.org/x/crypto) - Cryptography

Special thanks to all our **[contributors](https://github.com/vaultenv/vaultenv-cli/graphs/contributors)**!

---

<div align="center">
  
**[Website](https://vaultenv.dev)** • **[Documentation](https://docs.vaultenv.dev)** • **[Discord](https://discord.gg/vaultenv)** • **[Twitter](https://twitter.com/vaultenv)**

Made with ❤️ by developers who were tired of managing .env files

</div>
