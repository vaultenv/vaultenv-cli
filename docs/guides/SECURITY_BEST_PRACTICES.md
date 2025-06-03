# Security Best Practices Guide

This guide provides comprehensive security best practices for managing secrets with VaultEnv and general secret management principles.

## Table of Contents

- [Password and Passphrase Selection](#password-and-passphrase-selection)
- [Environment Separation Strategies](#environment-separation-strategies)
- [Key Rotation Procedures](#key-rotation-procedures)
- [Audit and Compliance](#audit-and-compliance)
- [Incident Response Planning](#incident-response-planning)
- [Security Checklist](#security-checklist)

## Password and Passphrase Selection

### Strong Password Requirements

VaultEnv uses your password to derive encryption keys. A strong password is your first line of defense.

**Minimum Requirements:**
- At least 12 characters (16+ recommended)
- Mix of uppercase and lowercase letters
- Include numbers and special characters
- Avoid dictionary words and personal information

### Passphrase Best Practices

Consider using passphrases instead of passwords:

```bash
# Good passphrase example
"correct-horse-battery-staple-2024!"

# Even better with personal twist
"MyC@t$Name!sFluffyAndLoves2Nap$"
```

### Password Managers

**Recommendation**: Store your VaultEnv passwords in a password manager:
- Generate unique passwords per environment
- Never reuse passwords across environments
- Enable 2FA on your password manager

## Environment Separation Strategies

### Environment Isolation

**Never share encryption keys between environments:**

```bash
# Create separate environments with unique passwords
vaultenv init development
vaultenv init staging  
vaultenv init production

# Use different passwords for each!
```

### Access Control Patterns

1. **Development Environment**
   - Shared password among developers (if necessary)
   - Rotate monthly
   - Use for non-sensitive test data only

2. **Staging Environment**
   - Limited access (senior developers + DevOps)
   - Rotate quarterly
   - Mirror production security practices

3. **Production Environment**
   - Minimal access (DevOps + authorized personnel only)
   - Rotate on personnel changes
   - Audit all access

### Environment-Specific Practices

```bash
# Development: More permissive
vaultenv set DEBUG=true --env development

# Production: Restrictive
vaultenv set DEBUG=false --env production
vaultenv env protect production  # Prevents accidental modifications
```

## Key Rotation Procedures

### Regular Rotation Schedule

Establish a rotation schedule based on environment sensitivity:

| Environment | Rotation Frequency | Trigger Events |
|------------|-------------------|----------------|
| Development | Monthly | Team changes |
| Staging | Quarterly | Security updates |
| Production | Bi-annually | Any security incident |

### Rotation Process

1. **Announce rotation** (1 week notice)
2. **Create new environment**
   ```bash
   vaultenv env copy production production-new
   vaultenv env change-password production-new
   ```

3. **Migrate systems** (staged rollout)
4. **Verify functionality**
5. **Deprecate old environment**
   ```bash
   vaultenv env remove production-old --force
   ```

### Emergency Rotation

In case of suspected compromise:

```bash
# 1. Immediately create new environment
vaultenv env copy production production-emergency

# 2. Change password
vaultenv env change-password production-emergency

# 3. Update critical systems first
vaultenv export --env production-emergency --format dotenv > .env.emergency

# 4. Rotate external secrets (API keys, etc.)
```

## Audit and Compliance

### Enable Audit Logging

Track all secret access and modifications:

```bash
# Enable audit logging
vaultenv config set audit.enabled true
vaultenv config set audit.level detailed

# View audit logs
vaultenv history --env production --format json > audit.log
```

### Compliance Considerations

1. **SOC 2 Compliance**
   - Document all secret access
   - Implement least privilege access
   - Regular access reviews
   - Encryption at rest (VaultEnv default)

2. **GDPR Compliance**
   - No PII in secret values
   - Right to erasure implementation
   - Data residency considerations

3. **HIPAA Compliance**
   - PHI encryption requirements met
   - Access logging mandatory
   - Regular security assessments

### Regular Security Reviews

Monthly tasks:
```bash
# Review who has access
vaultenv env list --show-details

# Check for unused variables
vaultenv list --env production | grep -i "deprecated\|old\|unused"

# Verify encryption settings
vaultenv config get encryption.algorithm
```

## Incident Response Planning

### Preparation Phase

1. **Document all environments**
   ```bash
   vaultenv env list > environments.txt
   vaultenv config show > config-backup.txt
   ```

2. **Establish response team**
   - Security lead
   - DevOps engineer
   - Development representative

3. **Create response playbook**

### Detection and Analysis

Signs of potential compromise:
- Unexpected environment access
- Modified secrets without authorization
- Unusual export operations
- Failed authentication attempts

```bash
# Check recent activity
vaultenv history --env production --limit 100

# Look for anomalies
vaultenv history --env production | grep -E "(export|batch|clear)"
```

### Containment and Recovery

```bash
# 1. Revoke access immediately
vaultenv env change-password production

# 2. Audit recent changes
vaultenv history --env production --since "2024-01-01"

# 3. Rotate affected external secrets
# (API keys, database passwords, etc.)

# 4. Restore from secure backup if needed
vaultenv load backup-production.vaultenv --env production-restored
```

### Post-Incident Activities

- Document timeline and impact
- Update security procedures
- Implement additional monitoring
- Conduct team retrospective

## Security Checklist

### Initial Setup
- [ ] Strong, unique password per environment
- [ ] Password stored in password manager
- [ ] Audit logging enabled
- [ ] Backup encryption configured
- [ ] Git integration reviewed

### Daily Operations
- [ ] Use minimum required environment
- [ ] Never log secret values
- [ ] Verify environment before operations
- [ ] Review changes before committing

### Weekly Maintenance
- [ ] Review access logs
- [ ] Check for unused secrets
- [ ] Verify backup integrity
- [ ] Update team on security changes

### Monthly Reviews
- [ ] Rotate development passwords
- [ ] Audit user access
- [ ] Review security patches
- [ ] Test incident response

### Quarterly Assessment
- [ ] Rotate staging passwords
- [ ] Security training refresh
- [ ] Update response playbook
- [ ] External security review

### Annual Security
- [ ] Rotate production passwords
- [ ] Full security audit
- [ ] Penetration testing
- [ ] Policy updates

## Additional Security Resources

### VaultEnv-Specific Security

- Use deterministic encryption for searchable fields only
- Enable file integrity checks
- Regular security updates: `go get -u github.com/yourusername/vaultenv`

### General Best Practices

1. **Defense in Depth**
   - Multiple security layers
   - Assume breach methodology
   - Regular security training

2. **Least Privilege**
   - Minimal access rights
   - Time-bound access
   - Regular access reviews

3. **Security Automation**
   - Automated secret scanning
   - CI/CD security checks
   - Automated rotation

### Reporting Security Issues

Found a security vulnerability in VaultEnv?

**Do NOT** create a public issue. Instead:
1. Email: security@vaultenv.com
2. Use responsible disclosure
3. Allow 90 days for fix

We appreciate security researchers and provide acknowledgment for valid reports.

## Conclusion

Security is not a feature, it's a practice. VaultEnv provides the tools, but effective security requires:
- Consistent application of best practices
- Regular reviews and updates
- Team training and awareness
- Proactive threat modeling

Stay secure, stay vigilant.