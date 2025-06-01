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

	"github.com/vaultenv/vaultenv-cli/internal/keystore"
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
	sessionCache map[string]*sessionEntry
}

type sessionEntry struct {
	key       []byte
	expiresAt time.Time
}

// NewPasswordManager creates a new password manager instance
func NewPasswordManager(ks *keystore.Keystore) *PasswordManager {
	return &PasswordManager{
		keystore:     ks,
		sessionCache: make(map[string]*sessionEntry),
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