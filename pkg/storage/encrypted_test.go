package storage

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewMockBackend creates a memory backend for testing
func NewMockBackend() Backend {
	return NewMemoryBackend()
}

func TestEncryptedBackend(t *testing.T) {
	// Create a mock backend
	mockBackend := NewMockBackend()
	
	// Create encrypted backend
	password := "test-password-123"
	encBackend, err := NewEncryptedBackend(mockBackend, password)
	require.NoError(t, err)
	require.NotNil(t, encBackend)
	
	t.Run("SetAndGetEncrypted", func(t *testing.T) {
		key := "SECRET_KEY"
		value := "super-secret-value"
		
		// Set encrypted value
		err := encBackend.Set(key, value, true)
		assert.NoError(t, err)
		
		// Get value back
		retrieved, err := encBackend.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
		
		// Verify it's actually encrypted in backend
		rawData, err := mockBackend.Get(key)
		assert.NoError(t, err)
		assert.NotEqual(t, value, rawData) // Should not be plaintext
		
		// Verify it's proper JSON
		var ev EncryptedValue
		err = json.Unmarshal([]byte(rawData), &ev)
		assert.NoError(t, err)
		assert.True(t, ev.IsEncrypted)
		assert.NotEmpty(t, ev.Salt)
		assert.NotEmpty(t, ev.Ciphertext)
		assert.Equal(t, "aes-gcm-256", ev.Algorithm)
	})
	
	t.Run("SetAndGetUnencrypted", func(t *testing.T) {
		key := "PLAIN_KEY"
		value := "plain-value"
		
		// Set unencrypted value
		err := encBackend.Set(key, value, false)
		assert.NoError(t, err)
		
		// Get value back
		retrieved, err := encBackend.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
		
		// Verify structure in backend
		rawData, err := mockBackend.Get(key)
		assert.NoError(t, err)
		
		var ev EncryptedValue
		err = json.Unmarshal([]byte(rawData), &ev)
		assert.NoError(t, err)
		assert.False(t, ev.IsEncrypted)
		assert.Equal(t, value, ev.Ciphertext)
	})
	
	t.Run("LegacyPlaintextSupport", func(t *testing.T) {
		key := "LEGACY_KEY"
		value := "legacy-plain-value"
		
		// Set directly in backend (simulating legacy data)
		err := mockBackend.Set(key, value, false)
		assert.NoError(t, err)
		
		// Should be able to get through encrypted backend
		retrieved, err := encBackend.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})
	
	t.Run("WrongPassword", func(t *testing.T) {
		key := "SECRET_KEY2"
		value := "another-secret"
		
		// Set with first password
		err := encBackend.Set(key, value, true)
		assert.NoError(t, err)
		
		// Create new backend with wrong password
		wrongBackend, err := NewEncryptedBackend(mockBackend, "wrong-password")
		require.NoError(t, err)
		
		// Should fail to decrypt
		_, err = wrongBackend.Get(key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decrypt")
	})
	
	t.Run("ExistsAndDelete", func(t *testing.T) {
		key := "TEST_KEY"
		value := "test-value"
		
		// Set value
		err := encBackend.Set(key, value, true)
		assert.NoError(t, err)
		
		// Check exists
		exists, err := encBackend.Exists(key)
		assert.NoError(t, err)
		assert.True(t, exists)
		
		// Delete
		err = encBackend.Delete(key)
		assert.NoError(t, err)
		
		// Check not exists
		exists, err = encBackend.Exists(key)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
	
	t.Run("List", func(t *testing.T) {
		// Clear backend
		mockBackend = NewMockBackend()
		encBackend, _ = NewEncryptedBackend(mockBackend, password)
		
		// Set some values
		keys := []string{"KEY1", "KEY2", "KEY3"}
		for _, key := range keys {
			err := encBackend.Set(key, "value-"+key, true)
			assert.NoError(t, err)
		}
		
		// List should return all keys
		listed, err := encBackend.List()
		assert.NoError(t, err)
		assert.ElementsMatch(t, keys, listed)
	})
	
	t.Run("UpdatePassword", func(t *testing.T) {
		// Clear backend
		mockBackend = NewMockBackend()
		encBackend, _ = NewEncryptedBackend(mockBackend, password)
		
		// Set some encrypted and unencrypted values
		err := encBackend.Set("ENCRYPTED1", "secret1", true)
		assert.NoError(t, err)
		err = encBackend.Set("ENCRYPTED2", "secret2", true)
		assert.NoError(t, err)
		err = encBackend.Set("PLAIN", "plain", false)
		assert.NoError(t, err)
		
		// Update password
		newPassword := "new-password-456"
		err = encBackend.UpdatePassword(password, newPassword)
		assert.NoError(t, err)
		
		// Create new backend with new password
		newBackend, err := NewEncryptedBackend(mockBackend, newPassword)
		require.NoError(t, err)
		
		// Should be able to read all values
		val1, err := newBackend.Get("ENCRYPTED1")
		assert.NoError(t, err)
		assert.Equal(t, "secret1", val1)
		
		val2, err := newBackend.Get("ENCRYPTED2")
		assert.NoError(t, err)
		assert.Equal(t, "secret2", val2)
		
		val3, err := newBackend.Get("PLAIN")
		assert.NoError(t, err)
		assert.Equal(t, "plain", val3)
		
		// Old password should not work
		oldBackend, err := NewEncryptedBackend(mockBackend, password)
		require.NoError(t, err)
		_, err = oldBackend.Get("ENCRYPTED1")
		assert.Error(t, err)
	})
}

func TestEncryptedBackendValidation(t *testing.T) {
	t.Run("NilBackend", func(t *testing.T) {
		_, err := NewEncryptedBackend(nil, "password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "backend cannot be nil")
	})
	
	t.Run("EmptyPassword", func(t *testing.T) {
		mockBackend := NewMockBackend()
		_, err := NewEncryptedBackend(mockBackend, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password cannot be empty")
	})
}

func TestGetBackendWithEncryption(t *testing.T) {
	// Test GetBackend without encryption (backward compatibility)
	backend, err := GetBackend("test")
	assert.NoError(t, err)
	assert.NotNil(t, backend)
	_, isEncrypted := backend.(*EncryptedBackend)
	assert.False(t, isEncrypted)
	backend.Close()
	
	// Test GetBackendWithOptions with encryption
	encBackend, err := GetBackendWithOptions(BackendOptions{
		Environment: "test",
		Password:    "test-password",
	})
	assert.NoError(t, err)
	assert.NotNil(t, encBackend)
	_, isEncrypted = encBackend.(*EncryptedBackend)
	assert.True(t, isEncrypted)
	encBackend.Close()
}

// BenchmarkEncryptedBackend benchmarks the encrypted backend operations
func BenchmarkEncryptedBackend(b *testing.B) {
	mockBackend := NewMockBackend()
	encBackend, _ := NewEncryptedBackend(mockBackend, "benchmark-password")
	
	b.Run("SetEncrypted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("KEY_%d", i)
			value := fmt.Sprintf("value_%d", i)
			encBackend.Set(key, value, true)
		}
	})
	
	b.Run("GetEncrypted", func(b *testing.B) {
		// Setup
		key := "BENCH_KEY"
		value := "benchmark-value"
		encBackend.Set(key, value, true)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			encBackend.Get(key)
		}
	})
	
	b.Run("SetUnencrypted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("PLAIN_%d", i)
			value := fmt.Sprintf("plain_%d", i)
			encBackend.Set(key, value, false)
		}
	})
}