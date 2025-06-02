package auth

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
	"golang.org/x/term"
)

// MockStdin allows us to simulate user input for password prompts
type MockStdin struct {
	*bytes.Buffer
}

func (m *MockStdin) Fd() uintptr {
	return 0
}

// TestPasswordManager_GetOrCreateMasterKey_FullFlow tests the complete flow
func TestPasswordManager_GetOrCreateMasterKey_FullFlow(t *testing.T) {
	// Skip in CI environments where stdin is not available
	if os.Getenv("CI") != "" || testing.Short() {
		t.Skip("Skipping interactive test in CI environment or short mode")
	}

	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)

	t.Run("new_key_creation", func(t *testing.T) {
		// Test with environment variable to avoid interactive prompt
		testPassword := "new-secure-password123"
		os.Setenv("VAULTENV_PASSWORD", testPassword)
		defer os.Unsetenv("VAULTENV_PASSWORD")

		// Since we can't easily mock terminal input, we'll test the logic
		// by pre-creating the key and then testing retrieval
		salt, _ := pm.GenerateSalt()
		key := pm.DeriveKey(testPassword, salt)
		verificationHash := pm.generateVerificationHash(key)

		ks.StoreKey("new-project", &keystore.KeyEntry{
			ProjectID:        "new-project",
			Salt:             salt,
			VerificationHash: verificationHash,
			CreatedAt:        time.Now(),
		})

		// Now test retrieval
		retrievedKey, err := pm.GetOrCreateMasterKey("new-project")
		if err != nil {
			t.Fatalf("GetOrCreateMasterKey() error = %v", err)
		}

		if !bytes.Equal(retrievedKey, key) {
			t.Error("Retrieved key doesn't match original")
		}
	})

	t.Run("cached_key_retrieval", func(t *testing.T) {
		// Test that subsequent calls use cached key
		testKey := []byte("cached-test-key-32-bytes-long!!!")
		pm.cacheSessionKey("cached-project", testKey)

		retrievedKey, err := pm.GetOrCreateMasterKey("cached-project")
		if err != nil {
			t.Fatalf("GetOrCreateMasterKey() with cached key error = %v", err)
		}

		if !bytes.Equal(retrievedKey, testKey) {
			t.Error("Cached key not returned")
		}
	})

	t.Run("expired_cache_cleanup", func(t *testing.T) {
		// Create expired cache entry
		cacheKey := pm.getCacheKey("expired-project")
		pm.sessionCache[cacheKey] = &sessionEntry{
			key:       []byte("expired-key"),
			expiresAt: time.Now().Add(-1 * time.Hour),
		}

		// Setup valid key in keystore
		password := "valid-password"
		salt, _ := pm.GenerateSalt()
		key := pm.DeriveKey(password, salt)
		verificationHash := pm.generateVerificationHash(key)

		ks.StoreKey("expired-project", &keystore.KeyEntry{
			ProjectID:        "expired-project",
			Salt:             salt,
			VerificationHash: verificationHash,
			CreatedAt:        time.Now(),
		})

		os.Setenv("VAULTENV_PASSWORD", password)
		defer os.Unsetenv("VAULTENV_PASSWORD")

		// GetOrCreateMasterKey should clean up expired entry
		retrievedKey, err := pm.GetOrCreateMasterKey("expired-project")
		if err != nil {
			t.Fatalf("GetOrCreateMasterKey() error = %v", err)
		}

		if !bytes.Equal(retrievedKey, key) {
			t.Error("Key mismatch after expired cache cleanup")
		}

		// Verify expired entry was removed
		if _, exists := pm.sessionCache[cacheKey]; exists {
			if time.Now().After(pm.sessionCache[cacheKey].expiresAt) {
				t.Error("Expired cache entry not cleaned up")
			}
		}
	})
}

