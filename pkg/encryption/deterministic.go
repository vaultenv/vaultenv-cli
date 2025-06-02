package encryption

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// DeterministicEncryptor provides consistent encryption for version control
// This wraps an existing encryptor to provide deterministic nonce generation
// while maintaining security through context-based nonce derivation
type DeterministicEncryptor struct {
	baseEncryptor Encryptor
}

// NewDeterministicEncryptor creates a new deterministic encryptor
// It uses the existing AES-GCM encryptor as the base
func NewDeterministicEncryptor() *DeterministicEncryptor {
	return &DeterministicEncryptor{
		baseEncryptor: NewAESGCMEncryptor(),
	}
}

// EncryptDeterministic produces the same ciphertext for the same plaintext
// This is safe because we're using unique keys per environment and context-based nonces
func (d *DeterministicEncryptor) EncryptDeterministic(plaintext []byte, key []byte, context []byte) ([]byte, error) {
	// Derive a deterministic nonce from the plaintext and context
	// This is safe when using unique keys per environment
	nonce := d.deriveNonce(plaintext, key, context)
	
	// Use the AES-GCM encryptor's EncryptWithNonce method
	gcmEncryptor, ok := d.baseEncryptor.(*AESGCMEncryptor)
	if !ok {
		return nil, fmt.Errorf("base encryptor must be AESGCMEncryptor for deterministic mode")
	}
	
	return gcmEncryptor.EncryptWithNonce(plaintext, key, nonce)
}

// DecryptDeterministic decrypts data encrypted with EncryptDeterministic
func (d *DeterministicEncryptor) DecryptDeterministic(ciphertext []byte, key []byte, context []byte) ([]byte, error) {
	// For decryption, we just use the base decryptor as the nonce is included in the ciphertext
	return d.baseEncryptor.Decrypt(ciphertext, key)
}

// deriveNonce creates a deterministic nonce from inputs
// Using HMAC ensures the nonce is unpredictable without the key
func (d *DeterministicEncryptor) deriveNonce(plaintext, key, context []byte) []byte {
	// Create HMAC with the key
	h := hmac.New(sha256.New, key)
	
	// Write context first (e.g., variable name, environment)
	h.Write(context)
	
	// Write separator to prevent collision between context and plaintext
	h.Write([]byte{0x00})
	
	// Write plaintext
	h.Write(plaintext)
	
	// Get hash and use first 12 bytes for GCM nonce
	hash := h.Sum(nil)
	return hash[:12]
}

// Implement Encryptor interface methods to make it a drop-in replacement

// Encrypt encrypts plaintext using deterministic mode with empty context
func (d *DeterministicEncryptor) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	// Use empty context for backward compatibility
	return d.EncryptDeterministic(plaintext, key, []byte{})
}

// EncryptString encrypts a string using deterministic mode
func (d *DeterministicEncryptor) EncryptString(plaintext string, key []byte) (string, error) {
	encrypted, err := d.EncryptDeterministic([]byte(plaintext), key, []byte{})
	if err != nil {
		return "", err
	}
	// Convert to base64 string
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts ciphertext
func (d *DeterministicEncryptor) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	return d.baseEncryptor.Decrypt(ciphertext, key)
}

// DecryptString decrypts a string
func (d *DeterministicEncryptor) DecryptString(ciphertext string, key []byte) (string, error) {
	// Decode from base64 string
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}
	
	plaintext, err := d.Decrypt(data, key)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
}

// Algorithm returns the encryption algorithm name
func (d *DeterministicEncryptor) Algorithm() string {
	return "aes-gcm-256-deterministic"
}

// GenerateKey derives an encryption key from a password using the base encryptor
func (d *DeterministicEncryptor) GenerateKey(password string, salt []byte) []byte {
	return d.baseEncryptor.GenerateKey(password, salt)
}

// GenerateSalt creates a new random salt using the base encryptor
func (d *DeterministicEncryptor) GenerateSalt() ([]byte, error) {
	return d.baseEncryptor.GenerateSalt()
}