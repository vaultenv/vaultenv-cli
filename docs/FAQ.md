# VaultEnv Frequently Asked Questions (FAQ)

This document answers common questions about VaultEnv. For technical issues, see the [Troubleshooting Guide](./TROUBLESHOOTING.md).

## Table of Contents

- [General Questions](#general-questions)
- [Security Questions](#security-questions)
- [Usage Questions](#usage-questions)
- [Integration Questions](#integration-questions)
- [Comparison Questions](#comparison-questions)
- [Technical Questions](#technical-questions)
- [Licensing and Support](#licensing-and-support)

## General Questions

### What is VaultEnv?

VaultEnv is a developer-focused command-line tool for secure environment variable and secrets management. It provides:

- **Zero-knowledge encryption**: Your secrets are encrypted locally before storage
- **Git-friendly workflow**: Encrypted files can be safely committed to version control
- **Team collaboration**: Share secrets securely with your team
- **Multiple environments**: Separate configurations for development, staging, and production
- **Cross-platform**: Works on Linux, macOS, and Windows

### How is VaultEnv different from .env files?

Traditional `.env` files have several limitations that VaultEnv addresses:

| Feature | .env Files | VaultEnv |
|---------|-----------|----------|
| Security | Plain text | AES-256-GCM encryption |
| Git Safety | Must add to .gitignore | Safe to commit |
| Team Sharing | Manual copying | Encrypted sync |
| Access Control | None | Password-protected environments |
| History | None | Full audit trail |
| Multiple Environments | Multiple files | Built-in environment support |

### Is VaultEnv suitable for production use?

Yes, VaultEnv is designed with production requirements in mind:

- **Battle-tested encryption**: Uses industry-standard AES-256-GCM
- **Zero-knowledge architecture**: Servers never see your plaintext secrets
- **Audit logging**: Complete history of all changes
- **High performance**: Optimized for large-scale deployments
- **Enterprise features**: SSO, compliance reports, and more (Pro version)

Many teams use VaultEnv in production to manage critical secrets and configurations.

### How much does VaultEnv cost?

VaultEnv offers multiple tiers:

- **Open Source (Free)**: Core CLI tool with full encryption capabilities
- **Pro ($49/month)**: Team features, sync, audit logs, priority support
- **Enterprise (Custom)**: SSO, compliance, SLA, dedicated support

The open-source version is fully functional for local secret management.

## Security Questions

### How secure is the encryption?

VaultEnv uses state-of-the-art encryption:

- **Encryption**: AES-256-GCM (authenticated encryption)
- **Key Derivation**: Argon2id (resistant to GPU attacks)
- **Alternative**: ChaCha20-Poly1305 (for performance)
- **Authentication**: HMAC-SHA256 for file integrity

All encryption happens client-side. Even with server access, your secrets remain secure.

### What happens if I forget my password?

**There is no password recovery mechanism by design**. This ensures true zero-knowledge security. If you forget your password:

1. **No backup**: Your encrypted secrets are permanently inaccessible
2. **With backup**: Restore from an unencrypted backup
3. **Team member**: Another team member can re-share the secrets
4. **Prevention**: Use a password manager to store VaultEnv passwords

### Can VaultEnv employees see my secrets?

No. VaultEnv uses zero-knowledge architecture:

- Encryption happens on your machine
- Only encrypted data is transmitted
- Servers never receive encryption keys
- Even with database access, secrets remain encrypted

This is verifiable by reviewing our open-source encryption code.

### How are secrets stored on disk?

Secrets are stored with multiple layers of protection:

```
Your Secret → Compression → Encryption → HMAC → Disk Storage
```

1. **Location**: `~/.vaultenv/storage/` (configurable)
2. **Permissions**: 0600 (user read/write only)
3. **Format**: Binary encrypted format or SQLite database
4. **Cleanup**: Secure deletion overwrites data

### Is it safe to commit encrypted files to Git?

Yes, it's safe to commit VaultEnv encrypted files:

- Strong encryption makes brute force infeasible
- Each environment has a unique encryption key
- No metadata leakage (keys are encrypted too)
- Git history is safe (old passwords can't decrypt new data)

However, always follow these practices:
- Use strong, unique passwords
- Rotate passwords periodically
- Never commit unencrypted exports

## Usage Questions

### How do I share secrets with my team?

VaultEnv provides several methods for team secret sharing:

1. **Via Git** (Recommended):
   ```bash
   # Team member 1: Push encrypted secrets
   vaultenv git push
   
   # Team member 2: Pull and decrypt with password
   vaultenv git pull
   ```

2. **Direct export/import**:
   ```bash
   # Export encrypted backup
   vaultenv export --format vaultenv > secrets.vaultenv
   # Share file securely, then import
   vaultenv load secrets.vaultenv
   ```

3. **VaultEnv Sync** (Pro feature):
   ```bash
   vaultenv sync enable
   # Automatic encrypted synchronization
   ```

### Can I use VaultEnv in CI/CD pipelines?

Yes, VaultEnv is designed for CI/CD integration:

1. **GitHub Actions**:
   ```yaml
   - name: Load secrets
     run: |
       echo "${{ secrets.VAULTENV_PASSWORD }}" | vaultenv auth
       eval $(vaultenv export --format shell)
   ```

2. **Environment variable**:
   ```bash
   export VAULTENV_PASSWORD="your-password"
   vaultenv execute -- npm run deploy
   ```

3. **Docker**:
   ```dockerfile
   RUN vaultenv export --format docker > /app/.env
   ```

### How do I migrate from other tools?

VaultEnv provides built-in migration commands:

```bash
# From .env files
vaultenv migrate dotenv --file .env

# From HashiCorp Vault
vaultenv migrate vault --path secret/myapp

# From AWS Secrets Manager
vaultenv migrate aws --secret-id prod/myapp

# From EnvKey (direct migration)
vaultenv migrate envkey --app myapp
```

See [Migration Guide](./guides/MIGRATION_GUIDE.md) for detailed instructions.

### What's the 5-minute time-to-value?

Here's how to get started in under 5 minutes:

```bash
# 1. Install (30 seconds)
curl -sSL https://vaultenv.dev/install.sh | bash

# 2. Initialize (30 seconds)
vaultenv init

# 3. Add secrets (1 minute)
vaultenv set DATABASE_URL=postgres://localhost/mydb
vaultenv set API_KEY=sk_test_123456

# 4. Use in development (30 seconds)
vaultenv execute -- npm run dev

# 5. Share with team (2 minutes)
git add .vaultenv/
git commit -m "Add encrypted secrets"
git push
```

## Integration Questions

### Which languages and frameworks are supported?

VaultEnv works with any language or framework:

- **Direct support**: Node.js, Python, Ruby, Go, Rust, PHP
- **Via shell**: Any language that reads environment variables
- **Export formats**: .env, JSON, YAML, Docker, shell scripts

Example integrations:
```bash
# Node.js
vaultenv execute -- node app.js

# Python
vaultenv execute -- python manage.py runserver

# Docker
vaultenv export --format docker | docker run --env-file - myapp

# Any command
vaultenv execute -- your-command
```

### Does VaultEnv work with Docker?

Yes, VaultEnv has excellent Docker support:

1. **Build time**:
   ```dockerfile
   FROM node:16
   RUN vaultenv export --env production > .env
   ```

2. **Runtime**:
   ```bash
   # Export for Docker
   vaultenv export --format docker > docker.env
   docker run --env-file docker.env myapp
   ```

3. **Docker Compose**:
   ```yaml
   services:
     app:
       env_file:
         - ${VAULTENV_EXPORT:-docker.env}
   ```

### Can I use VaultEnv with Kubernetes?

Yes, VaultEnv integrates with Kubernetes:

1. **Create secrets**:
   ```bash
   vaultenv export --format k8s | kubectl apply -f -
   ```

2. **Helm values**:
   ```bash
   vaultenv export --format yaml > values.secret.yaml
   helm install myapp ./chart -f values.secret.yaml
   ```

3. **Init containers**:
   ```yaml
   initContainers:
   - name: secrets
     image: vaultenv/cli
     command: ["vaultenv", "export", "--output", "/secrets/.env"]
   ```

## Comparison Questions

### How does VaultEnv compare to HashiCorp Vault?

| Feature | VaultEnv | HashiCorp Vault |
|---------|----------|-----------------|
| Complexity | Simple CLI tool | Full service platform |
| Setup Time | 5 minutes | Hours to days |
| Infrastructure | None required | Servers needed |
| Use Case | Dev/small teams | Enterprise |
| Learning Curve | Minimal | Significant |
| Cost | Free/Low | High TCO |

VaultEnv is perfect for teams wanting security without operational overhead.

### How does VaultEnv compare to AWS Secrets Manager?

| Feature | VaultEnv | AWS Secrets Manager |
|---------|----------|-------------------|
| Cloud Lock-in | None | AWS only |
| Local Development | Native | Requires AWS access |
| Cost | Predictable | Per-secret + API calls |
| Git Integration | Built-in | Manual |
| Setup | Immediate | AWS account required |

VaultEnv works everywhere, while AWS Secrets Manager ties you to AWS.

### What about 1Password CLI or Bitwarden CLI?

Password managers are great for personal credentials but lack developer features:

| Feature | VaultEnv | Password Manager CLIs |
|---------|----------|--------------------|
| Environment Focus | Built for env vars | General passwords |
| Git Workflow | Native | Not designed for it |
| Team Sharing | Encrypted files | Requires accounts |
| .env Compatible | Yes | Adapters needed |
| CI/CD Integration | Native | Limited |

VaultEnv is purpose-built for development workflows.

## Technical Questions

### What are the system requirements?

**Minimum Requirements:**
- OS: Linux, macOS 10.14+, Windows 10+
- RAM: 50MB
- Disk: 100MB for binary + storage
- CPU: Any x64 or ARM64

**Recommended:**
- RAM: 200MB for large secret stores
- SSD for better performance
- Git 2.0+ for sync features

### Which encryption algorithms are supported?

VaultEnv supports multiple algorithms:

1. **AES-256-GCM** (default)
   - NIST approved
   - Hardware acceleration on modern CPUs
   - Best for general use

2. **ChaCha20-Poly1305**
   - Faster on mobile/embedded devices
   - No timing vulnerabilities
   - Google's choice for TLS

3. **Key Derivation**:
   - Argon2id (default): Modern, GPU-resistant
   - scrypt: Legacy compatibility
   - PBKDF2: FIPS compliance

### How do I backup my secrets?

VaultEnv provides multiple backup strategies:

1. **Encrypted backups**:
   ```bash
   vaultenv backup create --all
   # Stored in ~/.vaultenv/backups/
   ```

2. **Git is your backup**:
   ```bash
   # Encrypted files in Git serve as distributed backup
   git push origin main
   ```

3. **Export for cold storage**:
   ```bash
   # Encrypted export
   vaultenv export --format vaultenv > backup-$(date +%Y%m%d).vaultenv
   
   # Unencrypted (store very securely!)
   vaultenv export --decrypt > plaintext-backup.env
   ```

### Can I use VaultEnv offline?

Yes, VaultEnv works completely offline:

- All encryption is local
- No internet required for core features
- Git sync works with local repositories
- Export/import via files

Online features (optional):
- VaultEnv Sync (Pro)
- Update checks
- Telemetry (opt-in)

### What data does VaultEnv collect?

**Open Source Version**: No data collection

**Pro Version** (opt-in telemetry):
- Usage statistics (command frequency)
- Error reports (no secret values)
- Performance metrics
- All data is anonymous

We never collect:
- Secret values
- Secret keys
- Passwords
- Personal information

## Licensing and Support

### Is VaultEnv really open source?

Yes! The core VaultEnv CLI is open source:

- **License**: MIT License
- **Repository**: github.com/yourusername/vaultenv
- **Contributions**: Welcome!
- **Fork**: Allowed and encouraged

The sync server and some enterprise features are proprietary.

### How do I get support?

Support options by tier:

1. **Open Source**:
   - GitHub Issues
   - Community Discord
   - Documentation
   - Stack Overflow

2. **Pro**:
   - Priority email support
   - Response within 24 hours
   - Technical onboarding

3. **Enterprise**:
   - Dedicated support engineer
   - SLA guarantees
   - Phone support
   - Custom training

### Can I use VaultEnv in commercial projects?

Yes! The MIT license allows commercial use:

- Use in proprietary software ✓
- Modify for your needs ✓
- Distribute with your product ✓
- No license fees ✓

Just include the MIT license notice with distributions.

### How do I report security issues?

Security is our top priority. To report issues:

1. **DO NOT** create public GitHub issues
2. Email: security@vaultenv.com
3. Use PGP: [Our public key](https://vaultenv.dev/pgp)
4. Responsible disclosure: 90-day window

We provide:
- Acknowledgment within 48 hours
- Fix timeline
- Credit in release notes (if desired)
- Bug bounty for critical issues

### What's on the roadmap?

Current priorities:

**Q1 2024:**
- Browser extension
- AWS KMS integration
- Terraform provider

**Q2 2024:**
- Mobile app for approvals
- SAML/SSO support
- Compliance reports (SOC2)

**Future:**
- Hardware security key support
- Secret scanning in code
- AI-powered secret detection

See our [public roadmap](https://github.com/yourusername/vaultenv/projects) for details.

## Still Have Questions?

- Check the [Documentation](https://docs.vaultenv.dev)
- Read the [Troubleshooting Guide](./TROUBLESHOOTING.md)
- Join our [Discord Community](https://discord.gg/vaultenv)
- Email support: support@vaultenv.com