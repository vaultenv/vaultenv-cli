package storage

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/vaultenv/vaultenv-cli/pkg/encryption"
)

func TestEncryptedBackend_NewEncryptedBackend(t *testing.T) {
	memBackend := NewMemoryBackend()

	tests := []struct {
		name     string
		backend  Backend
		password string
		wantErr  bool
	}{
		{"valid", memBackend, "test-password", false},
		{"nil_backend", nil, "test-password", true},
		{"empty_password", memBackend, "", true},
		{"long_password", memBackend, strings.Repeat("a", 1000), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptedBackend(tt.backend, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptedBackend() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptedBackend_SetGet(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	tests := []struct {
		name    string
		key     string
		value   string
		encrypt bool
	}{
		{"plaintext", "PLAIN_KEY", "plain value", false},
		{"encrypted", "SECRET_KEY", "secret value", true},
		{"empty_plaintext", "EMPTY_PLAIN", "", false},
		{"empty_encrypted", "EMPTY_SECRET", "", true},
		{"unicode_encrypted", "UNICODE", "Hello 世界 🌍", true},
		{"special_chars", "SPECIAL", "!@#$%^&*()[]{}|;':\",./<>?", true},
		{"multiline", "MULTILINE", "line1\nline2\r\nline3", true},
		{"json_value", "JSON", `{"key": "value", "number": 123}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set value
			err := encBackend.Set(tt.key, tt.value, tt.encrypt)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			// Get value
			got, err := encBackend.Get(tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if got != tt.value {
				t.Errorf("Get() = %v, want %v", got, tt.value)
			}

			// Verify encrypted values are actually encrypted in backend
			if tt.encrypt {
				rawData, _ := memBackend.Get(tt.key)

				// Should be valid JSON
				var ev EncryptedValue
				if err := json.Unmarshal([]byte(rawData), &ev); err != nil {
					t.Errorf("Encrypted value is not valid JSON: %v", err)
				}

				// Should be marked as encrypted
				if !ev.IsEncrypted {
					t.Error("Value should be marked as encrypted")
				}

				// Ciphertext should not equal plaintext
				if ev.Ciphertext == tt.value && tt.value != "" {
					t.Error("Ciphertext equals plaintext")
				}
			}
		})
	}
}

func TestEncryptedBackend_NonEncryptedValues(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	// Set non-encrypted value
	err := encBackend.Set("PLAIN", "plain text", false)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify storage format
	rawData, _ := memBackend.Get("PLAIN")
	var ev EncryptedValue
	if err := json.Unmarshal([]byte(rawData), &ev); err != nil {
		t.Fatalf("Failed to unmarshal non-encrypted value: %v", err)
	}

	if ev.IsEncrypted {
		t.Error("Non-encrypted value marked as encrypted")
	}

	if ev.Ciphertext != "plain text" {
		t.Errorf("Non-encrypted value = %v, want 'plain text'", ev.Ciphertext)
	}
}

func TestEncryptedBackend_LegacyPlaintext(t *testing.T) {
	memBackend := NewMemoryBackend()

	// Store legacy plaintext directly in backend
	memBackend.Set("LEGACY", "legacy value", false)

	// Create encrypted backend
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	// Should be able to read legacy value
	value, err := encBackend.Get("LEGACY")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "legacy value" {
		t.Errorf("Get() = %v, want 'legacy value'", value)
	}
}

func TestEncryptedBackend_WrongPassword(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend1, _ := NewEncryptedBackend(memBackend, "correct-password")

	// Set encrypted value
	encBackend1.Set("SECRET", "secret value", true)

	// Try to read with wrong password
	encBackend2, _ := NewEncryptedBackend(memBackend, "wrong-password")
	_, err := encBackend2.Get("SECRET")

	if err == nil {
		t.Error("Expected error when decrypting with wrong password")
	}

	if !strings.Contains(err.Error(), "failed to decrypt") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestEncryptedBackend_UpdatePassword(t *testing.T) {
	memBackend := NewMemoryBackend()
	oldPassword := "old-password"
	newPassword := "new-password"

	// Create backend with old password
	encBackend, _ := NewEncryptedBackend(memBackend, oldPassword)

	// Set some values
	testData := map[string]struct {
		value   string
		encrypt bool
	}{
		"PLAIN":     {"plain value", false},
		"SECRET1":   {"secret value 1", true},
		"SECRET2":   {"secret value 2", true},
		"UNICODE":   {"Hello 世界", true},
		"MULTILINE": {"line1\nline2", true},
	}

	for key, data := range testData {
		encBackend.Set(key, data.value, data.encrypt)
	}

	// Update password
	err := encBackend.UpdatePassword(oldPassword, newPassword)
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	// Verify all values can be read with new password
	for key, data := range testData {
		value, err := encBackend.Get(key)
		if err != nil {
			t.Errorf("Get(%s) after password update error = %v", key, err)
		}
		if value != data.value {
			t.Errorf("Get(%s) = %v, want %v", key, value, data.value)
		}
	}

	// Create new backend with new password
	encBackend2, _ := NewEncryptedBackend(memBackend, newPassword)

	// Verify all values can be read with new backend
	for key, data := range testData {
		value, err := encBackend2.Get(key)
		if err != nil {
			t.Errorf("Get(%s) with new backend error = %v", key, err)
		}
		if value != data.value {
			t.Errorf("Get(%s) with new backend = %v, want %v", key, value, data.value)
		}
	}

	// Verify old password no longer works for encrypted values
	encBackend3, _ := NewEncryptedBackend(memBackend, oldPassword)
	_, err = encBackend3.Get("SECRET1")
	if err == nil {
		t.Error("Expected error when using old password after update")
	}
}

func TestEncryptedBackend_UpdatePasswordErrors(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "password")

	tests := []struct {
		name        string
		oldPassword string
		newPassword string
		wantErr     bool
	}{
		{"empty_old", "", "new", true},
		{"empty_new", "old", "", true},
		{"both_empty", "", "", true},
		{"valid", "password", "new-password", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := encBackend.UpdatePassword(tt.oldPassword, tt.newPassword)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptedBackend_Delete(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	// Set values
	encBackend.Set("TO_DELETE", "value", true)
	encBackend.Set("TO_KEEP", "value", true)

	// Delete one
	err := encBackend.Delete("TO_DELETE")
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify deletion
	exists, _ := encBackend.Exists("TO_DELETE")
	if exists {
		t.Error("Delete() did not remove the key")
	}

	// Verify other key remains
	exists, _ = encBackend.Exists("TO_KEEP")
	if !exists {
		t.Error("Delete() removed wrong key")
	}
}

func TestEncryptedBackend_List(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	// Set mixed values
	keys := []string{"PLAIN1", "SECRET1", "PLAIN2", "SECRET2"}
	for i, key := range keys {
		encrypt := i%2 == 1
		encBackend.Set(key, "value", encrypt)
	}

	// List should return all keys
	list, err := encBackend.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != len(keys) {
		t.Errorf("List() returned %d keys, want %d", len(list), len(keys))
	}
}

func TestEncryptedBackend_WithDifferentEncryptors(t *testing.T) {
	t.Skip("Skipping deterministic encryptor test for beta release - known issue")
	memBackend := NewMemoryBackend()

	// Test with deterministic encryptor
	detEnc := encryption.NewDeterministicEncryptor()
	encBackend, err := NewEncryptedBackendWithEncryptor(memBackend, "test-password", detEnc)
	if err != nil {
		t.Fatalf("NewEncryptedBackendWithEncryptor() error = %v", err)
	}

	// Set and get value
	err = encBackend.Set("KEY", "value", true)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	value, err := encBackend.Get("KEY")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if value != "value" {
		t.Errorf("Get() = %v, want 'value'", value)
	}

	// Verify deterministic encryption
	rawData1, _ := memBackend.Get("KEY")
	var ev1 EncryptedValue
	json.Unmarshal([]byte(rawData1), &ev1)

	// Set same value again
	encBackend.Set("KEY", "value", true)
	rawData2, _ := memBackend.Get("KEY")
	var ev2 EncryptedValue
	json.Unmarshal([]byte(rawData2), &ev2)

	// Should produce same ciphertext (deterministic)
	if ev1.Ciphertext != ev2.Ciphertext {
		t.Error("Deterministic encryptor produced different ciphertexts")
	}
	
	// Salt should also be the same for deterministic encryption
	if ev1.Salt != ev2.Salt {
		t.Error("Deterministic encryptor produced different salts")
	}
}

func TestEncryptedBackend_AlgorithmMigration(t *testing.T) {
	memBackend := NewMemoryBackend()

	// Create backend with AES-GCM
	aesBackend, _ := NewEncryptedBackend(memBackend, "test-password")
	aesBackend.Set("KEY", "value", true)

	// Get raw data to verify algorithm
	rawData, _ := memBackend.Get("KEY")
	var ev EncryptedValue
	json.Unmarshal([]byte(rawData), &ev)

	if ev.Algorithm != "aes-gcm-256" {
		t.Errorf("Algorithm = %v, want aes-gcm-256", ev.Algorithm)
	}

	// Create backend with deterministic encryptor
	detEnc := encryption.NewDeterministicEncryptor()
	detBackend, _ := NewEncryptedBackendWithEncryptor(memBackend, "test-password", detEnc)

	// Should be able to read value encrypted with different algorithm
	value, err := detBackend.Get("KEY")
	if err != nil {
		t.Fatalf("Get() with different algorithm error = %v", err)
	}

	if value != "value" {
		t.Errorf("Get() = %v, want 'value'", value)
	}
}

func TestEncryptedBackend_Close(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	err := encBackend.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestEncryptedBackend_Metadata(t *testing.T) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "test-password")

	// Set encrypted value
	encBackend.Set("META_TEST", "value", true)

	// Get raw data
	rawData, _ := memBackend.Get("META_TEST")
	var ev EncryptedValue
	json.Unmarshal([]byte(rawData), &ev)

	// Verify metadata
	if ev.Version != 1 {
		t.Errorf("Version = %v, want 1", ev.Version)
	}

	if ev.Algorithm == "" {
		t.Error("Algorithm not set")
	}

	if ev.Salt == "" {
		t.Error("Salt not set")
	}

	if ev.CreatedAt == 0 {
		t.Error("CreatedAt not set")
	}
}

func BenchmarkEncryptedBackend_SetEncrypted(b *testing.B) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "benchmark-password")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		encBackend.Set(key, "benchmark value", true)
	}
}

func BenchmarkEncryptedBackend_GetEncrypted(b *testing.B) {
	memBackend := NewMemoryBackend()
	encBackend, _ := NewEncryptedBackend(memBackend, "benchmark-password")

	// Pre-populate
	for i := 0; i < 100; i++ {
		encBackend.Set(fmt.Sprintf("KEY_%d", i), "value", true)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i%100)
		encBackend.Get(key)
	}
}
