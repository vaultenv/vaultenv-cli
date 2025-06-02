package encryption

import (
	"errors"
)

// Common errors
var (
	ErrInvalidKey       = errors.New("invalid encryption key")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrInvalidData      = errors.New("invalid encrypted data")
)

// Encryptor defines the interface for encryption implementations
// This allows us to swap algorithms if needed while maintaining compatibility
type Encryptor interface {
	// Encrypt encrypts plaintext using the provided key
	Encrypt(plaintext []byte, key []byte) ([]byte, error)

	// Decrypt decrypts ciphertext using the provided key
	Decrypt(ciphertext []byte, key []byte) ([]byte, error)

	// EncryptString encrypts a string and returns base64-encoded result
	EncryptString(plaintext string, key []byte) (string, error)

	// DecryptString decrypts a base64-encoded string
	DecryptString(ciphertext string, key []byte) (string, error)

	// GenerateKey derives an encryption key from a password
	GenerateKey(password string, salt []byte) []byte

	// GenerateSalt creates a new random salt
	GenerateSalt() ([]byte, error)

	// Algorithm returns the name of the encryption algorithm
	Algorithm() string
}

// Metadata contains information about encrypted data
// This helps with key rotation and algorithm upgrades
type Metadata struct {
	Algorithm string `json:"algorithm"`
	Version   int    `json:"version"`
	Salt      []byte `json:"salt"`
	Nonce     []byte `json:"nonce,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

// EncryptedData represents encrypted content with metadata
type EncryptedData struct {
	Metadata   Metadata `json:"metadata"`
	Ciphertext []byte   `json:"ciphertext"`
}

// Factory creates an encryptor based on algorithm name
func NewEncryptor(algorithm string) (Encryptor, error) {
	switch algorithm {
	case "aes-gcm-256":
		return NewAESGCMEncryptor(), nil
	case "aes-gcm-256-deterministic":
		return NewDeterministicEncryptor(), nil
	case "chacha20-poly1305":
		return NewChaChaEncryptor(), nil
	default:
		return nil, errors.New("unsupported algorithm: " + algorithm)
	}
}

// DefaultEncryptor returns the default encryption algorithm
func DefaultEncryptor() Encryptor {
	return NewAESGCMEncryptor()
}
