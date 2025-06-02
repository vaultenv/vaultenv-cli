package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vaultenv/vaultenv-cli/pkg/encryption"
)

// EncryptedValue represents an encrypted value with metadata
type EncryptedValue struct {
	Algorithm   string `json:"algorithm"`
	Version     int    `json:"version"`
	Salt        string `json:"salt"`       // Base64 encoded
	Nonce       string `json:"nonce"`      // Base64 encoded
	Ciphertext  string `json:"ciphertext"` // Base64 encoded
	CreatedAt   int64  `json:"created_at"`
	IsEncrypted bool   `json:"is_encrypted"`
}

// EncryptedBackend wraps any storage backend with transparent encryption
type EncryptedBackend struct {
	backend   Backend
	encryptor encryption.Encryptor
	key       []byte
}

// NewEncryptedBackend creates a new encrypted storage backend
func NewEncryptedBackend(backend Backend, password string) (*EncryptedBackend, error) {
	if backend == nil {
		return nil, fmt.Errorf("backend cannot be nil")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Use default encryptor (AES-GCM-256)
	encryptor := encryption.DefaultEncryptor()

	// For the master key, we use a fixed salt derived from the password itself
	// This ensures consistent key derivation across instances
	masterSalt := []byte("vaultenv-master-salt-v1")

	// Derive key from password
	key := encryptor.GenerateKey(password, masterSalt)

	return &EncryptedBackend{
		backend:   backend,
		encryptor: encryptor,
		key:       key,
	}, nil
}

// NewEncryptedBackendWithEncryptor creates an encrypted backend with a specific encryptor
func NewEncryptedBackendWithEncryptor(backend Backend, password string, encryptor encryption.Encryptor) (*EncryptedBackend, error) {
	if backend == nil {
		return nil, fmt.Errorf("backend cannot be nil")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor cannot be nil")
	}

	// For the master key, we use a fixed salt derived from the password itself
	// This ensures consistent key derivation across instances
	masterSalt := []byte("vaultenv-master-salt-v1")

	// Derive key from password
	key := encryptor.GenerateKey(password, masterSalt)

	return &EncryptedBackend{
		backend:   backend,
		encryptor: encryptor,
		key:       key,
	}, nil
}

// Set stores a variable with optional encryption
func (e *EncryptedBackend) Set(key, value string, encrypt bool) error {
	if !encrypt {
		// Store as plain text with metadata indicating it's not encrypted
		ev := EncryptedValue{
			Algorithm:   e.encryptor.Algorithm(),
			Version:     1,
			IsEncrypted: false,
			Ciphertext:  value, // Store plaintext in ciphertext field
			CreatedAt:   time.Now().Unix(),
		}

		// Marshal to JSON
		data, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}

		return e.backend.Set(key, string(data), false)
	}

	// Generate new salt for this value
	salt, err := e.encryptor.GenerateSalt()
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key for this specific value
	valueKey := e.encryptor.GenerateKey(string(e.key), salt)

	// Encrypt the value
	ciphertext, err := e.encryptor.Encrypt([]byte(value), valueKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	// Create encrypted value with metadata
	ev := EncryptedValue{
		Algorithm:   e.encryptor.Algorithm(),
		Version:     1,
		Salt:        base64.StdEncoding.EncodeToString(salt),
		Ciphertext:  base64.StdEncoding.EncodeToString(ciphertext),
		IsEncrypted: true,
		CreatedAt:   time.Now().Unix(),
	}

	// Marshal to JSON
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted value: %w", err)
	}

	// Store in backend
	return e.backend.Set(key, string(data), false)
}

// Get retrieves and decrypts a variable value
func (e *EncryptedBackend) Get(key string) (string, error) {
	// Get from backend
	data, err := e.backend.Get(key)
	if err != nil {
		return "", err
	}

	// Try to unmarshal as encrypted value
	var ev EncryptedValue
	if err := json.Unmarshal([]byte(data), &ev); err != nil {
		// If unmarshal fails, assume it's legacy plaintext
		return data, nil
	}

	// If not encrypted, return the plaintext
	if !ev.IsEncrypted {
		return ev.Ciphertext, nil
	}

	// Decode salt
	salt, err := base64.StdEncoding.DecodeString(ev.Salt)
	if err != nil {
		return "", fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decode ciphertext
	ciphertext, err := base64.StdEncoding.DecodeString(ev.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Get the appropriate encryptor
	encryptor := e.encryptor
	if ev.Algorithm != e.encryptor.Algorithm() {
		// Try to create encryptor for the stored algorithm
		encryptor, err = encryption.NewEncryptor(ev.Algorithm)
		if err != nil {
			return "", fmt.Errorf("unsupported algorithm %s: %w", ev.Algorithm, err)
		}
	}

	// Derive key for this specific value
	valueKey := encryptor.GenerateKey(string(e.key), salt)

	// Decrypt
	plaintext, err := encryptor.Decrypt(ciphertext, valueKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt value: %w", err)
	}

	return string(plaintext), nil
}

// Exists checks if a variable exists
func (e *EncryptedBackend) Exists(key string) (bool, error) {
	return e.backend.Exists(key)
}

// Delete removes a variable
func (e *EncryptedBackend) Delete(key string) error {
	return e.backend.Delete(key)
}

// List returns all variable names
func (e *EncryptedBackend) List() ([]string, error) {
	return e.backend.List()
}

// Close closes the storage backend
func (e *EncryptedBackend) Close() error {
	return e.backend.Close()
}

// UpdatePassword changes the encryption password for all encrypted values
func (e *EncryptedBackend) UpdatePassword(oldPassword, newPassword string) error {
	if oldPassword == "" || newPassword == "" {
		return fmt.Errorf("passwords cannot be empty")
	}

	// Get all keys
	keys, err := e.backend.List()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	// Store decrypted values
	valuesToReencrypt := make(map[string]string)

	// First pass: decrypt all encrypted values with old password
	for _, key := range keys {
		// Get raw data
		data, err := e.backend.Get(key)
		if err != nil {
			return fmt.Errorf("failed to get raw data for %s: %w", key, err)
		}

		var ev EncryptedValue
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			// Skip non-encrypted values
			continue
		}

		if !ev.IsEncrypted {
			// Skip non-encrypted values
			continue
		}

		// Get decrypted value
		value, err := e.Get(key)
		if err != nil {
			return fmt.Errorf("failed to decrypt %s: %w", key, err)
		}

		valuesToReencrypt[key] = value
	}

	// Use the same fixed master salt for key derivation
	masterSalt := []byte("vaultenv-master-salt-v1")
	newKey := e.encryptor.GenerateKey(newPassword, masterSalt)

	// Store old key for rollback
	oldKey := e.key

	// Update to new key
	e.key = newKey

	// Second pass: re-encrypt all values with new password
	for key, value := range valuesToReencrypt {
		if err := e.Set(key, value, true); err != nil {
			// Restore old key on failure
			e.key = oldKey
			return fmt.Errorf("failed to re-encrypt %s: %w", key, err)
		}
	}

	return nil
}
