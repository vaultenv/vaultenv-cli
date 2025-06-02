# Contributing to VaultEnv CLI

Thank you for your interest in contributing to VaultEnv CLI! This document provides guidelines and instructions for contributing.

## 🚀 Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/your-username/vaultenv-cli.git
   cd vaultenv-cli
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/vaultenv/vaultenv-cli.git
   ```

## 🛠️ Development Setup

### Prerequisites

- Go 1.22 or higher
- Git
- A code editor (we recommend VS Code with the Go extension)

### Building the Project

```bash
# Build the binary
go build -o vaultenv ./cmd/vaultenv-cli

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
```

## 📝 Making Changes

### Code Style

We follow standard Go conventions:
- Run `gofmt` on all code
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Write clear, idiomatic Go code
- Add comments for exported functions and types

### Commit Messages

We use conventional commits:
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Test additions or modifications
- `refactor:` Code refactoring
- `chore:` Build process or auxiliary tool changes

Example:
```
feat: add support for YAML import format

- Parse YAML files in the import command
- Add comprehensive tests for YAML parsing
- Update documentation
```

### Testing

- Write tests for all new functionality
- Maintain or improve test coverage
- Run the full test suite before submitting PR
- Test files should be in the same package with `_test.go` suffix

## 🔄 Pull Request Process

1. Create a new branch for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit them with clear messages

3. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

4. Open a Pull Request against the `main` branch

5. Ensure all checks pass:
   - Tests pass
   - Code is formatted
   - No linting errors

6. Respond to review feedback promptly

## 🐛 Reporting Issues

### Bug Reports

Include:
- VaultEnv version (`vaultenv --version`)
- Go version (`go version`)
- Operating system and version
- Steps to reproduce
- Expected vs actual behavior
- Any error messages or logs

### Feature Requests

Include:
- Clear description of the feature
- Use cases and benefits
- Examples of how it would work
- Any potential drawbacks or considerations

## 🏗️ Architecture Decisions

### Security First

- All encryption must happen client-side
- Never log sensitive information
- Use established cryptographic libraries
- Follow security best practices

### User Experience

- Clear, helpful error messages
- Intuitive command structure
- Fast response times
- Comprehensive documentation

### Code Organization

- `cmd/` - Entry points
- `internal/` - Private application code
- `pkg/` - Public, reusable packages
- Keep interfaces small and focused
- Prefer composition over inheritance

## 📚 Resources

- [Go Documentation](https://golang.org/doc/)
- [Go Testing Guide](https://golang.org/doc/tutorial/add-a-test)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

## 💬 Getting Help

- Open an issue for bugs or features
- Join our [Discord community](https://discord.gg/vaultenv)
- Check existing issues and PRs before creating new ones

## 📄 License

By contributing to VaultEnv CLI, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to VaultEnv CLI! 🎉