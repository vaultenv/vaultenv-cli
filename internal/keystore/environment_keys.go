package keystore

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/ui"
	"github.com/vaultenv/vaultenv-cli/pkg/encryption"
	"golang.org/x/crypto/argon2"
)

// EnvironmentKeyManager handles per-environment encryption keys
type EnvironmentKeyManager struct {
	keystore  *Keystore
	projectID string
	encryptor encryption.Encryptor
}

// NewEnvironmentKeyManager creates a new manager for environment-specific keys
func NewEnvironmentKeyManager(keystore *Keystore, projectID string) *EnvironmentKeyManager {
	return &EnvironmentKeyManager{
		keystore:  keystore,
		projectID: projectID,
		encryptor: encryption.NewAESGCMEncryptor(),
	}
}


// GetOrCreateEnvironmentKey retrieves or creates an encryption key for a specific environment
// This method embodies the zero-knowledge principle - the key is derived from the user's
// password and never stored in plaintext
func (ekm *EnvironmentKeyManager) GetOrCreateEnvironmentKey(environment, password string) ([]byte, error) {
	// Create a unique identifier for this environment's key
	keyID := fmt.Sprintf("%s:%s", ekm.projectID, environment)

	// Try to retrieve existing key metadata
	entry, err := ekm.retrieveEnvironmentKey(keyID)
	if err == nil {
		// Key exists, derive it from password and verify
		return ekm.deriveAndVerifyKey(entry, password)
	}

	// Key doesn't exist, create new one
	ui.Debug("Creating new encryption key for environment: %s", environment)
	return ekm.createNewEnvironmentKey(keyID, environment, password)
}

// retrieveEnvironmentKey gets the key entry for a specific environment
func (ekm *EnvironmentKeyManager) retrieveEnvironmentKey(keyID string) (*EnvironmentKeyEntry, error) {
	// Extract environment from keyID
	environment := keyID[len(ekm.projectID)+1:]
	return ekm.keystore.GetEnvironmentKey(ekm.projectID, environment)
}

// deriveAndVerifyKey derives the encryption key from password and verifies it
func (ekm *EnvironmentKeyManager) deriveAndVerifyKey(entry *EnvironmentKeyEntry, password string) ([]byte, error) {
	// Derive key using stored parameters
	key := ekm.deriveKey(password, entry.Salt, entry.Iterations, entry.Memory, entry.Parallelism)

	// Verify the key by checking the verification hash
	verificationData := append([]byte("vaultenv-verification"), key...)
	verificationHash := base64.StdEncoding.EncodeToString(argon2.IDKey(
		verificationData,
		entry.Salt,
		1, // Single iteration for verification
		64*1024,
		4,
		32,
	))

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(verificationHash), []byte(entry.VerificationHash)) != 1 {
		return nil, fmt.Errorf("invalid password for environment: %s", entry.Environment)
	}

	return key, nil
}

// createNewEnvironmentKey creates a new encryption key for an environment
func (ekm *EnvironmentKeyManager) createNewEnvironmentKey(keyID, environment, password string) ([]byte, error) {
	// Generate a new salt
	salt, err := ekm.encryptor.GenerateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key with strong parameters
	iterations := uint32(3)
	memory := uint32(64 * 1024) // 64MB
	parallelism := uint8(4)

	key := ekm.deriveKey(password, salt, iterations, memory, parallelism)

	// Create verification hash
	verificationData := append([]byte("vaultenv-verification"), key...)
	verificationHash := base64.StdEncoding.EncodeToString(argon2.IDKey(
		verificationData,
		salt,
		1,
		64*1024,
		4,
		32,
	))

	// Create the key entry
	entry := &EnvironmentKeyEntry{
		ProjectID:        ekm.projectID,
		Environment:      environment,
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Algorithm:        "argon2id",
		Iterations:       iterations,
		Memory:           memory,
		Parallelism:      parallelism,
	}

	// Store the key entry
	if err := ekm.storeEnvironmentKey(keyID, entry); err != nil {
		return nil, fmt.Errorf("failed to store key entry: %w", err)
	}

	return key, nil
}

