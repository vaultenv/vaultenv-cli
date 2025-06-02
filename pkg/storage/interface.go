package storage

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNotFound      = errors.New("variable not found")
	ErrAlreadyExists = errors.New("variable already exists")
	ErrInvalidName   = errors.New("invalid variable name")
)

// Backend defines the interface for storage implementations
type Backend interface {
	// Set stores a variable with optional encryption
	Set(key, value string, encrypt bool) error

	// Get retrieves a variable value
	Get(key string) (string, error)

	// Exists checks if a variable exists
	Exists(key string) (bool, error)

	// Delete removes a variable
	Delete(key string) error

	// List returns all variable names
	List() ([]string, error)

	// Close closes the storage backend
	Close() error
}

// testBackend is used for testing to override the default backend
var testBackend Backend

// SetTestBackend sets a backend to use during testing
func SetTestBackend(backend Backend) {
	testBackend = backend
}

// ResetTestBackend clears the test backend
func ResetTestBackend() {
	testBackend = nil
}

// BackendOptions contains options for creating a backend
type BackendOptions struct {
	Environment string
	Password    string // Optional: if provided, backend will be encrypted
	Type        string // Optional: backend type ("file", "sqlite"), defaults to "file"
	BasePath    string // Optional: base path for storage, defaults to ".vaultenv"
}

// GetBackend returns a storage backend for the given environment
func GetBackend(environment string) (Backend, error) {
	return GetBackendWithOptions(BackendOptions{
		Environment: environment,
	})
}

// GetBackendWithOptions returns a storage backend with the given options
func GetBackendWithOptions(opts BackendOptions) (Backend, error) {
	// Use test backend if set
	if testBackend != nil {
		return testBackend, nil
	}

	// Set defaults
	if opts.BasePath == "" {
		opts.BasePath = ".vaultenv"
	}
	if opts.Type == "" {
		opts.Type = "file"
	}

	// Create base backend based on type
	var baseBackend Backend
	var err error

	switch opts.Type {
	case "sqlite":
		baseBackend, err = NewSQLiteBackend(opts.BasePath, opts.Environment)
	case "file":
		baseBackend, err = NewFileBackend(opts.BasePath, opts.Environment)
	case "git":
		baseBackend, err = NewGitBackend(opts.BasePath, opts.Environment)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", opts.Type)
	}

	if err != nil {
		return nil, err
	}

	// If password is provided, wrap with encryption
	if opts.Password != "" {
		return NewEncryptedBackend(baseBackend, opts.Password)
	}

	return baseBackend, nil
}
