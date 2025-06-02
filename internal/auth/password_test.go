package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/config"
	"github.com/vaultenv/vaultenv-cli/internal/keystore"
)

// MockEnvironmentKeyManager for testing
type MockEnvironmentKeyManager struct {
	hasKey    map[string]bool
	passwords map[string]string
}

func NewMockEnvironmentKeyManager() *MockEnvironmentKeyManager {
	return &MockEnvironmentKeyManager{
		hasKey:    make(map[string]bool),
		passwords: make(map[string]string),
	}
}

func TestPasswordManager_DeriveKey(t *testing.T) {
	// Create a temporary directory for the test
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
	
	password := "test-password"
	salt := make([]byte, 32)
	rand.Read(salt)
	
	// Test key derivation
	key1 := pm.DeriveKey(password, salt)
	if len(key1) != 32 {
		t.Errorf("DeriveKey() length = %d, want 32", len(key1))
	}
	
	// Test deterministic derivation
	key2 := pm.DeriveKey(password, salt)
	if !bytes.Equal(key1, key2) {
		t.Error("DeriveKey() not deterministic")
	}
	
	// Test different password produces different key
	key3 := pm.DeriveKey("different-password", salt)
	if bytes.Equal(key1, key3) {
		t.Error("Different passwords produced same key")
	}
	
	// Test different salt produces different key
	salt2 := make([]byte, 32)
	rand.Read(salt2)
	key4 := pm.DeriveKey(password, salt2)
	if bytes.Equal(key1, key4) {
		t.Error("Different salts produced same key")
	}
}

func TestPasswordManager_GenerateSalt(t *testing.T) {
	// Create a temporary directory for the test
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
	
	// Test salt generation
	salt1, err := pm.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	
	if len(salt1) != saltLen {
		t.Errorf("GenerateSalt() length = %d, want %d", len(salt1), saltLen)
	}
	
	// Test uniqueness
	salt2, err := pm.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	
	if bytes.Equal(salt1, salt2) {
		t.Error("GenerateSalt() produced identical salts")
	}
}

func TestPasswordManager_VerifyPassword(t *testing.T) {
	// Create a temporary directory for the test
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
	
	// Setup test key
	password := "correct-password"
	salt, _ := pm.GenerateSalt()
	key := pm.DeriveKey(password, salt)
	verificationHash := pm.generateVerificationHash(key)
	
	ks.StoreKey("test-project", &keystore.KeyEntry{
		ProjectID:        "test-project",
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	})
	
	// Test correct password
	err = pm.VerifyPassword("test-project", password)
	if err != nil {
		t.Errorf("VerifyPassword() with correct password error = %v", err)
	}
	
	// Test incorrect password
	err = pm.VerifyPassword("test-project", "wrong-password")
	if err != ErrInvalidPassword {
		t.Errorf("VerifyPassword() with wrong password error = %v, want %v", err, ErrInvalidPassword)
	}
	
	// Test non-existent project
	err = pm.VerifyPassword("non-existent", password)
	if err == nil {
		t.Error("VerifyPassword() expected error for non-existent project")
	}
}

func TestPasswordManager_SessionCache(t *testing.T) {
	// Create a temporary directory for the test
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
	
	projectID := "test-project"
	testKey := []byte("test-encryption-key-32-bytes-long")
	
	// Cache a key
	pm.cacheSessionKey(projectID, testKey)
	
	// Verify key is in cache
	cacheKey := pm.getCacheKey(projectID)
	entry, ok := pm.sessionCache[cacheKey]
	if !ok {
		t.Error("Key not found in cache")
	}
	
	if !bytes.Equal(entry.key, testKey) {
		t.Error("Cached key doesn't match")
	}
	
	// Test cache expiration
	if entry.expiresAt.Before(time.Now()) {
		t.Error("Cache entry already expired")
	}
	
	// Clear project cache
	pm.ClearProjectCache(projectID)
	_, ok = pm.sessionCache[cacheKey]
	if ok {
		t.Error("Key still in cache after ClearProjectCache")
	}
	
	// Test ClearSessionCache
	pm.cacheSessionKey(projectID, testKey)
	pm.ClearSessionCache()
	if len(pm.sessionCache) != 0 {
		t.Error("Cache not empty after ClearSessionCache")
	}
}