// storeEnvironmentKey saves the key entry to the keystore
func (ekm *EnvironmentKeyManager) storeEnvironmentKey(keyID string, entry *EnvironmentKeyEntry) error {
	return ekm.keystore.StoreEnvironmentKey(ekm.projectID, entry.Environment, entry)
}

// deriveKey derives an encryption key from a password using Argon2id
func (ekm *EnvironmentKeyManager) deriveKey(password string, salt []byte, iterations, memory uint32, parallelism uint8) []byte {
	return argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, 32)
}

// ChangeEnvironmentPassword changes the password for a specific environment
func (ekm *EnvironmentKeyManager) ChangeEnvironmentPassword(environment, oldPassword, newPassword string) error {
	keyID := fmt.Sprintf("%s:%s", ekm.projectID, environment)

	// Verify old password first
	_, err := ekm.GetOrCreateEnvironmentKey(environment, oldPassword)
	if err != nil {
		return fmt.Errorf("current password is incorrect: %w", err)
	}

	// Create new key with new password
	salt, err := ekm.encryptor.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Use same strong parameters
	iterations := uint32(3)
	memory := uint32(64 * 1024)
	parallelism := uint8(4)

	newKey := ekm.deriveKey(newPassword, salt, iterations, memory, parallelism)

	// Create new verification hash
	verificationData := append([]byte("vaultenv-verification"), newKey...)
	verificationHash := base64.StdEncoding.EncodeToString(argon2.IDKey(
		verificationData,
		salt,
		1,
		64*1024,
		4,
		32,
	))

	// Update the key entry
	entry := &EnvironmentKeyEntry{
		ProjectID:        ekm.projectID,
		Environment:      environment,
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(), // Keep original creation time if we had it
		UpdatedAt:        time.Now(),
		Algorithm:        "argon2id",
		Iterations:       iterations,
		Memory:           memory,
		Parallelism:      parallelism,
	}

	// Store updated entry
	return ekm.storeEnvironmentKey(keyID, entry)
}

// DeleteEnvironmentKey removes the key for a specific environment
func (ekm *EnvironmentKeyManager) DeleteEnvironmentKey(environment string) error {
	return ekm.keystore.DeleteEnvironmentKey(ekm.projectID, environment)
}

// ListEnvironmentKeys returns all environments that have keys
func (ekm *EnvironmentKeyManager) ListEnvironmentKeys() ([]string, error) {
	return ekm.keystore.ListEnvironments(ekm.projectID)
}

// HasEnvironmentKey checks if a key exists for the given environment
func (ekm *EnvironmentKeyManager) HasEnvironmentKey(environment string) bool {
	_, err := ekm.keystore.GetEnvironmentKey(ekm.projectID, environment)
	return err == nil
}

// CopyEnvironmentKey copies the key from one environment to another
// This is useful when creating a new environment based on an existing one
func (ekm *EnvironmentKeyManager) CopyEnvironmentKey(sourceEnv, targetEnv, password string) error {
	// Verify password for source environment
	_, err := ekm.GetOrCreateEnvironmentKey(sourceEnv, password)
	if err != nil {
		return fmt.Errorf("invalid password for source environment: %w", err)
	}

	// Get source key entry
	sourceKeyID := fmt.Sprintf("%s:%s", ekm.projectID, sourceEnv)
	sourceEntry, err := ekm.retrieveEnvironmentKey(sourceKeyID)
	if err != nil {
		return fmt.Errorf("failed to retrieve source key: %w", err)
	}

	// Create new entry for target environment
	targetEntry := &EnvironmentKeyEntry{
		ProjectID:        ekm.projectID,
		Environment:      targetEnv,
		Salt:             sourceEntry.Salt,
		VerificationHash: sourceEntry.VerificationHash,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Algorithm:        sourceEntry.Algorithm,
		Iterations:       sourceEntry.Iterations,
		Memory:           sourceEntry.Memory,
		Parallelism:      sourceEntry.Parallelism,
	}

	// Store target entry
	targetKeyID := fmt.Sprintf("%s:%s", ekm.projectID, targetEnv)
	return ekm.storeEnvironmentKey(targetKeyID, targetEntry)
}