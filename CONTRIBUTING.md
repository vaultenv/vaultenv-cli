# Contributing to VaultEnv CLI 🎉

First off, **thank you** for considering contributing to VaultEnv! It's people like you that make VaultEnv such a great tool for the developer community. We're excited to have you aboard! 🚀

## 🌟 Why Contribute?

Your contributions help make secure secret management accessible to developers worldwide. Whether you're fixing a typo, adding a feature, or reporting a bug - every contribution matters and is appreciated.

## 🤝 Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [conduct@vaultenv.dev](mailto:conduct@vaultenv.dev).

## 💡 How Can I Contribute?

### 🎯 Types of Contributions We Love

#### 🐛 **Reporting Bugs**
Found something that doesn't work? We want to know! 

**Before creating a bug report:**
- Check the [existing issues](https://github.com/vaultenv/vaultenv-cli/issues) to avoid duplicates
- Collect relevant information (version, OS, steps to reproduce)

**Great bug reports include:**
```markdown
**Environment:**
- VaultEnv version: v0.1.0-beta.1
- OS: macOS 14.0
- Go version: 1.22

**Steps to Reproduce:**
1. Run `vaultenv init`
2. Set a variable with special characters: `vaultenv set KEY="value@#$"`
3. Try to retrieve it: `vaultenv get KEY`

**Expected:** Returns "value@#$"
**Actual:** Returns error: "invalid character in value"

**Additional Context:**
[Any logs, screenshots, or additional information]
```

#### ✨ **Suggesting Features**
Have an idea to make VaultEnv better? We'd love to hear it!

**Great feature requests include:**
- **Problem Statement**: What problem does this solve?
- **Proposed Solution**: How would it work?
- **Alternatives**: What other solutions did you consider?
- **Examples**: Mock commands or UI sketches

#### 📖 **Improving Documentation**
Documentation improvements are incredibly valuable! This includes:
- Fixing typos or clarifying confusing sections
- Adding examples or use cases
- Translating documentation
- Creating tutorials or blog posts

#### 🧪 **Adding Tests**
More tests = more confidence. We especially appreciate:
- Tests for edge cases
- Integration tests
- Performance benchmarks
- Security-focused tests

#### 🔧 **Submitting Code**
Ready to dive into the code? Awesome! Check out our development setup below.

## 🛠️ Development Setup

### Prerequisites

```bash
# Required
- Go 1.22 or higher
- Git
- Make

# Recommended
- golangci-lint (for linting)
- VS Code with Go extension
- Docker (for testing different environments)
```

### Getting Started

1. **Fork the repository**
   ```bash
   # Click the 'Fork' button on GitHub, then:
   git clone https://github.com/YOUR-USERNAME/vaultenv-cli.git
   cd vaultenv-cli
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/vaultenv/vaultenv-cli.git
   git fetch upstream
   ```

3. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

4. **Set up development environment**
   ```bash
   # Install dependencies
   go mod download
   
   # Build the project
   make build
   
   # Run tests
   make test
   
   # Run linter
   make lint
   ```

### 📁 Project Structure

```
vaultenv-cli/
├── cmd/vaultenv-cli/    # CLI entry point
├── internal/            # Private application code
│   ├── auth/           # Authentication & passwords
│   ├── cmd/            # Command implementations
│   ├── config/         # Configuration management
│   ├── keystore/       # Key storage
│   └── ui/             # Terminal UI helpers
├── pkg/                # Public packages
│   ├── encryption/     # Encryption implementations
│   ├── storage/        # Storage backends
│   └── export/         # Export formats
├── docs/               # Documentation
└── scripts/            # Build & test scripts
```

## 🎨 Code Style Guidelines

We follow standard Go conventions with some additions:

### Go Code Style

```go
// Good: Clear, idiomatic Go
func (s *Storage) Set(key, value string) error {
    if key == "" {
        return ErrEmptyKey
    }
    
    encrypted, err := s.encrypt(value)
    if err != nil {
        return fmt.Errorf("encryption failed: %w", err)
    }
    
    return s.backend.Write(key, encrypted)
}

// Bad: Non-idiomatic, unclear
func (s *Storage) Set(k string, v string) error {
    if k == "" { return ErrEmptyKey }
    enc, e := s.encrypt(v)
    if e != nil { return e }
    return s.backend.Write(k, enc)
}
```

### Best Practices

1. **Error Handling**
   - Always check errors
   - Wrap errors with context using `fmt.Errorf`
   - Define sentinel errors for common cases

2. **Testing**
   - Write table-driven tests
   - Test edge cases and error paths
   - Use meaningful test names

3. **Documentation**
   - Document all exported types and functions
   - Include examples in doc comments
   - Keep comments up-to-date with code

4. **Security**
   - Never log sensitive data
   - Always use constant-time comparisons for secrets
   - Validate all inputs

## 📝 Commit Message Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code changes that neither fix bugs nor add features
- `perf`: Performance improvements
- `test`: Test additions or corrections
- `chore`: Maintenance tasks

**Examples:**
```bash
# Feature
feat(storage): add SQLite backend support

# Bug fix
fix(encryption): handle special characters in passwords

# Documentation
docs(readme): add installation instructions for Windows

# Breaking change
feat(api)!: rename Storage.Put to Storage.Set

BREAKING CHANGE: Storage.Put has been renamed to Storage.Set
for consistency with other methods
```

## 🔄 Pull Request Process

1. **Before submitting:**
   - ✅ Run tests: `make test`
   - ✅ Run linter: `make lint`
   - ✅ Update documentation if needed
   - ✅ Add tests for new functionality
   - ✅ Ensure commits follow our guidelines

2. **PR checklist:**
   ```markdown
   ## Description
   Brief description of changes
   
   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update
   
   ## Testing
   - [ ] Tests pass locally
   - [ ] Added new tests
   - [ ] Tested manually
   
   ## Checklist
   - [ ] My code follows the project style
   - [ ] I've updated documentation
   - [ ] I've added tests
   - [ ] All tests pass
   - [ ] I've run the linter
   ```

3. **After submitting:**
   - Respond to review feedback promptly
   - Update your branch with upstream changes if needed
   - Be patient - maintainers review PRs as time permits

## 🧪 Testing Guidelines

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific tests
go test ./pkg/encryption/...

# Run tests with race detection
go test -race ./...

# Run integration tests
make test-integration
```

### Writing Tests

```go
func TestStorage_Set(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        value   string
        wantErr bool
    }{
        {
            name:  "valid key and value",
            key:   "API_KEY",
            value: "secret123",
        },
        {
            name:    "empty key",
            key:     "",
            value:   "value",
            wantErr: true,
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := NewStorage()
            err := s.Set(tt.key, tt.value)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## 📚 Additional Resources

- **[Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)** - Go team's style guide
- **[Effective Go](https://golang.org/doc/effective_go.html)** - Writing clear, idiomatic Go
- **[Security Best Practices](./docs/guides/SECURITY_BEST_PRACTICES.md)** - Keep VaultEnv secure

## 🎖️ Recognition

We believe in recognizing all contributions:

- **Contributors** are listed in our [README](README.md)
- **Significant contributors** get mentioned in release notes
- **Security researchers** are credited in security advisories
- **Documentation contributors** are acknowledged in docs

## 💬 Getting Help

- **Discord**: Join our [community server](https://discord.gg/vaultenv)
- **Discussions**: Use [GitHub Discussions](https://github.com/vaultenv/vaultenv-cli/discussions) for questions
- **Issues**: Check [existing issues](https://github.com/vaultenv/vaultenv-cli/issues) or create new ones

## 🚀 What Happens Next?

After you submit a contribution:

1. **Automated checks** run (tests, linting, security scans)
2. **Maintainer review** (usually within 48-72 hours)
3. **Feedback/approval** process
4. **Merge** and celebration! 🎉

## ❤️ Thank You!

Your contributions make VaultEnv better for everyone. Whether it's your first open source contribution or your thousandth, we're grateful for your time and effort.

Happy coding! 🚀

---

<div align="center">

**Questions?** Reach out on [Discord](https://discord.gg/vaultenv) or [open a discussion](https://github.com/vaultenv/vaultenv-cli/discussions)

</div>