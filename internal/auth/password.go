package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

const (
	// Argon2 parameters for key derivation
	argon2Time    = 3
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32 // 256-bit key

	// Salt length
	saltLen = 32

	// Session key cache duration
	sessionCacheDuration = 15 * time.Minute
)

var (
	ErrInvalidPassword    = errors.New("invalid password")
	ErrPasswordMismatch   = errors.New("passwords do not match")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrNoPasswordProvided = errors.New("no password provided")
)

// PasswordManager handles password operations and key derivation
type PasswordManager struct {
	keystore     *keystore.Keystore
	environmentKeyManager *keystore.EnvironmentKeyManager
	config       *config.Config
	sessionCache map[string]*sessionEntry
}

type sessionEntry struct {
	key       []byte
	expiresAt time.Time
}

// NewPasswordManager creates a new password manager instance
func NewPasswordManager(ks *keystore.Keystore, cfg *config.Config) *PasswordManager {
	envKeyManager := keystore.NewEnvironmentKeyManager(ks, cfg.Project.ID)
	return &PasswordManager{
		keystore:              ks,
		environmentKeyManager: envKeyManager,
		config:               cfg,
		sessionCache:          make(map[string]*sessionEntry),
	}
}

// PromptPassword prompts the user for a password with the given prompt message
func (pm *PasswordManager) PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	
	// Read password from terminal without echoing
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	
	passwordStr := string(password)
	if passwordStr == "" {
		return "", ErrNoPasswordProvided
	}
	
	return passwordStr, nil
}

// PromptNewPassword prompts for a new password with confirmation
func (pm *PasswordManager) PromptNewPassword() (string, error) {
	password, err := pm.PromptPassword("Enter password: ")
	if err != nil {
		return "", err
	}
	
	if len(password) < 8 {
		return "", ErrPasswordTooShort
	}
	
	confirm, err := pm.PromptPassword("Confirm password: ")
	if err != nil {
		return "", err
	}
	
	if password != confirm {
		return "", ErrPasswordMismatch
	}
	
	return password, nil
}

// DeriveKey derives an encryption key from a password using Argon2
func (pm *PasswordManager) DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)
}

// GenerateSalt generates a new random salt
func (pm *PasswordManager) GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GetOrCreateMasterKey gets the master key for a project, creating it if necessary
func (pm *PasswordManager) GetOrCreateMasterKey(projectID string) ([]byte, error) {
	// Check session cache first
	cacheKey := pm.getCacheKey(projectID)
	if entry, ok := pm.sessionCache[cacheKey]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.key, nil
		}
		// Clean up expired entry
		delete(pm.sessionCache, cacheKey)
	}
	
	// Try to get existing key from keystore
	existingKey, err := pm.keystore.GetKey(projectID)
	if err == nil && existingKey != nil {
		// Verify with password
		password, err := pm.PromptPassword("Enter password: ")
		if err != nil {
			return nil, err
		}
		
		key := pm.DeriveKey(password, existingKey.Salt)
		
		// Verify the key by checking the verification hash
		if !pm.verifyKey(key, existingKey.VerificationHash) {
			return nil, ErrInvalidPassword
		}
		
		// Cache the key for the session
		pm.cacheSessionKey(projectID, key)
		
		return key, nil
	}
	
	// Create new key
	fmt.Println("Creating new master key for project...")
	password, err := pm.PromptNewPassword()
	if err != nil {
		return nil, err
	}
	
	salt, err := pm.GenerateSalt()
	if err != nil {
		return nil, err
	}
	
	key := pm.DeriveKey(password, salt)
	
	// Generate verification hash
	verificationHash := pm.generateVerificationHash(key)
	
	// Store in keystore
	keyEntry := &keystore.KeyEntry{
		ProjectID:        projectID,
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	}
	
	if err := pm.keystore.StoreKey(projectID, keyEntry); err != nil {
		return nil, fmt.Errorf("failed to store key: %w", err)
	}
	
	// Cache the key for the session
	pm.cacheSessionKey(projectID, key)
	
	return key, nil
}