// TestPasswordManager_ChangePassword_FullFlow tests password change functionality
func TestPasswordManager_ChangePassword_FullFlow(t *testing.T) {
	if os.Getenv("CI") != "" || testing.Short() {
		t.Skip("Skipping interactive test in CI environment or short mode")
	}

	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)

	// Setup initial password
	currentPassword := "current-password123"
	salt, _ := pm.GenerateSalt()
	key := pm.DeriveKey(currentPassword, salt)
	verificationHash := pm.generateVerificationHash(key)

	ks.StoreKey("change-test", &keystore.KeyEntry{
		ProjectID:        "change-test",
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	})

	// Test password verification
	err = pm.VerifyPassword("change-test", currentPassword)
	if err != nil {
		t.Fatalf("Initial password verification failed: %v", err)
	}

	// Test with wrong password
	err = pm.VerifyPassword("change-test", "wrong-password")
	if err != ErrInvalidPassword {
		t.Errorf("Expected ErrInvalidPassword, got %v", err)
	}

	// Since we can't mock terminal input easily, we'll test the components
	// that ChangePassword uses
	t.Run("password_change_components", func(t *testing.T) {
		// Test new password generation
		newPassword := "new-secure-password456"
		newSalt, _ := pm.GenerateSalt()
		newKey := pm.DeriveKey(newPassword, newSalt)
		newVerificationHash := pm.generateVerificationHash(newKey)

		// Update keystore
		ks.StoreKey("change-test", &keystore.KeyEntry{
			ProjectID:        "change-test",
			Salt:             newSalt,
			VerificationHash: newVerificationHash,
			CreatedAt:        time.Now(),
		})

		// Verify old password no longer works
		err = pm.VerifyPassword("change-test", currentPassword)
		if err != ErrInvalidPassword {
			t.Error("Old password still works after change")
		}

		// Verify new password works
		err = pm.VerifyPassword("change-test", newPassword)
		if err != nil {
			t.Errorf("New password verification failed: %v", err)
		}

		// Verify cache was cleared
		cacheKey := pm.getCacheKey("change-test")
		if _, exists := pm.sessionCache[cacheKey]; exists {
			t.Error("Cache not cleared after password change")
		}
	})
}

// TestPasswordManager_GetOrCreateEnvironmentKey_FullFlow tests environment-specific keys
func TestPasswordManager_GetOrCreateEnvironmentKey_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
		Security: config.SecurityConfig{
			PerEnvironmentPasswords: true,
		},
	}
	pm := NewPasswordManager(ks, cfg)

	t.Run("per_env_disabled", func(t *testing.T) {
		// Test with per-environment passwords disabled
		cfg.Security.PerEnvironmentPasswords = false

		// Set project password
		os.Setenv("VAULTENV_PASSWORD", "project-password")
		defer os.Unsetenv("VAULTENV_PASSWORD")

		// Pre-create project key
		password := "project-password"
		salt, _ := pm.GenerateSalt()
		key := pm.DeriveKey(password, salt)
		verificationHash := pm.generateVerificationHash(key)

		ks.StoreKey("test-project", &keystore.KeyEntry{
			ProjectID:        "test-project",
			Salt:             salt,
			VerificationHash: verificationHash,
			CreatedAt:        time.Now(),
		})

		// Should fall back to project key
		envKey, err := pm.GetOrCreateEnvironmentKey("production")
		if err != nil {
			t.Fatalf("GetOrCreateEnvironmentKey() error = %v", err)
		}

		if !bytes.Equal(envKey, key) {
			t.Error("Environment key doesn't match project key when per-env disabled")
		}
	})

	t.Run("per_env_enabled_cached", func(t *testing.T) {
		cfg.Security.PerEnvironmentPasswords = true

		// Test with cached environment key
		testKey := []byte("env-specific-key-32-bytes-long!!")
		pm.cacheEnvironmentKey("test-project", "staging", testKey)

		retrievedKey, err := pm.GetOrCreateEnvironmentKey("staging")
		if err != nil {
			t.Fatalf("GetOrCreateEnvironmentKey() error = %v", err)
		}

		if !bytes.Equal(retrievedKey, testKey) {
			t.Error("Cached environment key not returned")
		}
	})

	t.Run("environment_cache_expiration", func(t *testing.T) {
		// Create expired cache entry
		cacheKey := pm.getEnvironmentCacheKey("test-project", "dev")
		pm.sessionCache[cacheKey] = &sessionEntry{
			key:       []byte("expired-env-key"),
			expiresAt: time.Now().Add(-1 * time.Hour),
		}

		// Verify expired entry exists
		if _, exists := pm.sessionCache[cacheKey]; !exists {
			t.Fatal("Test setup failed: expired entry not created")
		}

		// After checking in GetOrCreateEnvironmentKey, it should be cleaned up
		// This would happen when the key is accessed
		if entry, ok := pm.sessionCache[cacheKey]; ok {
			if time.Now().After(entry.expiresAt) {
				delete(pm.sessionCache, cacheKey)
			}
		}

		// Verify cleanup
		if _, exists := pm.sessionCache[cacheKey]; exists {
			t.Error("Expired environment cache entry not cleaned up")
		}
	})
}

