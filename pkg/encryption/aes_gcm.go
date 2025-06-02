package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// AESGCMEncryptor implements AES-256-GCM encryption
type AESGCMEncryptor struct {
	// Key derivation parameters (chosen for security/performance balance)
	iterations uint32
	memory     uint32
	threads    uint8
	keyLength  uint32
}

// NewAESGCMEncryptor creates a new AES-GCM encryptor with secure defaults
func NewAESGCMEncryptor() *AESGCMEncryptor {
	return &AESGCMEncryptor{
		iterations: 3,         // Number of iterations
		memory:     64 * 1024, // Memory in KiB (64 MB)
		threads:    4,         // Number of threads
		keyLength:  32,        // Key length in bytes (256 bits)
	}
}

// Algorithm returns the algorithm identifier
func (e *AESGCMEncryptor) Algorithm() string {
	return "aes-gcm-256"
}

// GenerateSalt creates a cryptographically secure random salt
func (e *AESGCMEncryptor) GenerateSalt() ([]byte, error) {
	salt := make([]byte, 32) // 256-bit salt
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateKey derives an encryption key from a password using Argon2id
func (e *AESGCMEncryptor) GenerateKey(password string, salt []byte) []byte {
	// Argon2id is the recommended algorithm for password hashing
	// It provides both side-channel resistance (from Argon2i)
	// and GPU cracking resistance (from Argon2d)
	return argon2.IDKey(
		[]byte(password),
		salt,
		e.iterations,
		e.memory,
		e.threads,
		e.keyLength,
	)
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *AESGCMEncryptor) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	// Prepend nonce to ciphertext for storage
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with AES-256-GCM
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Validate ciphertext length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidData
	}

	// Extract nonce and actual ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// Helper functions for string encoding

// EncryptString encrypts a string and returns base64-encoded result
func (e *AESGCMEncryptor) EncryptString(plaintext string, key []byte) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a base64-encoded string
func (e *AESGCMEncryptor) DecryptString(ciphertext string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}

	plaintext, err := e.Decrypt(data, key)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptWithNonce encrypts plaintext using AES-256-GCM with a specific nonce
// This is used by DeterministicEncryptor to provide consistent encryption
func (e *AESGCMEncryptor) EncryptWithNonce(plaintext []byte, key []byte, nonce []byte) ([]byte, error) {
	// Validate key length
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Validate nonce size
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: got %d, want %d", len(nonce), gcm.NonceSize())
	}

	// Encrypt data
	// Prepend nonce to ciphertext for storage (same format as regular Encrypt)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}