// VerifyPassword verifies a password against stored key data
func (pm *PasswordManager) VerifyPassword(projectID, password string) error {
	keyEntry, err := pm.keystore.GetKey(projectID)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}
	
	key := pm.DeriveKey(password, keyEntry.Salt)
	
	if !pm.verifyKey(key, keyEntry.VerificationHash) {
		return ErrInvalidPassword
	}
	
	return nil
}

// ChangePassword changes the password for a project
func (pm *PasswordManager) ChangePassword(projectID string) error {
	// Verify current password
	currentPassword, err := pm.PromptPassword("Enter current password: ")
	if err != nil {
		return err
	}
	
	if err := pm.VerifyPassword(projectID, currentPassword); err != nil {
		return err
	}
	
	// Get new password
	newPassword, err := pm.PromptNewPassword()
	if err != nil {
		return err
	}
	
	// Generate new salt
	salt, err := pm.GenerateSalt()
	if err != nil {
		return err
	}
	
	// Derive new key
	newKey := pm.DeriveKey(newPassword, salt)
	
	// Generate new verification hash
	verificationHash := pm.generateVerificationHash(newKey)
	
	// Update keystore
	keyEntry := &keystore.KeyEntry{
		ProjectID:        projectID,
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	}
	
	if err := pm.keystore.StoreKey(projectID, keyEntry); err != nil {
		return fmt.Errorf("failed to update key: %w", err)
	}
	
	// Clear session cache for this project
	delete(pm.sessionCache, pm.getCacheKey(projectID))
	
	fmt.Println("Password changed successfully")
	return nil
}

// ClearSessionCache clears all cached session keys
func (pm *PasswordManager) ClearSessionCache() {
	pm.sessionCache = make(map[string]*sessionEntry)
}

// ClearProjectCache clears cached session key for a specific project
func (pm *PasswordManager) ClearProjectCache(projectID string) {
	delete(pm.sessionCache, pm.getCacheKey(projectID))
}

// Helper methods

func (pm *PasswordManager) generateVerificationHash(key []byte) string {
	// Create a verification string by hashing the key with a known prefix
	verifier := append([]byte("vaultenv-verify:"), key...)
	hash := sha256.Sum256(verifier)
	return hex.EncodeToString(hash[:])
}

func (pm *PasswordManager) verifyKey(key []byte, verificationHash string) bool {
	expectedHash := pm.generateVerificationHash(key)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(verificationHash)) == 1
}

func (pm *PasswordManager) cacheSessionKey(projectID string, key []byte) {
	cacheKey := pm.getCacheKey(projectID)
	pm.sessionCache[cacheKey] = &sessionEntry{
		key:       key,
		expiresAt: time.Now().Add(sessionCacheDuration),
	}
}

func (pm *PasswordManager) getCacheKey(projectID string) string {
	return fmt.Sprintf("project:%s", projectID)
}

func (pm *PasswordManager) getEnvironmentCacheKey(projectID, environment string) string {
	return fmt.Sprintf("project:%s:env:%s", projectID, environment)
}

// GetPasswordFromEnv gets password from environment variable if set
func (pm *PasswordManager) GetPasswordFromEnv() (string, bool) {
	password := os.Getenv("VAULTENV_PASSWORD")
	return password, password != ""
}

