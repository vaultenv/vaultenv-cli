package keystore

import (
	"fmt"
	"runtime"

	"github.com/99designs/keyring"
)

// Keystore provides secure storage for encryption keys
type Keystore interface {
	// Store saves a key securely
	Store(service, account string, data []byte) error

	// Retrieve gets a stored key
	Retrieve(service, account string) ([]byte, error)

	// Delete removes a stored key
	Delete(service, account string) error

	// List returns all stored keys for a service
	List(service string) ([]string, error)
}

// OSKeystore uses the operating system's secure storage
type OSKeystore struct {
	ring keyring.Keyring
}

// NewOSKeystore creates a keystore using OS facilities
func NewOSKeystore(appName string) (*OSKeystore, error) {
	// Configure keyring with appropriate backends for each OS
	config := keyring.Config{
		ServiceName: appName,

		// Try these backends in order
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,      // macOS Keychain
			keyring.WinCredBackend,       // Windows Credential Manager
			keyring.SecretServiceBackend, // Linux Secret Service (GNOME/KDE)
			keyring.KWalletBackend,       // KDE Wallet (legacy)
			keyring.FileBackend,          // Encrypted file (fallback)
		},

		// Prompt for password if using file backend
		FilePasswordFunc: keyring.TerminalPrompt,

		// Use a consistent file location
		FileDir: "~/.vaultenv-cli/keyring",

		// Key names can contain these characters
		KeychainName: "vaultenv-cli",

		// Windows-specific settings
		WinCredPrefix: "vaultenv-cli",
	}

	ring, err := keyring.Open(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &OSKeystore{ring: ring}, nil
}

// Store saves a key securely
func (k *OSKeystore) Store(service, account string, data []byte) error {
	return k.ring.Set(keyring.Item{
		Key:         k.makeKey(service, account),
		Data:        data,
		Label:       fmt.Sprintf("vaultenv-cli key for %s", account),
		Description: fmt.Sprintf("Encryption key for %s environment", account),
	})
}

// Retrieve gets a stored key
func (k *OSKeystore) Retrieve(service, account string) ([]byte, error) {
	item, err := k.ring.Get(k.makeKey(service, account))
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, fmt.Errorf("key not found for %s", account)
		}
		return nil, fmt.Errorf("failed to retrieve key: %w", err)
	}

	return item.Data, nil
}

// Delete removes a stored key
func (k *OSKeystore) Delete(service, account string) error {
	err := k.ring.Remove(k.makeKey(service, account))
	if err != nil && err != keyring.ErrKeyNotFound {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	return nil
}

// List returns all stored keys for a service
func (k *OSKeystore) List(service string) ([]string, error) {
	keys, err := k.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var accounts []string
	prefix := service + ":"

	for _, key := range keys {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			account := key[len(prefix):]
			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

// makeKey creates a consistent key format
func (k *OSKeystore) makeKey(service, account string) string {
	return fmt.Sprintf("%s:%s", service, account)
}

// GetBackend returns the active keyring backend name
func (k *OSKeystore) GetBackend() string {
	// This helps with debugging and user support
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain"
	case "windows":
		return "Windows Credential Manager"
	case "linux":
		// Try to detect which backend is actually in use
		if runtime.GOARCH == "amd64" {
			return "Secret Service"
		}
		return "Encrypted File"
	default:
		return "Encrypted File"
	}
}