func TestPasswordManager_ExportImportKey(t *testing.T) {
	// Create a temporary directory for the test
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
	
	// Setup test key
	password := "test-password"
	salt, _ := pm.GenerateSalt()
	key := pm.DeriveKey(password, salt)
	verificationHash := pm.generateVerificationHash(key)
	
	ks.StoreKey("test-project", &keystore.KeyEntry{
		ProjectID:        "test-project",
		Salt:             salt,
		VerificationHash: verificationHash,
		CreatedAt:        time.Now(),
	})
	
	// Export key
	exportData, err := pm.ExportKey("test-project", password)
	if err != nil {
		t.Fatalf("ExportKey() error = %v", err)
	}
	
	// Verify export format
	if !strings.HasPrefix(exportData, "vaultenv:v1:") {
		t.Errorf("ExportKey() invalid format = %v", exportData)
	}
	
	// Import to new project
	err = pm.ImportKey("new-project", exportData, password)
	if err != nil {
		t.Fatalf("ImportKey() error = %v", err)
	}
	
	// Verify imported key works
	err = pm.VerifyPassword("new-project", password)
	if err != nil {
		t.Errorf("VerifyPassword() after import error = %v", err)
	}
	
	// Test import with wrong password
	err = pm.ImportKey("another-project", exportData, "wrong-password")
	if err != ErrInvalidPassword {
		t.Errorf("ImportKey() with wrong password error = %v, want %v", err, ErrInvalidPassword)
	}
	
	// Test invalid export format
	err = pm.ImportKey("bad-project", "invalid:format", password)
	if err == nil || !strings.Contains(err.Error(), "invalid export format") {
		t.Errorf("ImportKey() with invalid format error = %v", err)
	}
}

func TestPasswordManager_GetPasswordFromEnv(t *testing.T) {
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
	
	// Test no environment variable
	password, exists := pm.GetPasswordFromEnv()
	if exists {
		t.Error("GetPasswordFromEnv() should return false when env var not set")
	}
	
	// Set environment variable
	testPassword := "env-password"
	os.Setenv("VAULTENV_PASSWORD", testPassword)
	defer os.Unsetenv("VAULTENV_PASSWORD")
	
	password, exists = pm.GetPasswordFromEnv()
	if !exists {
		t.Error("GetPasswordFromEnv() should return true when env var is set")
	}
	if password != testPassword {
		t.Errorf("GetPasswordFromEnv() = %v, want %v", password, testPassword)
	}
}

