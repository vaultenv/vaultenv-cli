package storage

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFileBackend(t *testing.T) {
    // Create temporary directory for testing
    tempDir, err := os.MkdirTemp("", "vaultenv-test")
    require.NoError(t, err)
    defer os.RemoveAll(tempDir)

    t.Run("create and basic operations", func(t *testing.T) {
        backend, err := NewFileBackend(tempDir, "test")
        require.NoError(t, err)
        defer backend.Close()

        // Test Set
        err = backend.Set("TEST_KEY", "test_value", false)
        assert.NoError(t, err)

        // Test Get
        value, err := backend.Get("TEST_KEY")
        assert.NoError(t, err)
        assert.Equal(t, "test_value", value)

        // Test Exists
        exists, err := backend.Exists("TEST_KEY")
        assert.NoError(t, err)
        assert.True(t, exists)

        // Test List
        keys, err := backend.List()
        assert.NoError(t, err)
        assert.Contains(t, keys, "TEST_KEY")

        // Test Delete
        err = backend.Delete("TEST_KEY")
        assert.NoError(t, err)

        // Verify deletion
        _, err = backend.Get("TEST_KEY")
        assert.ErrorIs(t, err, ErrNotFound)
    })

    t.Run("persistence across instances", func(t *testing.T) {
        // Create first backend instance
        backend1, err := NewFileBackend(tempDir, "persist")
        require.NoError(t, err)

        // Set some values
        err = backend1.Set("KEY1", "value1", false)
        assert.NoError(t, err)
        err = backend1.Set("KEY2", "value2", false)
        assert.NoError(t, err)

        // Close first backend
        backend1.Close()

        // Create second backend instance
        backend2, err := NewFileBackend(tempDir, "persist")
        require.NoError(t, err)
        defer backend2.Close()

        // Values should persist
        value1, err := backend2.Get("KEY1")
        assert.NoError(t, err)
        assert.Equal(t, "value1", value1)

        value2, err := backend2.Get("KEY2")
        assert.NoError(t, err)
        assert.Equal(t, "value2", value2)

        // List should show both keys
        keys, err := backend2.List()
        assert.NoError(t, err)
        assert.Len(t, keys, 2)
        assert.Contains(t, keys, "KEY1")
        assert.Contains(t, keys, "KEY2")
    })

    t.Run("separate environments", func(t *testing.T) {
        // Create backend for dev environment
        devBackend, err := NewFileBackend(tempDir, "development")
        require.NoError(t, err)
        defer devBackend.Close()

        // Create backend for prod environment
        prodBackend, err := NewFileBackend(tempDir, "production")
        require.NoError(t, err)
        defer prodBackend.Close()

        // Set values in different environments
        err = devBackend.Set("API_KEY", "dev-key", false)
        assert.NoError(t, err)

        err = prodBackend.Set("API_KEY", "prod-key", false)
        assert.NoError(t, err)

        // Values should be isolated
        devValue, err := devBackend.Get("API_KEY")
        assert.NoError(t, err)
        assert.Equal(t, "dev-key", devValue)

        prodValue, err := prodBackend.Get("API_KEY")
        assert.NoError(t, err)
        assert.Equal(t, "prod-key", prodValue)
    })

    t.Run("file creation", func(t *testing.T) {
        backend, err := NewFileBackend(tempDir, "filetest")
        require.NoError(t, err)
        defer backend.Close()

        // Set a value
        err = backend.Set("TEST", "value", false)
        assert.NoError(t, err)

        // Check that the file was created
        dataFile := filepath.Join(tempDir, "data", "filetest.json")
        assert.FileExists(t, dataFile)

        // Read the file directly
        data, err := os.ReadFile(dataFile)
        assert.NoError(t, err)
        assert.Contains(t, string(data), "TEST")
        assert.Contains(t, string(data), "value")
    })
}