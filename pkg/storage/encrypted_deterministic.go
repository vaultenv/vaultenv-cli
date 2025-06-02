package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vaultenv/vaultenv-cli/pkg/encryption"
)

// DeterministicEncryptedBackend extends EncryptedBackend with deterministic encryption support
type DeterministicEncryptedBackend struct {
	*EncryptedBackend
	deterministicEncryptor *encryption.DeterministicEncryptor
	useDeterministic       bool
}

// NewDeterministicEncryptedBackend creates a new encrypted backend with optional deterministic mode
func NewDeterministicEncryptedBackend(backend Backend, password string, useDeterministic bool) (*DeterministicEncryptedBackend, error) {
	// Create base encrypted backend
	base, err := NewEncryptedBackend(backend, password)
	if err != nil {
		return nil, err
	}
	
	// Create deterministic encryptor if needed
	var deterministicEnc *encryption.DeterministicEncryptor
	if useDeterministic {
		deterministicEnc = encryption.NewDeterministicEncryptor()
	}
	
	return &DeterministicEncryptedBackend{
		EncryptedBackend:       base,
		deterministicEncryptor: deterministicEnc,
		useDeterministic:       useDeterministic,
	}, nil
}

// Set stores a variable with encryption (deterministic if enabled)
func (d *DeterministicEncryptedBackend) Set(key, value string, encrypt bool) error {
	if !encrypt {
		// Use base implementation for unencrypted values
		return d.EncryptedBackend.Set(key, value, false)
	}
	
	if !d.useDeterministic {
		// Use base implementation for regular encryption
		return d.EncryptedBackend.Set(key, value, true)
	}
	
	// Use deterministic encryption
	// For deterministic mode, we don't generate a new salt per value
	// Instead, we use a fixed salt and rely on the context for uniqueness
	salt := []byte("vaultenv-deterministic-salt-v1")
	
	// Derive key for encryption
	valueKey := d.deterministicEncryptor.GenerateKey(string(d.key), salt)
	
	// Use the key name as context for deterministic encryption
	context := []byte(key)
	
	// Encrypt the value deterministically
	ciphertext, err := d.deterministicEncryptor.EncryptDeterministic([]byte(value), valueKey, context)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}
	
	// Create encrypted value with metadata
	ev := EncryptedValue{
		Algorithm:   d.deterministicEncryptor.Algorithm(),
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
	return d.backend.Set(key, string(data), false)
}

// SetWithEnvironment stores a variable with environment-specific context for deterministic encryption
func (d *DeterministicEncryptedBackend) SetWithEnvironment(environment, key, value string, encrypt bool) error {
	if !encrypt || !d.useDeterministic {
		// Use regular Set for non-deterministic mode
		return d.Set(key, value, encrypt)
	}
	
	// Use deterministic encryption with environment context
	salt := []byte("vaultenv-deterministic-salt-v1")
	
	// Derive key for encryption
	valueKey := d.deterministicEncryptor.GenerateKey(string(d.key), salt)
	
	// Use environment + key as context for deterministic encryption
	// This ensures same value in different environments gets different ciphertext
	context := []byte(fmt.Sprintf("%s:%s", environment, key))
	
	// Encrypt the value deterministically
	ciphertext, err := d.deterministicEncryptor.EncryptDeterministic([]byte(value), valueKey, context)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}
	
	// Create encrypted value with metadata
	ev := EncryptedValue{
		Algorithm:   d.deterministicEncryptor.Algorithm(),
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
	return d.backend.Set(key, string(data), false)
}