// ExportKey exports the encryption key in a safe format for backup
func (pm *PasswordManager) ExportKey(projectID string, password string) (string, error) {
	keyEntry, err := pm.keystore.GetKey(projectID)
	if err != nil {
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	
	// Verify password
	key := pm.DeriveKey(password, keyEntry.Salt)
	if !pm.verifyKey(key, keyEntry.VerificationHash) {
		return "", ErrInvalidPassword
	}
	
	// Create export data
	exportData := fmt.Sprintf("vaultenv:v1:%s:%s",
		base64.StdEncoding.EncodeToString(keyEntry.Salt),
		keyEntry.VerificationHash,
	)
	
	return exportData, nil
}

// ImportKey imports a key from export format
func (pm *PasswordManager) ImportKey(projectID string, exportData string, password string) error {
	parts := strings.Split(exportData, ":")
	if len(parts) != 4 || parts[0] != "vaultenv" || parts[1] != "v1" {
		return errors.New("invalid export format")
	}
	
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("invalid salt format: %w", err)
	}
	
	verificationHash := parts[3]
	
	// Verify the password works with imported data
	key := pm.DeriveKey(password, salt)
	if !pm.verifyKey(key, verificationHash) {
		return ErrInvalidPassword
	}
	
	// Store in keystore
	keyEntry := &keystore.KeyEntry{
		ProjectID:        projectID,
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	}
	
	if err := pm.keystore.StoreKey(projectID, keyEntry); err != nil {
		return fmt.Errorf("failed to store imported key: %w", err)
	}
	
	return nil
}

// GetOrCreateEnvironmentKey gets the encryption key for a specific environment, creating it if necessary
func (pm *PasswordManager) GetOrCreateEnvironmentKey(environment string) ([]byte, error) {
	projectID := pm.config.Project.ID
	
	// If per-environment passwords are disabled, fall back to project-level key
	if !pm.config.IsPerEnvironmentPasswordsEnabled() {
		return pm.GetOrCreateMasterKey(projectID)
	}
	
	// Check session cache first
	cacheKey := pm.getEnvironmentCacheKey(projectID, environment)
	if entry, ok := pm.sessionCache[cacheKey]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.key, nil
		}
		// Clean up expired entry
		delete(pm.sessionCache, cacheKey)
	}
	
	// Try to get existing key from environment-specific keystore
	if pm.environmentKeyManager.HasEnvironmentKey(environment) {
		// Prompt for password with environment context
		password, err := pm.PromptEnvironmentPassword(environment, "Enter password: ")
		if err != nil {
			return nil, err
		}
		
		key, err := pm.environmentKeyManager.GetOrCreateEnvironmentKey(environment, password)
		if err != nil {
			return nil, err
		}
		
		// Cache the key for the session
		pm.cacheEnvironmentKey(projectID, environment, key)
		return key, nil
	}
	
	// Create new environment key
	ui.Info("Creating new encryption key for environment: %s", environment)
	password, err := pm.PromptNewEnvironmentPassword(environment)
	if err != nil {
		return nil, err
	}
	
	key, err := pm.environmentKeyManager.GetOrCreateEnvironmentKey(environment, password)
	if err != nil {
		return nil, err
	}
	
	// Cache the key for the session
	pm.cacheEnvironmentKey(projectID, environment, key)
	return key, nil
}

// PromptEnvironmentPassword prompts for a password for a specific environment
func (pm *PasswordManager) PromptEnvironmentPassword(environment, prompt string) (string, error) {
	// Check environment variable first
	envVar := fmt.Sprintf("VAULTENV_PASSWORD_%s", strings.ToUpper(environment))
	if password := os.Getenv(envVar); password != "" {
		return password, nil
	}
	
	// Fall back to generic environment variable
	if password, exists := pm.GetPasswordFromEnv(); exists {
		return password, nil
	}
	
	// Prompt user with environment context
	fullPrompt := fmt.Sprintf("[%s] %s", environment, prompt)
	return pm.PromptPassword(fullPrompt)
}

// PromptNewEnvironmentPassword prompts for a new password for an environment with policy validation
func (pm *PasswordManager) PromptNewEnvironmentPassword(environment string) (string, error) {
	policy := pm.config.GetPasswordPolicy(environment)
	
	ui.Info("Setting up password for environment: %s", environment)
	if policy.MinLength > 8 {
		ui.Info("Password policy requires: minimum %d characters", policy.MinLength)
		if policy.RequireUpper {
			ui.Info("  - At least one uppercase letter")
		}
		if policy.RequireLower {
			ui.Info("  - At least one lowercase letter")
		}
		if policy.RequireNumbers {
			ui.Info("  - At least one number")
		}
		if policy.RequireSpecial {
			ui.Info("  - At least one special character")
		}
		if policy.PreventCommon {
			ui.Info("  - Cannot be a common password")
		}
	}
	
	for {
		password, err := pm.PromptEnvironmentPassword(environment, "Enter password: ")
		if err != nil {
			return "", err
		}
		
		// Validate against policy
		if err := pm.validatePasswordPolicy(password, policy); err != nil {
			ui.Error("Password validation failed: %v", err)
			continue
		}
		
		confirm, err := pm.PromptEnvironmentPassword(environment, "Confirm password: ")
		if err != nil {
			return "", err
		}
		
		if password != confirm {
			ui.Error("Passwords do not match, please try again")
			continue
		}
		
		return password, nil
	}
}

