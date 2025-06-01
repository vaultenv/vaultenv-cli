package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaultenv/vaultenv-cli/internal/keystore"
)

// Example integration showing how to use the PasswordManager
// This is not part of the production code, but serves as documentation

// InitializeAuth initializes the authentication system for a project
func InitializeAuth(dataDir, projectID string) (*PasswordManager, error) {
	// Create keystore
	ks, err := keystore.NewKeystore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create keystore: %w", err)
	}
	
	// Create password manager
	pm := NewPasswordManager(ks)
	
	return pm, nil
}

// ExampleUsage shows how to use the password manager in CLI commands
func ExampleUsage() {
	// Get data directory (usually from config)
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".vaultenv", "data")
	
	projectID := "my-project"
	
	// Initialize auth
	pm, err := InitializeAuth(dataDir, projectID)
	if err != nil {
		fmt.Printf("Failed to initialize auth: %v\n", err)
		return
	}
	
	// Get or create master key for the project
	// This will prompt for password if needed
	key, err := pm.GetOrCreateMasterKey(projectID)
	if err != nil {
		fmt.Printf("Failed to get master key: %v\n", err)
		return
	}
	
	// Now you can use the key for encryption/decryption
	fmt.Printf("Got encryption key (length: %d bytes)\n", len(key))
	
	// The key is cached for the session, so subsequent calls won't prompt
	key2, err := pm.GetOrCreateMasterKey(projectID)
	if err != nil {
		fmt.Printf("Failed to get cached key: %v\n", err)
		return
	}
	
	fmt.Printf("Keys match: %v\n", string(key) == string(key2))
}

// ExampleCLIIntegration shows how to integrate with Cobra commands
type AuthenticatedCommand struct {
	pm        *PasswordManager
	projectID string
}

func (ac *AuthenticatedCommand) Execute() error {
	// Get encryption key
	key, err := ac.pm.GetOrCreateMasterKey(ac.projectID)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	// Use the key for your command's operations
	_ = key
	
	return nil
}

// ExampleEnvironmentVariable shows how to use environment variable for CI/CD
func ExampleEnvironmentVariable(pm *PasswordManager, projectID string) error {
	// Check if password is in environment
	if password, ok := pm.GetPasswordFromEnv(); ok {
		// Verify the password
		if err := pm.VerifyPassword(projectID, password); err != nil {
			return fmt.Errorf("invalid password from environment: %w", err)
		}
		
		// Get the key entry to derive the actual key
		ks := pm.keystore
		keyEntry, err := ks.GetKey(projectID)
		if err != nil {
			return fmt.Errorf("failed to get key entry: %w", err)
		}
		
		// Derive the key
		key := pm.DeriveKey(password, keyEntry.Salt)
		
		// Cache it for the session
		pm.cacheSessionKey(projectID, key)
		
		fmt.Println("Successfully authenticated using environment variable")
		return nil
	}
	
	// Fall back to interactive prompt
	_, err := pm.GetOrCreateMasterKey(projectID)
	return err
}