func TestPasswordValidation_Helpers(t *testing.T) {
	tests := []struct {
		name     string
		password string
		checks   map[string]bool
	}{
		{
			name:     "all_lowercase",
			password: "abcdefgh",
			checks: map[string]bool{
				"upper":   false,
				"lower":   true,
				"number":  false,
				"special": false,
			},
		},
		{
			name:     "all_uppercase",
			password: "ABCDEFGH",
			checks: map[string]bool{
				"upper":   true,
				"lower":   false,
				"number":  false,
				"special": false,
			},
		},
		{
			name:     "with_numbers",
			password: "abc123DEF",
			checks: map[string]bool{
				"upper":   true,
				"lower":   true,
				"number":  true,
				"special": false,
			},
		},
		{
			name:     "with_special",
			password: "Abc123!@#",
			checks: map[string]bool{
				"upper":   true,
				"lower":   true,
				"number":  true,
				"special": true,
			},
		},
		{
			name:     "empty",
			password: "",
			checks: map[string]bool{
				"upper":   false,
				"lower":   false,
				"number":  false,
				"special": false,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsUppercase(tt.password); got != tt.checks["upper"] {
				t.Errorf("containsUppercase(%q) = %v, want %v", tt.password, got, tt.checks["upper"])
			}
			if got := containsLowercase(tt.password); got != tt.checks["lower"] {
				t.Errorf("containsLowercase(%q) = %v, want %v", tt.password, got, tt.checks["lower"])
			}
			if got := containsNumber(tt.password); got != tt.checks["number"] {
				t.Errorf("containsNumber(%q) = %v, want %v", tt.password, got, tt.checks["number"])
			}
			if got := containsSpecial(tt.password); got != tt.checks["special"] {
				t.Errorf("containsSpecial(%q) = %v, want %v", tt.password, got, tt.checks["special"])
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	tests := []struct {
		password string
		isCommon bool
	}{
		{"password", true},
		{"Password1", true},
		{"123456", true},
		{"qwerty", true},
		{"welcome123", true},
		{"VeryUniquePassword123!", false},
		{"MyS3cur3P@ssw0rd", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			if got := isCommonPassword(tt.password); got != tt.isCommon {
				t.Errorf("isCommonPassword(%q) = %v, want %v", tt.password, got, tt.isCommon)
			}
		})
	}
}

func TestPasswordManager_ValidatePasswordPolicy(t *testing.T) {
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
	
	tests := []struct {
		name     string
		password string
		policy   config.PassPolicy
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "min_length_fail",
			password: "short",
			policy: config.PassPolicy{
				MinLength: 10,
			},
			wantErr: true,
			errMsg:  "at least 10 characters",
		},
		{
			name:     "require_upper_fail",
			password: "alllowercase123",
			policy: config.PassPolicy{
				MinLength:    8,
				RequireUpper: true,
			},
			wantErr: true,
			errMsg:  "uppercase letter",
		},
		{
			name:     "require_lower_fail",
			password: "ALLUPPERCASE123",
			policy: config.PassPolicy{
				MinLength:    8,
				RequireLower: true,
			},
			wantErr: true,
			errMsg:  "lowercase letter",
		},
		{
			name:     "require_number_fail",
			password: "NoNumbersHere",
			policy: config.PassPolicy{
				MinLength:      8,
				RequireNumbers: true,
			},
			wantErr: true,
			errMsg:  "one number",
		},
		{
			name:     "require_special_fail",
			password: "NoSpecialChars123",
			policy: config.PassPolicy{
				MinLength:      8,
				RequireSpecial: true,
			},
			wantErr: true,
			errMsg:  "special character",
		},
		{
			name:     "common_password_fail",
			password: "password123",
			policy: config.PassPolicy{
				MinLength:     8,
				PreventCommon: true,
			},
			wantErr: true,
			errMsg:  "too common",
		},
		{
			name:     "all_requirements_pass",
			password: "MyS3cur3P@ssw0rd",
			policy: config.PassPolicy{
				MinLength:      12,
				RequireUpper:   true,
				RequireLower:   true,
				RequireNumbers: true,
				RequireSpecial: true,
				PreventCommon:  true,
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.validatePasswordPolicy(tt.password, tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePasswordPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validatePasswordPolicy() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func TestPasswordManager_EnvironmentPassword(t *testing.T) {
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
	
	// Test environment-specific password from env var
	envPassword := "env-specific-password"
	os.Setenv("VAULTENV_PASSWORD_DEV", envPassword)
	defer os.Unsetenv("VAULTENV_PASSWORD_DEV")
	
	password, err := pm.PromptEnvironmentPassword("dev", "Enter password: ")
	if err != nil {
		t.Fatalf("PromptEnvironmentPassword() error = %v", err)
	}
	if password != envPassword {
		t.Errorf("PromptEnvironmentPassword() = %v, want %v", password, envPassword)
	}
	
	// Test fallback to generic env var
	genericPassword := "generic-password"
	os.Setenv("VAULTENV_PASSWORD", genericPassword)
	defer os.Unsetenv("VAULTENV_PASSWORD")
	
	password, err = pm.PromptEnvironmentPassword("prod", "Enter password: ")
	if err != nil {
		t.Fatalf("PromptEnvironmentPassword() error = %v", err)
	}
	if password != genericPassword {
		t.Errorf("PromptEnvironmentPassword() = %v, want %v", password, genericPassword)
	}
}

func TestPasswordManager_CacheKeys(t *testing.T) {
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
	
	// Test project key cache
	projectKey := pm.getCacheKey("test-project")
	if projectKey != "project:test-project" {
		t.Errorf("getCacheKey() = %v, want 'project:test-project'", projectKey)
	}
	
	// Test environment key cache
	envKey := pm.getEnvironmentCacheKey("test-project", "dev")
	if envKey != "project:test-project:env:dev" {
		t.Errorf("getEnvironmentCacheKey() = %v, want 'project:test-project:env:dev'", envKey)
	}
	
	// Test caching and retrieval
	testKey := []byte("test-key-32-bytes-for-encryption")
	pm.cacheEnvironmentKey("test-project", "dev", testKey)
	
	entry, exists := pm.sessionCache[envKey]
	if !exists {
		t.Error("Environment key not found in cache")
	}
	if !bytes.Equal(entry.key, testKey) {
		t.Error("Cached environment key doesn't match")
	}
	
	// Test clear environment cache
	pm.ClearEnvironmentCache("dev")
	_, exists = pm.sessionCache[envKey]
	if exists {
		t.Error("Environment key still in cache after clear")
	}
}

func TestPasswordManager_ImportKeyValidation(t *testing.T) {
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
	
	tests := []struct {
		name       string
		exportData string
		password   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "invalid_prefix",
			exportData: "invalid:v1:salt:hash",
			password:   "password",
			wantErr:    true,
			errMsg:     "invalid export format",
		},
		{
			name:       "invalid_version",
			exportData: "vaultenv:v2:salt:hash",
			password:   "password",
			wantErr:    true,
			errMsg:     "invalid export format",
		},
		{
			name:       "too_few_parts",
			exportData: "vaultenv:v1:salt",
			password:   "password",
			wantErr:    true,
			errMsg:     "invalid export format",
		},
		{
			name:       "invalid_base64_salt",
			exportData: "vaultenv:v1:not-base64!:hash",
			password:   "password",
			wantErr:    true,
			errMsg:     "invalid salt format",
		},
		{
			name:       "valid_format_wrong_password",
			exportData: fmt.Sprintf("vaultenv:v1:%s:wronghash", base64.StdEncoding.EncodeToString(make([]byte, 32))),
			password:   "password",
			wantErr:    true,
			errMsg:     "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.ImportKey("test-import", tt.exportData, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImportKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ImportKey() error = %v, want error containing %v", err, tt.errMsg)
			}
		})
	}
}

func BenchmarkPasswordManager_DeriveKey(b *testing.B) {
	tmpDir := b.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()
	
	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)
	
	password := "benchmark-password"
	salt := make([]byte, 32)
	rand.Read(salt)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.DeriveKey(password, salt)
	}
}

func BenchmarkPasswordManager_GenerateVerificationHash(b *testing.B) {
	tmpDir := b.TempDir()
	ks, err := keystore.NewKeystore(tmpDir)
	if err != nil {
		b.Fatalf("Failed to create keystore: %v", err)
	}
	defer ks.Close()
	
	cfg := &config.Config{
		Project: config.ProjectConfig{ID: "test-project"},
	}
	pm := NewPasswordManager(ks, cfg)
	
	key := make([]byte, 32)
	rand.Read(key)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.generateVerificationHash(key)
	}
}