// validatePasswordPolicy validates a password against the given policy
func (pm *PasswordManager) validatePasswordPolicy(password string, policy config.PassPolicy) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", policy.MinLength)
	}
	
	if policy.RequireUpper && !containsUppercase(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	
	if policy.RequireLower && !containsLowercase(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	
	if policy.RequireNumbers && !containsNumber(password) {
		return fmt.Errorf("password must contain at least one number")
	}
	
	if policy.RequireSpecial && !containsSpecial(password) {
		return fmt.Errorf("password must contain at least one special character")
	}
	
	if policy.PreventCommon && isCommonPassword(password) {
		return fmt.Errorf("this password is too common, please choose a more unique password")
	}
	
	return nil
}

// ChangeEnvironmentPassword changes the password for a specific environment
func (pm *PasswordManager) ChangeEnvironmentPassword(environment string) error {
	if !pm.config.IsPerEnvironmentPasswordsEnabled() {
		return fmt.Errorf("per-environment passwords are not enabled for this project")
	}
	
	// Verify current password
	currentPassword, err := pm.PromptEnvironmentPassword(environment, "Enter current password: ")
	if err != nil {
		return err
	}
	
	// Verify current password by attempting to derive key
	_, err = pm.environmentKeyManager.GetOrCreateEnvironmentKey(environment, currentPassword)
	if err != nil {
		return fmt.Errorf("current password is incorrect: %w", err)
	}
	
	// Get new password
	newPassword, err := pm.PromptNewEnvironmentPassword(environment)
	if err != nil {
		return err
	}
	
	// Change password using environment key manager
	err = pm.environmentKeyManager.ChangeEnvironmentPassword(environment, currentPassword, newPassword)
	if err != nil {
		return fmt.Errorf("failed to change password: %w", err)
	}
	
	// Clear session cache for this environment
	cacheKey := pm.getEnvironmentCacheKey(pm.config.Project.ID, environment)
	delete(pm.sessionCache, cacheKey)
	
	ui.Success("Password changed successfully for environment: %s", environment)
	return nil
}

// cacheEnvironmentKey caches an environment-specific key for the session
func (pm *PasswordManager) cacheEnvironmentKey(projectID, environment string, key []byte) {
	cacheKey := pm.getEnvironmentCacheKey(projectID, environment)
	pm.sessionCache[cacheKey] = &sessionEntry{
		key:       key,
		expiresAt: time.Now().Add(sessionCacheDuration),
	}
}

// ClearEnvironmentCache clears cached session key for a specific environment
func (pm *PasswordManager) ClearEnvironmentCache(environment string) {
	cacheKey := pm.getEnvironmentCacheKey(pm.config.Project.ID, environment)
	delete(pm.sessionCache, cacheKey)
}

// Helper functions for password validation

func containsUppercase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func containsLowercase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}

func containsNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsSpecial(s string) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	for _, r := range s {
		for _, special := range specialChars {
			if r == special {
				return true
			}
		}
	}
	return false
}

func isCommonPassword(password string) bool {
	// Basic list of common passwords - in production, this would be a comprehensive list
	commonPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty", 
		"letmein", "welcome", "monkey", "1234567890", "abc123",
		"Password1", "password1", "123456789", "welcome123",
	}
	
	lowerPassword := strings.ToLower(password)
	for _, common := range commonPasswords {
		if lowerPassword == strings.ToLower(common) {
			return true
		}
	}
	return false
}