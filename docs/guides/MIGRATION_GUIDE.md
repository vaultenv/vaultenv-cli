# VaultEnv Migration Guide

This guide helps you migrate from other environment management tools to VaultEnv. Whether you're using plain `.env` files, direnv, dotenv, or similar tools, we'll show you how to transition smoothly while improving your security posture.

## Table of Contents

- [Migration from .env Files](#migration-from-env-files)
- [Migration from direnv](#migration-from-direnv)
- [Migration from dotenv-vault](#migration-from-dotenv-vault)
- [Migration from AWS Secrets Manager](#migration-from-aws-secrets-manager)
- [Team Migration Strategies](#team-migration-strategies)
- [Handling Edge Cases](#handling-edge-cases)
- [Rollback Procedures](#rollback-procedures)
- [Common Migration Issues](#common-migration-issues)

## Migration from .env Files

### Quick Migration (Single File)

The fastest way to migrate a `.env` file:

```bash
# Initialize VaultEnv in your project
vaultenv init

# Import your .env file
vaultenv import .env

# Verify the import
vaultenv list

# Remove the old .env file (after verifying!)
rm .env
```

### Multiple .env Files

If you have environment-specific files:

```bash
# Create environments matching your .env files
vaultenv env create development
vaultenv env create staging
vaultenv env create production

# Import each file to its environment
vaultenv import .env.development --env development
vaultenv import .env.staging --env staging
vaultenv import .env.production --env production

# Clean up old files
rm .env.*
```

### Gradual Migration

For large teams, migrate gradually:

```bash
# Step 1: Import but keep .env as backup
vaultenv import .env
cp .env .env.backup

# Step 2: Update your start scripts
# Old: node app.js
# New: vaultenv run -- node app.js

# Step 3: After verification, remove .env
rm .env .env.backup
```

### Update Your Scripts

Replace `.env` loading in your code:

#### Node.js (Before)
```javascript
require('dotenv').config();
```

#### Node.js (After)
```javascript
// No changes needed! Run with: vaultenv run -- node app.js
// Or keep dotenv as fallback:
if (!process.env.VAULTENV) {
  require('dotenv').config();
}
```

#### Python (Before)
```python
from dotenv import load_dotenv
load_dotenv()
```

#### Python (After)
```python
# No changes needed! Run with: vaultenv run -- python app.py
# Or keep as fallback:
import os
if not os.environ.get('VAULTENV'):
    from dotenv import load_dotenv
    load_dotenv()
```

### Update .gitignore

```bash
# Remove .env from gitignore (we'll track encrypted files)
sed -i '' '/.env/d' .gitignore

# Add VaultEnv encryption keys (if not using OS keychain)
echo ".vaultenv/keys/" >> .gitignore
```

## Migration from direnv

### Understanding the Differences

| Feature | direnv | VaultEnv |
|---------|--------|----------|
| Auto-load | ✅ On directory change | ✅ With shell integration |
| Encryption | ❌ Plain text | ✅ AES-256-GCM |
| Team Sync | ❌ Manual | ✅ Git-based |
| Multiple Envs | ✅ Per directory | ✅ Named environments |

### Migration Steps

1. **Export Current Variables**
   ```bash
   # In directory with .envrc
   direnv exec . env > current-env.txt
   ```

2. **Initialize VaultEnv**
   ```bash
   vaultenv init
   ```

3. **Import Variables**
   ```bash
   # Convert direnv format to VaultEnv
   grep -v '^DIRENV' current-env.txt | vaultenv load -
   ```

4. **Set Up Auto-loading (Optional)**
   ```bash
   # Add to .bashrc/.zshrc
   vaultenv_auto() {
     if [[ -f .vaultenv/config.json ]]; then
       eval $(vaultenv export --format shell)
     fi
   }
   
   # For bash
   PROMPT_COMMAND="vaultenv_auto;$PROMPT_COMMAND"
   
   # For zsh
   precmd_functions+=(vaultenv_auto)
   ```

5. **Remove direnv**
   ```bash
   rm .envrc
   direnv deny
   ```

### Layout Migration

If using direnv layouts:

```bash
# direnv layout python
# Equivalent in VaultEnv:
vaultenv set VIRTUAL_ENV=.direnv/python-3.9.7
vaultenv set PATH=".direnv/python-3.9.7/bin:$PATH"
```

## Migration from dotenv-vault

### Feature Comparison

| Feature | dotenv-vault | VaultEnv |
|---------|--------------|----------|
| Encryption | ✅ Cloud-based | ✅ Local zero-knowledge |
| Offline Work | ❌ Requires internet | ✅ Fully offline |
| Team Sync | ✅ Via cloud | ✅ Via Git |
| Pricing | 💰 Paid tiers | ✅ Free & open source |

### Migration Process

1. **Export from dotenv-vault**
   ```bash
   # Pull latest from dotenv-vault
   npx dotenv-vault@latest pull
   
   # Export each environment
   npx dotenv-vault@latest decrypt .env.vault -e development > dev.env
   npx dotenv-vault@latest decrypt .env.vault -e production > prod.env
   ```

2. **Import to VaultEnv**
   ```bash
   # Initialize VaultEnv
   vaultenv init
   
   # Create environments
   vaultenv env create development
   vaultenv env create production
   
   # Import variables
   vaultenv import dev.env --env development
   vaultenv import prod.env --env production
   
   # Clean up
   rm dev.env prod.env .env.vault
   ```

3. **Update CI/CD**
   ```yaml
   # Before (GitHub Actions)
   - name: Load secrets
     run: |
       npx dotenv-vault@latest decrypt .env.vault -e production
   
   # After
   - name: Load secrets
     env:
       VAULTENV_PASSWORD: ${{ secrets.VAULTENV_PROD_PASSWORD }}
     run: |
       vaultenv export --env production --format dotenv > .env
   ```

## Migration from AWS Secrets Manager

### Using VaultEnv's AWS Integration

```bash
# Export from AWS Secrets Manager
aws secretsmanager get-secret-value \
  --secret-id myapp/production \
  --query SecretString \
  --output text | jq -r 'to_entries|.[]|"\(.key)=\(.value)"' > aws-secrets.env

# Import to VaultEnv
vaultenv import aws-secrets.env --env production

# Clean up
rm aws-secrets.env
```

### Maintaining AWS Sync (Advanced)

Create a sync script:

```bash
#!/bin/bash
# sync-from-aws.sh

SECRET_ID="myapp/production"
ENV="production"

# Get secrets from AWS
aws secretsmanager get-secret-value \
  --secret-id $SECRET_ID \
  --query SecretString \
  --output text | jq -r 'to_entries|.[]|"\(.key)=\(.value)"' | \
while IFS='=' read -r key value; do
  vaultenv set "$key=$value" --env $ENV
done
```

## Team Migration Strategies

### Small Team (2-5 developers)

1. **Designated Migration Lead**
   ```bash
   # Lead developer
   vaultenv init
   vaultenv import .env
   git add .vaultenv
   git commit -m "Initialize VaultEnv"
   git push
   ```

2. **Team Members Pull and Setup**
   ```bash
   # Each team member
   git pull
   vaultenv env use default  # Enter agreed password
   ```

3. **Communication**
   - Share passwords securely (use a password manager)
   - Schedule a quick team sync to ensure everyone's set up

### Large Team (5+ developers)

1. **Phased Rollout**
   ```bash
   # Phase 1: Dev environment only
   vaultenv init
   vaultenv import .env.development --env development
   
   # Phase 2: Add staging after 1 week
   vaultenv import .env.staging --env staging
   
   # Phase 3: Production after validation
   vaultenv import .env.production --env production
   ```

2. **Parallel Running**
   ```javascript
   // Support both methods temporarily
   if (process.env.VAULTENV) {
     console.log('Using VaultEnv');
   } else {
     require('dotenv').config();
     console.log('Using .env file');
   }
   ```

3. **Documentation**
   Create a migration guide for your team:
   ```markdown
   # MyApp VaultEnv Migration
   
   1. Pull latest: `git pull`
   2. Install VaultEnv: `brew install vaultenv-cli`
   3. Get password from team lead
   4. Run: `vaultenv env use development`
   5. Test: `vaultenv run -- npm start`
   ```

### Enterprise Migration

1. **Pilot Project**
   - Start with non-critical service
   - Document lessons learned
   - Build internal expertise

2. **Security Review**
   ```bash
   # Generate security report
   vaultenv security audit > vaultenv-security-review.txt
   ```

3. **Training Materials**
   - Record setup video
   - Create internal wiki page
   - Host lunch-and-learn session

## Handling Edge Cases

### Multi-line Values

```bash
# .env file with multi-line value
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"

# VaultEnv handles these automatically
vaultenv import .env
```

### Special Characters

```bash
# Values with special characters
vaultenv set 'PASSWORD=p@$$w0rd!#$%'
vaultenv set 'URL=https://user:pass@host:5432/db?ssl=true'
```

### Large Values

```bash
# For very large values (certificates, etc.)
vaultenv set CERT=@certificate.pem  # Read from file
```

### Dynamic Values

For values that change frequently:

```bash
# Don't encrypt frequently changing values
vaultenv set BUILD_NUMBER="$CI_BUILD_NUMBER" --no-encrypt
```

## Rollback Procedures

### Quick Rollback

If you need to rollback quickly:

```bash
# 1. Export current VaultEnv variables
vaultenv export --format dotenv > .env.emergency

# 2. Revert to using .env file
# Update your scripts to use .env directly

# 3. When ready to retry, import again
vaultenv import .env.emergency
```

### Partial Rollback

Keep both systems during transition:

```javascript
// Use VaultEnv if available, fallback to .env
function loadConfig() {
  if (process.env.VAULTENV) {
    console.log('Using VaultEnv');
    return;
  }
  
  try {
    require('dotenv').config();
    console.log('Using .env file');
  } catch (e) {
    console.error('No configuration found!');
    process.exit(1);
  }
}
```

### Data Recovery

If you lose access to VaultEnv:

```bash
# If you have backups
cp .vaultenv.backup .vaultenv

# If you have git history
git checkout HEAD~1 -- .vaultenv

# Last resort: check CI/CD logs
# Most CI systems log environment variables (masked)
```

## Common Migration Issues

### Issue: "Variable not found" after migration

**Cause**: Variable names might have been modified during import.

**Solution**:
```bash
# Check exact variable names
vaultenv list
# Look for differences in case or underscores vs dashes
```

### Issue: Special characters corrupted

**Cause**: Encoding issues during import.

**Solution**:
```bash
# Re-import with explicit encoding
iconv -f ISO-8859-1 -t UTF-8 .env | vaultenv load -
```

### Issue: Git conflicts in .vaultenv files

**Cause**: Team members setting different values.

**Solution**:
```bash
# View conflicts
vaultenv conflicts

# Resolve conflicts
vaultenv conflicts resolve KEY --use-theirs
```

### Issue: CI/CD pipeline fails

**Cause**: Missing password in CI environment.

**Solution**:
```yaml
# Add to CI secrets
VAULTENV_PASSWORD_PRODUCTION

# In pipeline
env:
  VAULTENV_PASSWORD: ${{ secrets.VAULTENV_PASSWORD_PRODUCTION }}
```

### Issue: Performance degradation

**Cause**: File backend is slower with many variables.

**Solution**:
```bash
# Switch to SQLite backend
vaultenv config set storage.type sqlite
vaultenv migrate
```

## Post-Migration Checklist

- [ ] All environments imported successfully
- [ ] Team members have access
- [ ] CI/CD pipelines updated
- [ ] Old .env files removed
- [ ] .gitignore updated
- [ ] Documentation updated
- [ ] Backup strategy in place
- [ ] Monitoring for issues

## Best Practices After Migration

1. **Regular Audits**
   ```bash
   vaultenv audit --days 30
   ```

2. **Key Rotation**
   ```bash
   vaultenv security rotate --env production
   ```

3. **Access Reviews**
   ```bash
   vaultenv access review
   ```

4. **Backup Strategy**
   ```bash
   # Automated backups
   vaultenv backup create --all-envs
   ```

## Getting Help

- **Migration Support**: [Discord #migration channel](https://discord.gg/vaultenv)
- **Common Issues**: [FAQ](https://docs.vaultenv.dev/faq)
- **Professional Help**: migration@vaultenv.dev

Congratulations on making your environment variables more secure! 🎉