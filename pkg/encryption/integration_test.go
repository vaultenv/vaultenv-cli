package encryption

import (
	"testing"

	"github.com/vaultenv/vaultenv-cli/pkg/keystore"
)

// TestSecurityIntegration demonstrates the complete security workflow
func TestSecurityIntegration(t *testing.T) {
	// Create encryption and keystore instances
	encryptor := DefaultEncryptor().(*AESGCMEncryptor)
	keyStore := keystore.NewMockKeystore()

	service := "vaultenv-test"
	account := "production"
	password := "super-secure-password-123!"

	// Sensitive data to protect
	secrets := map[string]string{
		"DATABASE_URL":     "postgresql://user:pass@localhost/db",
		"API_KEY":          "sk-1234567890abcdef",
		"JWT_SECRET":       "my-super-secret-jwt-key",
		"STRIPE_SECRET":    "sk_test_abcdef123456",
		"AWS_SECRET_KEY":   "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Generate salt for key derivation
		salt, err := encryptor.GenerateSalt()
		if err != nil {
			t.Fatalf("Failed to generate salt: %v", err)
		}

		// 2. Derive encryption key from password
		encryptionKey := encryptor.GenerateKey(password, salt)

		// 3. Store the key securely in the keystore
		err = keyStore.Store(service, account, encryptionKey)
		if err != nil {
			t.Fatalf("Failed to store key: %v", err)
		}

		// 4. Encrypt each secret
		encryptedSecrets := make(map[string]string)
		for key, value := range secrets {
			encrypted, err := encryptor.EncryptString(value, encryptionKey)
			if err != nil {
				t.Fatalf("Failed to encrypt %s: %v", key, err)
			}
			encryptedSecrets[key] = encrypted
		}

		// 5. Simulate retrieving and decrypting later
		t.Run("RetrieveAndDecrypt", func(t *testing.T) {
			// Retrieve key from keystore
			retrievedKey, err := keyStore.Retrieve(service, account)
			if err != nil {
				t.Fatalf("Failed to retrieve key: %v", err)
			}

			// Decrypt each secret
			for key, encrypted := range encryptedSecrets {
				decrypted, err := encryptor.DecryptString(encrypted, retrievedKey)
				if err != nil {
					t.Fatalf("Failed to decrypt %s: %v", key, err)
				}

				// Verify it matches original
				if decrypted != secrets[key] {
					t.Errorf("Decrypted value for %s doesn't match original", key)
				}
			}
		})

		// 6. Test with wrong password
		t.Run("WrongPassword", func(t *testing.T) {
			wrongKey := encryptor.GenerateKey("wrong-password", salt)

			// Try to decrypt with wrong key
			for key, encrypted := range encryptedSecrets {
				_, err := encryptor.DecryptString(encrypted, wrongKey)
				if err == nil {
					t.Errorf("Decryption with wrong key should fail for %s", key)
				}
			}
		})

		// 7. Test key rotation
		t.Run("KeyRotation", func(t *testing.T) {
			// Generate new salt and key
			newSalt, _ := encryptor.GenerateSalt()
			newKey := encryptor.GenerateKey(password, newSalt)

			// Re-encrypt with new key
			for key, value := range secrets {
				// Decrypt with old key
				decrypted, err := encryptor.DecryptString(encryptedSecrets[key], encryptionKey)
				if err != nil {
					t.Fatalf("Failed to decrypt %s with old key: %v", key, err)
				}

				// Re-encrypt with new key
				reEncrypted, err := encryptor.EncryptString(decrypted, newKey)
				if err != nil {
					t.Fatalf("Failed to re-encrypt %s: %v", key, err)
				}

				// Verify we can decrypt with new key
				verified, err := encryptor.DecryptString(reEncrypted, newKey)
				if err != nil {
					t.Fatalf("Failed to decrypt %s with new key: %v", key, err)
				}

				if verified != value {
					t.Errorf("Key rotation failed for %s", key)
				}
			}

			// Update keystore with new key
			err = keyStore.Store(service, account, newKey)
			if err != nil {
				t.Fatalf("Failed to update key in keystore: %v", err)
			}
		})
	})

	t.Run("MultipleEnvironments", func(t *testing.T) {
		environments := []string{"development", "staging", "production"}
		
		for _, env := range environments {
			// Each environment gets its own salt and key
			salt, _ := encryptor.GenerateSalt()
			key := encryptor.GenerateKey(password, salt)
			
			// Store key for this environment
			err := keyStore.Store(service, env, key)
			if err != nil {
				t.Fatalf("Failed to store key for %s: %v", env, err)
			}
		}

		// List all environments
		envList, err := keyStore.List(service)
		if err != nil {
			t.Fatalf("Failed to list environments: %v", err)
		}

		// We should have all environments plus the original "production" from above
		if len(envList) < 3 {
			t.Errorf("Expected at least 3 environments, got %d", len(envList))
		}
	})
}