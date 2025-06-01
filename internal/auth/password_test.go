package auth

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/keystore"
)

func TestPasswordManager_DeriveKey(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	pm := NewPasswordManager(ks)
	
	password := "testpassword123"
	salt := []byte("test-salt-32-bytes-long-exactly!")
	
	// Derive key
	key := pm.DeriveKey(password, salt)
	
	// Verify key length
	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
	
	// Verify deterministic - same inputs should produce same key
	key2 := pm.DeriveKey(password, salt)
	if !bytes.Equal(key, key2) {
		t.Error("Key derivation is not deterministic")
	}
	
	// Verify different password produces different key
	key3 := pm.DeriveKey("differentpassword", salt)
	if bytes.Equal(key, key3) {
		t.Error("Different passwords should produce different keys")
	}
	
	// Verify different salt produces different key
	salt2 := []byte("different-salt-32-bytes-long-ok!")
	key4 := pm.DeriveKey(password, salt2)
	if bytes.Equal(key, key4) {
		t.Error("Different salts should produce different keys")
	}
}

func TestPasswordManager_GenerateSalt(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	pm := NewPasswordManager(ks)
	
	salt1, err := pm.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	
	// Verify salt length
	if len(salt1) != saltLen {
		t.Errorf("Expected salt length %d, got %d", saltLen, len(salt1))
	}
	
	// Verify randomness - two salts should be different
	salt2, err := pm.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate second salt: %v", err)
	}
	
	if bytes.Equal(salt1, salt2) {
		t.Error("Generated salts should be different")
	}
}

func TestPasswordManager_VerificationHash(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	pm := NewPasswordManager(ks)
	
	key := []byte("test-encryption-key-32-bytes-ok!")
	
	// Generate verification hash
	hash := pm.generateVerificationHash(key)
	
	// Verify it's a valid hex string
	_, err := hex.DecodeString(hash)
	if err != nil {
		t.Errorf("Verification hash is not valid hex: %v", err)
	}
	
	// Verify deterministic
	hash2 := pm.generateVerificationHash(key)
	if hash != hash2 {
		t.Error("Verification hash should be deterministic")
	}
	
	// Verify verification works
	if !pm.verifyKey(key, hash) {
		t.Error("Key verification failed for correct key")
	}
	
	// Verify wrong key fails
	wrongKey := []byte("wrong-encryption-key-32-bytes-no")
	if pm.verifyKey(wrongKey, hash) {
		t.Error("Key verification should fail for wrong key")
	}
}

func TestPasswordManager_SessionCache(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	pm := NewPasswordManager(ks)
	
	projectID := "test-project"
	key := []byte("test-key-32-bytes-for-caching-ok")
	
	// Cache key
	pm.cacheSessionKey(projectID, key)
	
	// Verify key is in cache
	cacheKey := pm.getCacheKey(projectID)
	entry, ok := pm.sessionCache[cacheKey]
	if !ok {
		t.Fatal("Key not found in cache")
	}
	
	if !bytes.Equal(entry.key, key) {
		t.Error("Cached key does not match")
	}
	
	// Verify expiration is set correctly
	expectedExpiry := time.Now().Add(sessionCacheDuration)
	if entry.expiresAt.Before(time.Now()) {
		t.Error("Cache entry expired immediately")
	}
	if entry.expiresAt.After(expectedExpiry.Add(time.Second)) {
		t.Error("Cache entry expiry too far in future")
	}
	
	// Test clear project cache
	pm.ClearProjectCache(projectID)
	_, ok = pm.sessionCache[cacheKey]
	if ok {
		t.Error("Project cache should be cleared")
	}
	
	// Test clear all cache
	pm.cacheSessionKey("project1", key)
	pm.cacheSessionKey("project2", key)
	pm.ClearSessionCache()
	if len(pm.sessionCache) != 0 {
		t.Error("All cache should be cleared")
	}
}

func TestPasswordManager_ExportImport(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	pm := NewPasswordManager(ks)
	
	projectID := "test-project"
	password := "testpassword123"
	
	// Create a key entry
	salt, _ := pm.GenerateSalt()
	key := pm.DeriveKey(password, salt)
	verificationHash := pm.generateVerificationHash(key)
	
	keyEntry := &keystore.KeyEntry{
		ProjectID:        projectID,
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	}
	
	err := ks.StoreKey(projectID, keyEntry)
	if err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}
	
	// Export key
	exportData, err := pm.ExportKey(projectID, password)
	if err != nil {
		t.Fatalf("Failed to export key: %v", err)
	}
	
	// Verify export format
	if !strings.HasPrefix(exportData, "vaultenv:v1:") {
		t.Error("Export data should start with vaultenv:v1:")
	}
	
	// Test export with wrong password
	_, err = pm.ExportKey(projectID, "wrongpassword")
	if err != ErrInvalidPassword {
		t.Error("Export with wrong password should fail")
	}
	
	// Import to new project
	newProjectID := "imported-project"
	err = pm.ImportKey(newProjectID, exportData, password)
	if err != nil {
		t.Fatalf("Failed to import key: %v", err)
	}
	
	// Verify imported key works
	importedEntry, err := ks.GetKey(newProjectID)
	if err != nil {
		t.Fatalf("Failed to get imported key: %v", err)
	}
	
	if !bytes.Equal(importedEntry.Salt, salt) {
		t.Error("Imported salt does not match")
	}
	
	if importedEntry.VerificationHash != verificationHash {
		t.Error("Imported verification hash does not match")
	}
	
	// Test import with wrong password
	err = pm.ImportKey("another-project", exportData, "wrongpassword")
	if err != ErrInvalidPassword {
		t.Error("Import with wrong password should fail")
	}
	
	// Test import with invalid format
	err = pm.ImportKey("bad-project", "invalid:format", password)
	if err == nil {
		t.Error("Import with invalid format should fail")
	}
}

func TestPasswordManager_GetPasswordFromEnv(t *testing.T) {
	ks, cleanup := setupTestKeystore(t)
	defer cleanup()
	
	pm := NewPasswordManager(ks)
	
	// Test with no env var
	password, ok := pm.GetPasswordFromEnv()
	if ok {
		t.Error("Should return false when env var not set")
	}
	if password != "" {
		t.Error("Password should be empty when env var not set")
	}
	
	// Test with env var set
	testPassword := "env-password-123"
	os.Setenv("VAULTENV_PASSWORD", testPassword)
	defer os.Unsetenv("VAULTENV_PASSWORD")
	
	password, ok = pm.GetPasswordFromEnv()
	if !ok {
		t.Error("Should return true when env var is set")
	}
	if password != testPassword {
		t.Errorf("Expected password %q, got %q", testPassword, password)
	}
}

// Helper function to setup test keystore
func setupTestKeystore(t *testing.T) (*keystore.Keystore, func()) {
	tempDir, err := os.MkdirTemp("", "vaultenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	ks, err := keystore.NewKeystore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create keystore: %v", err)
	}
	
	cleanup := func() {
		ks.Close()
		os.RemoveAll(tempDir)
	}
	
	return ks, cleanup
}