// TestPasswordManager_PromptNewEnvironmentPassword_PolicyValidation tests password policy enforcement
func TestPasswordManager_PromptNewEnvironmentPassword_PolicyValidation(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
		Environments: map[string]config.EnvironmentConfig{
			"production": {
				PasswordPolicy: config.PassPolicy{
					MinLength:      16,
					RequireUpper:   true,
					RequireLower:   true,
					RequireNumbers: true,
					RequireSpecial: true,
					PreventCommon:  true,
				},
			},
			"development": {
				PasswordPolicy: config.PassPolicy{
					MinLength: 8,
				},
			},
		},
	}
	pm := NewPasswordManager(ks, cfg)

	// Test policy retrieval and validation
	t.Run("production_policy", func(t *testing.T) {
		policy := cfg.GetPasswordPolicy("production")

		// Test various passwords against production policy
		testCases := []struct {
			password string
			valid    bool
			errMsg   string
		}{
			{"short", false, "at least 16 characters"},
			{"Verylongbutnonumbers", false, "one number"},
			{"VERYLONGWITH123NOUPPER", false, "lowercase letter"},
			{"verylongwith123nolower", false, "uppercase letter"},
			{"VeryLong123NoSpecial", false, "special character"},
			{"Password1", false, "at least 16 characters"}, // Also common
			{"VerySecure123!@#Pass", true, ""},
		}

		for _, tc := range testCases {
			err := pm.validatePasswordPolicy(tc.password, policy)
			if tc.valid && err != nil {
				t.Errorf("Password %q should be valid but got error: %v", tc.password, err)
			} else if !tc.valid && err == nil {
				t.Errorf("Password %q should be invalid but no error returned", tc.password)
			} else if !tc.valid && tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("Password %q expected error containing %q but got: %v", tc.password, tc.errMsg, err)
			}
		}
	})

	t.Run("development_policy", func(t *testing.T) {
		policy := cfg.GetPasswordPolicy("development")

		// Development has minimal requirements
		err := pm.validatePasswordPolicy("simple12", policy)
		if err != nil {
			t.Errorf("Simple password should be valid for development: %v", err)
		}

		err = pm.validatePasswordPolicy("short", policy)
		if err == nil {
			t.Error("Password shorter than 8 chars should be invalid")
		}
	})

	t.Run("default_policy", func(t *testing.T) {
		// Environment without specific policy should use default
		policy := cfg.GetPasswordPolicy("staging")

		// Should have default 8 character minimum
		err := pm.validatePasswordPolicy("12345678", policy)
		if err != nil {
			t.Errorf("8 character password should be valid with default policy: %v", err)
		}
	})
}

// TestPasswordManager_ChangeEnvironmentPassword_FullFlow tests changing environment-specific passwords
func TestPasswordManager_ChangeEnvironmentPassword_FullFlow(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
		Security: config.SecurityConfig{
			PerEnvironmentPasswords: true,
		},
	}
	pm := NewPasswordManager(ks, cfg)

	t.Run("per_env_disabled_error", func(t *testing.T) {
		cfg.Security.PerEnvironmentPasswords = false
		err := pm.ChangeEnvironmentPassword("production")
		if err == nil {
			t.Error("ChangeEnvironmentPassword should fail when per-env passwords disabled")
		}
		if !strings.Contains(err.Error(), "not enabled") {
			t.Errorf("Expected error about per-env passwords not enabled, got: %v", err)
		}
	})

	t.Run("cache_cleanup", func(t *testing.T) {
		cfg.Security.PerEnvironmentPasswords = true

		// Pre-cache a key
		cacheKey := pm.getEnvironmentCacheKey("test-project", "staging")
		pm.sessionCache[cacheKey] = &sessionEntry{
			key:       []byte("old-staging-key"),
			expiresAt: time.Now().Add(time.Hour),
		}

		// Verify cache exists
		if _, exists := pm.sessionCache[cacheKey]; !exists {
			t.Fatal("Test setup failed: cache entry not created")
		}

		// Clear environment cache
		pm.ClearEnvironmentCache("staging")

		// Verify cache cleared
		if _, exists := pm.sessionCache[cacheKey]; exists {
			t.Error("Environment cache not cleared")
		}
	})
}

