package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteBackend(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "vaultenv-sqlite-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create backend
	backend, err := NewSQLiteBackend(tempDir, "test")
	require.NoError(t, err)
	defer backend.Close()

	t.Run("Set and Get", func(t *testing.T) {
		// Set a value
		err := backend.Set("TEST_KEY", "test_value", false)
		assert.NoError(t, err)

		// Get the value
		value, err := backend.Get("TEST_KEY")
		assert.NoError(t, err)
		assert.Equal(t, "test_value", value)
	})

	t.Run("Update existing key", func(t *testing.T) {
		// Set initial value
		err := backend.Set("UPDATE_KEY", "initial", false)
		assert.NoError(t, err)

		// Update value
		err = backend.Set("UPDATE_KEY", "updated", false)
		assert.NoError(t, err)

		// Verify updated value
		value, err := backend.Get("UPDATE_KEY")
		assert.NoError(t, err)
		assert.Equal(t, "updated", value)
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		_, err := backend.Get("NON_EXISTENT")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Exists", func(t *testing.T) {
		// Set a value
		err := backend.Set("EXISTS_KEY", "value", false)
		assert.NoError(t, err)

		// Check existence
		exists, err := backend.Exists("EXISTS_KEY")
		assert.NoError(t, err)
		assert.True(t, exists)

		// Check non-existent
		exists, err = backend.Exists("NOT_EXISTS")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		// Set a value
		err := backend.Set("DELETE_KEY", "value", false)
		assert.NoError(t, err)

		// Delete it
		err = backend.Delete("DELETE_KEY")
		assert.NoError(t, err)

		// Verify it's gone
		_, err = backend.Get("DELETE_KEY")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Delete non-existent key", func(t *testing.T) {
		err := backend.Delete("NON_EXISTENT")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("List", func(t *testing.T) {
		// Clear any existing data
		keys, err := backend.List()
		assert.NoError(t, err)
		for _, key := range keys {
			backend.Delete(key)
		}

		// Set multiple values
		testKeys := []string{"LIST_KEY1", "LIST_KEY2", "LIST_KEY3"}
		for _, key := range testKeys {
			err := backend.Set(key, "value", false)
			assert.NoError(t, err)
		}

		// List keys
		keys, err = backend.List()
		assert.NoError(t, err)
		assert.Len(t, keys, 3)
		assert.ElementsMatch(t, testKeys, keys)
	})
}

func TestSQLiteBackendHistory(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "vaultenv-sqlite-history-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create backend
	backend, err := NewSQLiteBackend(tempDir, "test")
	require.NoError(t, err)
	defer backend.Close()

	t.Run("Track history", func(t *testing.T) {
		key := "HISTORY_KEY"

		// Set initial value
		err := backend.Set(key, "value1", false)
		assert.NoError(t, err)

		// Update value
		err = backend.Set(key, "value2", false)
		assert.NoError(t, err)

		// Update again
		err = backend.Set(key, "value3", false)
		assert.NoError(t, err)

		// Get history
		history, err := backend.GetHistory(key, 10)
		assert.NoError(t, err)
		assert.Len(t, history, 3)

		// Verify history order (newest first)
		assert.Equal(t, 3, history[0].Version)
		assert.Equal(t, "value3", history[0].Value)
		assert.Equal(t, "SET", history[0].ChangeType)

		assert.Equal(t, 2, history[1].Version)
		assert.Equal(t, "value2", history[1].Value)

		assert.Equal(t, 1, history[2].Version)
		assert.Equal(t, "value1", history[2].Value)
	})

	t.Run("Delete history", func(t *testing.T) {
		key := "DELETE_HISTORY_KEY"

		// Set and delete
		err := backend.Set(key, "to_delete", false)
		assert.NoError(t, err)

		err = backend.Delete(key)
		assert.NoError(t, err)

		// Get history
		history, err := backend.GetHistory(key, 10)
		assert.NoError(t, err)
		assert.Len(t, history, 2)

		// Verify delete is tracked
		assert.Equal(t, 2, history[0].Version)
		assert.Equal(t, "DELETE", history[0].ChangeType)
	})

	t.Run("Limit history results", func(t *testing.T) {
		key := "LIMIT_HISTORY_KEY"

		// Create many versions
		for i := 1; i <= 10; i++ {
			err := backend.Set(key, fmt.Sprintf("value%d", i), false)
			assert.NoError(t, err)
		}

		// Get limited history
		history, err := backend.GetHistory(key, 3)
		assert.NoError(t, err)
		assert.Len(t, history, 3)

		// Should get the 3 most recent
		assert.Equal(t, 10, history[0].Version)
		assert.Equal(t, 9, history[1].Version)
		assert.Equal(t, 8, history[2].Version)
	})
}

func TestSQLiteBackendAuditLog(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "vaultenv-sqlite-audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create backend
	backend, err := NewSQLiteBackend(tempDir, "test")
	require.NoError(t, err)
	defer backend.Close()

	t.Run("Audit log entries", func(t *testing.T) {
		// Perform various operations
		err := backend.Set("AUDIT_KEY1", "value1", false)
		assert.NoError(t, err)

		_, err = backend.Get("AUDIT_KEY1")
		assert.NoError(t, err)

		err = backend.Delete("AUDIT_KEY1")
		assert.NoError(t, err)

		// Wait a bit for async audit log
		time.Sleep(100 * time.Millisecond)

		// Get audit log
		entries, err := backend.GetAuditLog(10)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 3)

		// Verify actions are logged
		actions := make(map[string]bool)
		for _, entry := range entries {
			actions[entry.Action] = true
			assert.NotEmpty(t, entry.User)
			assert.True(t, entry.Success)
		}

		assert.True(t, actions["SET"])
		assert.True(t, actions["GET"])
		assert.True(t, actions["DELETE"])
	})
}

func TestSQLiteBackendConcurrency(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "vaultenv-sqlite-concurrent-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create backend
	backend, err := NewSQLiteBackend(tempDir, "test")
	require.NoError(t, err)
	defer backend.Close()

	t.Run("Concurrent writes", func(t *testing.T) {
		done := make(chan bool, 10)

		// Start 10 concurrent writers
		for i := 0; i < 10; i++ {
			go func(n int) {
				key := fmt.Sprintf("CONCURRENT_%d", n)
				value := fmt.Sprintf("value_%d", n)
				err := backend.Set(key, value, false)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all values
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("CONCURRENT_%d", i)
			value, err := backend.Get(key)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("value_%d", i), value)
		}
	})
}

func TestSQLiteBackendWithEncryption(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "vaultenv-sqlite-encrypt-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create SQLite backend
	sqliteBackend, err := NewSQLiteBackend(tempDir, "test")
	require.NoError(t, err)
	defer sqliteBackend.Close()

	// Wrap with encryption
	encryptedBackend, err := NewEncryptedBackend(sqliteBackend, "test-password")
	require.NoError(t, err)

	t.Run("Encrypted storage", func(t *testing.T) {
		// Set encrypted value
		err := encryptedBackend.Set("SECRET_KEY", "secret_value", true)
		assert.NoError(t, err)

		// Get through encrypted backend
		value, err := encryptedBackend.Get("SECRET_KEY")
		assert.NoError(t, err)
		assert.Equal(t, "secret_value", value)

		// Try to get directly from SQLite (should be encrypted)
		rawValue, err := sqliteBackend.Get("SECRET_KEY")
		assert.NoError(t, err)
		assert.NotEqual(t, "secret_value", rawValue)
		assert.Contains(t, rawValue, "encrypted")
	})
}