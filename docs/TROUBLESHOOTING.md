# VaultEnv Troubleshooting Guide

This guide helps you resolve common issues when using VaultEnv. Each problem includes symptoms, possible causes, and step-by-step solutions.

## Table of Contents

- [Authentication Issues](#authentication-issues)
  - [Permission Denied](#permission-denied)
  - [Password Not Accepted](#password-not-accepted)
  - [Session Timeout](#session-timeout)
- [Environment Issues](#environment-issues)
  - [Environment Not Found](#environment-not-found)
  - [Cannot Switch Environments](#cannot-switch-environments)
  - [Environment Corruption](#environment-corruption)
- [Storage Issues](#storage-issues)
  - [Database Locked](#database-locked)
  - [Storage Path Not Found](#storage-path-not-found)
  - [Disk Space Issues](#disk-space-issues)
- [Encryption Issues](#encryption-issues)
  - [Decryption Failed](#decryption-failed)
  - [Key Derivation Slow](#key-derivation-slow)
  - [Algorithm Not Supported](#algorithm-not-supported)
- [Git Integration Issues](#git-integration-issues)
  - [Sync Conflicts](#sync-conflicts)
  - [Push/Pull Failures](#pushpull-failures)
  - [File Not Tracked](#file-not-tracked)
- [Import/Export Issues](#importexport-issues)
  - [Invalid File Format](#invalid-file-format)
  - [Character Encoding Issues](#character-encoding-issues)
  - [Large File Handling](#large-file-handling)
- [Performance Issues](#performance-issues)
  - [Slow Operations](#slow-operations)
  - [High Memory Usage](#high-memory-usage)
  - [Cache Problems](#cache-problems)
- [Command Issues](#command-issues)
  - [Command Not Found](#command-not-found)
  - [Invalid Arguments](#invalid-arguments)
  - [Shell Integration](#shell-integration)
- [Configuration Issues](#configuration-issues)
  - [Config Not Loading](#config-not-loading)
  - [Invalid Configuration](#invalid-configuration)
  - [Migration Failures](#migration-failures)
- [Security Issues](#security-issues)
  - [Weak Password Warning](#weak-password-warning)
  - [Audit Log Missing](#audit-log-missing)
  - [Suspicious Activity](#suspicious-activity)

## Authentication Issues

### Permission Denied

#### Problem: "Permission denied" when accessing production environment

**Symptoms:**
- Error message: `Error: Permission denied for environment: production`
- Occurs when trying to set or get variables
- Other environments work fine

**Possible Causes:**
1. No password set for production environment
2. Incorrect password entered
3. Corrupted keychain entry
4. Session has expired
5. Environment is protected

**Solutions:**

1. **Check if password is set:**
   ```bash
   vaultenv env list --show-details
   ```
   Look for "Password Set: No" for the environment.

2. **Set or reset password:**
   ```bash
   vaultenv env change-password production
   ```

3. **Clear keychain and re-authenticate:**
   ```bash
   # Clear stored credentials
   vaultenv auth clear production
   
   # Try operation again (will prompt for password)
   vaultenv set DUMMY=test --env production
   ```

4. **Check environment protection:**
   ```bash
   vaultenv env info production
   ```

### Password Not Accepted

**Symptoms:**
- Error: `Invalid password`
- Multiple password prompts
- Previously working password fails

**Solutions:**

1. **Verify caps lock and keyboard layout**

2. **Check for special characters:**
   ```bash
   # Test with simple password first
   vaultenv env change-password test-env
   # Use: SimplePassword123
   ```

3. **Reset password with recovery:**
   ```bash
   # If you have access to another admin
   vaultenv env reset-password production --admin-env development
   ```

### Session Timeout

**Symptoms:**
- Operations suddenly require password
- Error: `Session expired`
- Works after re-entering password

**Solutions:**

1. **Increase timeout:**
   ```bash
   vaultenv config set core.auto_lock_timeout 1800  # 30 minutes
   ```

2. **Disable timeout for current session:**
   ```bash
   export VAULTENV_CORE_AUTO_LOCK_TIMEOUT=0
   ```

## Environment Issues

### Environment Not Found

**Symptoms:**
- Error: `Environment 'staging' not found`
- `vaultenv env list` doesn't show expected environment

**Solutions:**

1. **List all environments:**
   ```bash
   vaultenv env list --all
   ```

2. **Check current directory:**
   ```bash
   # VaultEnv might be using project-specific storage
   pwd
   vaultenv config get core.storage_path
   ```

3. **Initialize missing environment:**
   ```bash
   vaultenv init staging
   ```

### Cannot Switch Environments

**Symptoms:**
- `vaultenv env switch` doesn't change environment
- Operations still use old environment

**Solutions:**

1. **Check current environment:**
   ```bash
   vaultenv env current
   ```

2. **Use explicit environment flag:**
   ```bash
   vaultenv list --env production
   ```

3. **Set default environment:**
   ```bash
   vaultenv config set core.default_environment production
   ```

### Environment Corruption

**Symptoms:**
- Error: `Database corruption detected`
- Inconsistent variable listings
- Missing variables

**Solutions:**

1. **Run integrity check:**
   ```bash
   vaultenv security scan --env production
   ```

2. **Restore from backup:**
   ```bash
   # List available backups
   vaultenv backup list
   
   # Restore specific backup
   vaultenv backup restore --file backup-20240115.vaultenv
   ```

3. **Rebuild from Git:**
   ```bash
   vaultenv git pull --force --env production
   ```

## Storage Issues

### Database Locked

**Symptoms:**
- Error: `database is locked`
- Operations hang indefinitely
- Multiple processes accessing VaultEnv

**Solutions:**

1. **Check for running processes:**
   ```bash
   ps aux | grep vaultenv
   ```

2. **Clear lock file:**
   ```bash
   # Find storage path
   vaultenv config get storage.sqlite.path
   
   # Remove lock files
   rm ~/.vaultenv/storage/vaultenv.db-wal
   rm ~/.vaultenv/storage/vaultenv.db-shm
   ```

3. **Use exclusive mode:**
   ```bash
   vaultenv --exclusive set KEY=value
   ```

### Storage Path Not Found

**Symptoms:**
- Error: `Storage path does not exist`
- Cannot initialize environment
- Fresh installation issues

**Solutions:**

1. **Create storage directory:**
   ```bash
   mkdir -p ~/.vaultenv/storage
   chmod 700 ~/.vaultenv/storage
   ```

2. **Check permissions:**
   ```bash
   ls -la ~/.vaultenv/
   ```

3. **Use custom path:**
   ```bash
   vaultenv init --path /custom/secure/path
   ```

### Disk Space Issues

**Symptoms:**
- Error: `No space left on device`
- Operations fail randomly
- Backup creation fails

**Solutions:**

1. **Check disk space:**
   ```bash
   df -h ~/.vaultenv/
   ```

2. **Clean old history:**
   ```bash
   vaultenv history clean --older-than 30d
   ```

3. **Rotate backups:**
   ```bash
   vaultenv backup rotate --keep 5
   ```

## Encryption Issues

### Decryption Failed

**Symptoms:**
- Error: `Decryption failed: invalid data`
- Corrupted values shown
- Variables appear as binary data

**Solutions:**

1. **Verify password:**
   ```bash
   vaultenv env verify-password production
   ```

2. **Check encryption algorithm:**
   ```bash
   vaultenv env info production | grep Algorithm
   ```

3. **Re-encrypt with correct settings:**
   ```bash
   vaultenv security rotate --env production
   ```

### Key Derivation Slow

**Symptoms:**
- Long delays when entering password
- CPU usage spikes
- Operations timeout

**Solutions:**

1. **Check current settings:**
   ```bash
   vaultenv config get encryption.key_derivation_params
   ```

2. **Adjust parameters:**
   ```bash
   vaultenv config set encryption.key_derivation_params.memory 32768
   vaultenv config set encryption.key_derivation_params.time 2
   ```

3. **Use faster algorithm for development:**
   ```bash
   vaultenv env recreate development --fast-crypto
   ```

### Algorithm Not Supported

**Symptoms:**
- Error: `Unsupported algorithm: aes-128-cbc`
- Cannot read old encrypted files
- Migration from other tools fails

**Solutions:**

1. **Check supported algorithms:**
   ```bash
   vaultenv --help-algorithms
   ```

2. **Convert to supported format:**
   ```bash
   vaultenv migrate legacy --from-algorithm aes-128-cbc
   ```

## Git Integration Issues

### Sync Conflicts

**Symptoms:**
- Error: `Merge conflict in encrypted file`
- Git shows binary file conflicts
- Cannot pull or push changes

**Solutions:**

1. **Use VaultEnv's conflict resolution:**
   ```bash
   vaultenv git resolve-conflicts
   ```

2. **Manual resolution:**
   ```bash
   # Backup local version
   vaultenv export > local-backup.env
   
   # Accept remote version
   git checkout --theirs .vaultenv/production.vaultenv
   
   # Re-apply local changes
   vaultenv load local-backup.env --merge
   ```

3. **Prevent future conflicts:**
   ```bash
   vaultenv config set git.sync_on_change true
   ```

### Push/Pull Failures

**Symptoms:**
- Git operations fail
- Error: `File not in repository`
- Changes not syncing

**Solutions:**

1. **Initialize Git integration:**
   ```bash
   vaultenv git init
   ```

2. **Check Git status:**
   ```bash
   vaultenv git status
   git status .vaultenv/
   ```

3. **Force sync:**
   ```bash
   vaultenv git push --force
   ```

## Import/Export Issues

### Invalid File Format

**Symptoms:**
- Error: `Invalid format: unknown`
- Import fails silently
- Partial data imported

**Solutions:**

1. **Specify format explicitly:**
   ```bash
   vaultenv load config.env --format dotenv
   ```

2. **Validate file first:**
   ```bash
   vaultenv validate --file config.env
   ```

3. **Convert format:**
   ```bash
   # Convert JSON to .env
   jq -r 'to_entries[] | "\(.key)=\(.value)"' config.json > config.env
   ```

### Character Encoding Issues

**Symptoms:**
- Special characters appear as `?` or �
- Unicode errors during import
- Export produces invalid files

**Solutions:**

1. **Check file encoding:**
   ```bash
   file -i problematic.env
   ```

2. **Convert to UTF-8:**
   ```bash
   iconv -f ISO-8859-1 -t UTF-8 input.env > output.env
   ```

3. **Use ASCII-safe export:**
   ```bash
   vaultenv export --ascii-only
   ```

## Performance Issues

### Slow Operations

**Symptoms:**
- Commands take several seconds
- List operations very slow
- High CPU usage

**Solutions:**

1. **Enable caching:**
   ```bash
   vaultenv config set performance.cache_enabled true
   ```

2. **Optimize database:**
   ```bash
   vaultenv maintenance optimize
   ```

3. **Use batch operations:**
   ```bash
   # Instead of multiple set commands
   vaultenv batch set-multiple.txt
   ```

### High Memory Usage

**Symptoms:**
- VaultEnv consumes excessive RAM
- System becomes unresponsive
- Out of memory errors

**Solutions:**

1. **Limit cache size:**
   ```bash
   vaultenv config set performance.cache_max_size 50
   ```

2. **Disable compression for large values:**
   ```bash
   vaultenv config set storage.compression false
   ```

## Command Issues

### Command Not Found

**Symptoms:**
- `bash: vaultenv: command not found`
- Shell completion not working
- Path issues

**Solutions:**

1. **Check installation:**
   ```bash
   which vaultenv
   echo $PATH
   ```

2. **Add to PATH:**
   ```bash
   export PATH=$PATH:/usr/local/bin
   echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
   ```

3. **Install completions:**
   ```bash
   vaultenv completion bash > ~/.vaultenv-completion
   source ~/.vaultenv-completion
   ```

### Shell Integration

**Symptoms:**
- Variables not available in shell
- `vaultenv shell` not working
- Environment not inherited

**Solutions:**

1. **Use proper shell syntax:**
   ```bash
   # Bash/Zsh
   eval "$(vaultenv export --format shell)"
   
   # Fish
   vaultenv export --format fish | source
   ```

2. **Check shell support:**
   ```bash
   vaultenv shell --shell $SHELL
   ```

## Configuration Issues

### Config Not Loading

**Symptoms:**
- Custom settings ignored
- Default values always used
- Config changes have no effect

**Solutions:**

1. **Check config location:**
   ```bash
   vaultenv config get --show-source
   ```

2. **Validate config syntax:**
   ```bash
   vaultenv config validate --file ~/.vaultenv/config.yaml
   ```

3. **Use environment override:**
   ```bash
   export VAULTENV_CORE_DEFAULT_ENVIRONMENT=production
   ```

## Security Issues

### Weak Password Warning

**Symptoms:**
- Warning: `Password does not meet security requirements`
- Cannot set simple passwords
- Password change required

**Solutions:**

1. **Check requirements:**
   ```bash
   vaultenv config get security.password_complexity
   ```

2. **Generate strong password:**
   ```bash
   vaultenv generate-password --length 20
   ```

3. **Override for development:**
   ```bash
   vaultenv env create dev --no-password-policy
   ```

### Suspicious Activity

**Symptoms:**
- Unexpected environment access in logs
- Variables changed without your action
- Audit shows unknown operations

**Solutions:**

1. **Review audit logs:**
   ```bash
   vaultenv history --env production --limit 100 --format detailed
   ```

2. **Immediate response:**
   ```bash
   # Change all passwords
   vaultenv security emergency-rotation
   
   # Export current state
   vaultenv export --all > emergency-backup.json
   
   # Notify team
   vaultenv security alert "Suspicious activity detected"
   ```

## Getting Additional Help

If these solutions don't resolve your issue:

1. **Enable debug logging:**
   ```bash
   export VAULTENV_DEBUG=true
   vaultenv [your-command] --verbose
   ```

2. **Collect diagnostic info:**
   ```bash
   vaultenv doctor > diagnostic-info.txt
   ```

3. **Check documentation:**
   - [Command Reference](./reference/COMMANDS.md)
   - [Configuration Guide](./reference/CONFIGURATION.md)
   - [FAQ](./FAQ.md)

4. **Report issues:**
   - GitHub: https://github.com/yourusername/vaultenv/issues
   - Include diagnostic info and steps to reproduce

## Common Quick Fixes

```bash
# Reset everything (CAUTION: Data loss)
vaultenv reset --all --force

# Safe mode (bypass custom config)
vaultenv --safe-mode [command]

# Verbose debugging
vaultenv --debug --verbose [command] 2> debug.log

# Check system compatibility
vaultenv doctor --full
```