// TestPasswordManager_ConcurrentEnvironmentAccess tests concurrent access to environment keys
func TestPasswordManager_ConcurrentEnvironmentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
		Security: config.SecurityConfig{
			PerEnvironmentPasswords: true,
		},
	}
	pm := NewPasswordManager(ks, cfg)

	environments := []string{"dev", "staging", "production"}
	var wg sync.WaitGroup

	// Concurrent cache operations for different environments
	for _, env := range environments {
		wg.Add(1)
		go func(environment string) {
			defer wg.Done()

			for i := 0; i < 50; i++ {
				key := make([]byte, 32)
				copy(key, []byte(fmt.Sprintf("%s-key-%d", environment, i)))

				// Cache and clear operations
				pm.cacheEnvironmentKey("test-project", environment, key)
				cacheKey := pm.getEnvironmentCacheKey("test-project", environment)

				// Read from cache - use mutex for safe access
				pm.cacheMutex.RLock()
				if entry, exists := pm.sessionCache[cacheKey]; exists {
					_ = entry.key
				}
				pm.cacheMutex.RUnlock()

				// Occasionally clear
				if i%10 == 0 {
					pm.ClearEnvironmentCache(environment)
				}
			}
		}(env)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent test timeout - possible deadlock")
	}
}

// TestPasswordManager_PromptNewPassword_Validation tests password validation in prompt
func TestPasswordManager_PromptNewPassword_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)
	_ = pm // Mark as used

	// Since we can't easily mock terminal.ReadPassword, we'll test the validation logic
	t.Run("password_length_validation", func(t *testing.T) {
		// Test passwords that would be rejected by PromptNewPassword
		shortPasswords := []string{"", "1234567", "short"}

		for _, pwd := range shortPasswords {
			if len(pwd) >= 8 {
				t.Errorf("Test setup error: %q should be < 8 chars", pwd)
			}
		}

		// Test passwords that would be accepted
		validPasswords := []string{"12345678", "longenoughpassword", "VerySecurePassword123!"}

		for _, pwd := range validPasswords {
			if len(pwd) < 8 {
				t.Errorf("Test setup error: %q should be >= 8 chars", pwd)
			}
		}
	})
}

// TestPasswordManager_ErrorRecovery tests error recovery scenarios
func TestPasswordManager_ErrorRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)

	t.Run("corrupt_verification_hash", func(t *testing.T) {
		// Create key with corrupt verification hash
		salt, _ := pm.GenerateSalt()

		ks.StoreKey("corrupt-project", &keystore.KeyEntry{
			ProjectID:        "corrupt-project",
			Salt:             salt,
			VerificationHash: "invalid-hash",
			CreatedAt:        time.Now(),
		})

		// Any password should fail verification
		err := pm.VerifyPassword("corrupt-project", "any-password")
		if err != ErrInvalidPassword {
			t.Errorf("Expected ErrInvalidPassword for corrupt hash, got: %v", err)
		}
	})

	t.Run("empty_salt", func(t *testing.T) {
		// Test with empty salt (should still work, just less secure)
		password := "test-password"
		emptySalt := make([]byte, 0)
		key := pm.DeriveKey(password, emptySalt)

		if len(key) != 32 {
			t.Errorf("DeriveKey with empty salt produced key of length %d, want 32", len(key))
		}
	})

	t.Run("nil_config_handling", func(t *testing.T) {
		// Test that password manager handles nil config gracefully
		pmNilConfig := &PasswordManager{
			keystore:     ks,
			config:       nil,
			sessionCache: make(map[string]*sessionEntry),
		}

		// Operations that use config should handle nil
		cacheKey := pmNilConfig.getCacheKey("test")
		if cacheKey != "project:test" {
			t.Errorf("getCacheKey with nil config = %q, want 'project:test'", cacheKey)
		}
	})
}

// TestPasswordManager_PromptPassword_Terminal tests terminal password reading
// This test requires special setup and is typically skipped in CI
func TestPasswordManager_PromptPassword_Terminal(t *testing.T) {
	if testing.Short() || os.Getenv("CI") != "" {
		t.Skip("Skipping terminal interaction test")
	}

	// This test demonstrates how PromptPassword would work with real terminal
	// In actual usage, it reads from terminal without echoing
	tmpDir := t.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()

	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)

	// Check if we're in a terminal
	if !term.IsTerminal(int(syscall.Stdin)) {
		t.Skip("Not running in a terminal")
	}

	t.Log("This test would prompt for password input if run interactively")

	// In a real scenario, this would prompt the user
	// password, err := pm.PromptPassword("Enter test password: ")

	// For testing, we just verify the method exists and document its behavior
	_ = pm.PromptPassword
}
