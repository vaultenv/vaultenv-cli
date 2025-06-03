# Getting Started with VaultEnv

Welcome to VaultEnv! This guide will take you from installation to managing your first encrypted environment variables in just a few minutes. By the end, you'll understand how VaultEnv keeps your secrets safe while making development easier.

## Table of Contents

- [Installation](#installation)
- [Initial Setup](#initial-setup)
- [Your First Variable](#your-first-variable)
- [Understanding Environments](#understanding-environments)
- [Basic Workflow Example](#basic-workflow-example)
- [Next Steps](#next-steps)

## Installation

### macOS and Linux

#### Using Homebrew (Recommended)

```bash
brew tap vaultenv/tap
brew install vaultenv-cli
```

#### Using the Install Script

```bash
curl -sSL https://install.vaultenv.dev | bash
```

#### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/vaultenv/vaultenv-cli/releases)
2. Extract the archive:
   ```bash
   tar -xzf vaultenv-cli_*_linux_amd64.tar.gz
   ```
3. Move to your PATH:
   ```bash
   sudo mv vaultenv /usr/local/bin/
   ```

### Windows

#### Using Scoop

```powershell
scoop bucket add vaultenv https://github.com/vaultenv/scoop-bucket
scoop install vaultenv-cli
```

#### Manual Installation

1. Download the Windows binary from [GitHub Releases](https://github.com/vaultenv/vaultenv-cli/releases)
2. Extract the ZIP file
3. Add the directory to your PATH environment variable

### Go Developers

If you have Go 1.22+ installed:

```bash
go install github.com/vaultenv/vaultenv-cli/cmd/vaultenv-cli@latest
```

### Docker

```bash
docker run -it -v $(pwd):/workspace vaultenv/cli:latest
```

### Verify Installation

```bash
vaultenv version
# Output: vaultenv version 0.1.0-beta.1
```

## Initial Setup

### 1. Initialize Your Project

Navigate to your project directory and initialize VaultEnv:

```bash
cd my-project
vaultenv init
```

This creates a `.vaultenv` directory with the following structure:

```
.vaultenv/
├── config.json          # VaultEnv configuration
├── environments/        # Environment-specific files
│   └── default/        # Default environment
└── keys/               # Encrypted keys (if not using OS keychain)
```

**What happens during init:**
- Creates the VaultEnv directory structure
- Sets up the default environment
- Configures encryption settings
- Initializes Git integration (if in a Git repository)

### 2. Understanding the Configuration

VaultEnv created a `config.json` file. Let's look at key settings:

```json
{
  "version": "1.0",
  "default_environment": "default",
  "encryption": {
    "algorithm": "aes-gcm-256",
    "key_derivation": "argon2id"
  },
  "storage": {
    "type": "file",
    "path": ".vaultenv"
  }
}
```

You can modify these settings with:

```bash
vaultenv config set storage.type sqlite  # Use SQLite for better performance
vaultenv config get encryption.algorithm  # View current encryption
```

## Your First Variable

### Setting a Variable

Let's set your first encrypted environment variable:

```bash
vaultenv set DATABASE_URL="postgres://localhost:5432/myapp"
```

**What happens:**
1. VaultEnv prompts for a password (first time only)
2. The password is used to derive an encryption key
3. The value is encrypted using AES-256-GCM
4. The encrypted value is stored in `.vaultenv/environments/default/`
5. The encryption key is stored in your OS keychain

### Getting a Variable

Retrieve your variable:

```bash
vaultenv get DATABASE_URL
# Output: postgres://localhost:5432/myapp
```

### Listing Variables

See all variables in the current environment:

```bash
vaultenv list
# Output:
# DATABASE_URL
```

To see values too (be careful in shared environments!):

```bash
vaultenv list --show-values
# Output:
# DATABASE_URL=postgres://localhost:5432/myapp
```

### Using Variables in Your Application

#### Method 1: Run Command

Execute any command with your variables loaded:

```bash
vaultenv run -- npm start
vaultenv run -- python app.py
vaultenv run -- cargo run
```

#### Method 2: Export to Shell

Load variables into your current shell:

```bash
eval $(vaultenv export --format shell)
echo $DATABASE_URL  # Now available in shell
```

#### Method 3: Export to File

Export for CI/CD or Docker:

```bash
vaultenv export --format dotenv > .env
# Remember to add .env to .gitignore!
```

## Understanding Environments

Environments let you separate variables for different contexts (development, staging, production).

### Creating Environments

```bash
# Create a new environment
vaultenv env create production

# Create with description
vaultenv env create staging --description "Staging environment for QA"
```

### Switching Environments

```bash
# Switch to production
vaultenv env use production

# Or use --env flag with any command
vaultenv set API_KEY="prod-key" --env production
vaultenv get API_KEY --env production
```

### Listing Environments

```bash
vaultenv env list
# Output:
# * default
#   staging    - Staging environment for QA
#   production
```

### Environment-Specific Passwords

Each environment can have its own password:

```bash
# Set production password
vaultenv env use production
vaultenv set SECRET="prod-secret"  # Will prompt for production password
```

## Basic Workflow Example

Here's a complete example of using VaultEnv in a Node.js project:

### 1. Initialize and Configure

```bash
# Initialize VaultEnv
vaultenv init

# Create environments
vaultenv env create development --description "Local development"
vaultenv env create production --description "Production environment"
```

### 2. Set Development Variables

```bash
# Switch to development
vaultenv env use development

# Set development variables
vaultenv set DATABASE_URL="postgres://localhost:5432/myapp_dev"
vaultenv set API_KEY="dev-api-key-123"
vaultenv set LOG_LEVEL="debug"
```

### 3. Set Production Variables

```bash
# Switch to production
vaultenv env use production

# Set production variables (will prompt for production password)
vaultenv set DATABASE_URL="postgres://prod.example.com:5432/myapp"
vaultenv set API_KEY="prod-api-key-456"
vaultenv set LOG_LEVEL="info"
```

### 4. Use in Development

```bash
# Back to development
vaultenv env use development

# Run your app with development variables
vaultenv run -- npm run dev
```

### 5. Share with Team

```bash
# Encrypted files are safe to commit
git add .vaultenv
git commit -m "Add encrypted environment configuration"
git push
```

Your teammate can now:

```bash
git pull
vaultenv env use development  # Will prompt for password
vaultenv run -- npm run dev   # Same environment!
```

### 6. Deploy to Production

In your CI/CD pipeline:

```yaml
# GitHub Actions example
- name: Setup VaultEnv
  run: |
    curl -sSL https://install.vaultenv.dev | bash
    
- name: Export production variables
  env:
    VAULTENV_PASSWORD: ${{ secrets.VAULTENV_PROD_PASSWORD }}
  run: |
    vaultenv export --env production --format dotenv > .env
    
- name: Deploy
  run: |
    # Your deployment commands
    # Variables are now in .env file
```

## Common Patterns

### Import Existing .env Files

Migrating from plain .env files:

```bash
# Import into current environment
vaultenv import .env

# Import into specific environment
vaultenv import .env.production --env production
```

### Batch Operations

Set multiple variables at once:

```bash
# From arguments
vaultenv set KEY1=value1 KEY2=value2 KEY3=value3

# From file
vaultenv load variables.txt
```

### Working with JSON

```bash
# Export as JSON
vaultenv export --format json > config.json

# Use with jq
vaultenv export --format json | jq '.DATABASE_URL'
```

### Shell Aliases

Add to your `.bashrc` or `.zshrc`:

```bash
# Quick shortcuts
alias ve="vaultenv"
alias ver="vaultenv run --"

# Run with production env
alias vep="vaultenv --env production"
```

## Security Best Practices

1. **Strong Passwords**: Use unique, strong passwords for each environment
2. **Don't Share Passwords**: Each team member should have their own
3. **Separate Environments**: Keep production credentials separate
4. **Regular Rotation**: Change passwords periodically
5. **Audit Access**: Use `vaultenv history` to track changes

## Troubleshooting

### "Permission denied" Error

This usually means you haven't entered the password for this environment:

```bash
# This will prompt for password
vaultenv set DUMMY=test
```

### Forgot Password

If you forget an environment's password, you'll need to:
1. Delete the environment: `vaultenv env delete myenv`
2. Recreate it: `vaultenv env create myenv`
3. Re-add your variables

### Can't Find vaultenv Command

Make sure it's in your PATH:

```bash
which vaultenv
# Should show: /usr/local/bin/vaultenv
```

### Git Merge Conflicts

If you get conflicts in `.vaultenv` files:

```bash
# View conflicts
vaultenv conflicts

# Resolve by choosing version
vaultenv conflicts resolve KEY --use-ours
vaultenv conflicts resolve KEY --use-theirs
```

## Next Steps

Now that you're up and running with VaultEnv:

1. **Read about team collaboration**: [Team Collaboration Guide](./TEAM_COLLABORATION.md)
2. **Learn security best practices**: [Security Best Practices](./SECURITY_BEST_PRACTICES.md)
3. **Migrate from other tools**: [Migration Guide](./MIGRATION_GUIDE.md)
4. **Explore advanced features**:
   - SQLite backend for better performance
   - Git integration for version control
   - Audit trails for compliance
   - Shell completions for productivity

## Getting Help

- **Documentation**: [docs.vaultenv.dev](https://docs.vaultenv.dev)
- **GitHub Issues**: [Report bugs or request features](https://github.com/vaultenv/vaultenv-cli/issues)
- **Discord Community**: [discord.gg/vaultenv](https://discord.gg/vaultenv)
- **Email Support**: support@vaultenv.dev

Welcome to the VaultEnv community! 🚀