package storage

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryBackend(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		// Test Set and Get
		err := backend.Set("TEST_KEY", "test_value", false)
		require.NoError(t, err)

		value, err := backend.Get("TEST_KEY")
		require.NoError(t, err)
		assert.Equal(t, "test_value", value)

		// Test Exists
		exists, err := backend.Exists("TEST_KEY")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = backend.Exists("NON_EXISTENT")
		require.NoError(t, err)
		assert.False(t, exists)

		// Test List
		err = backend.Set("ANOTHER_KEY", "another_value", false)
		require.NoError(t, err)

		keys, err := backend.List()
		require.NoError(t, err)
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "TEST_KEY")
		assert.Contains(t, keys, "ANOTHER_KEY")

		// Test Delete
		err = backend.Delete("TEST_KEY")
		require.NoError(t, err)

		_, err = backend.Get("TEST_KEY")
		assert.ErrorIs(t, err, ErrNotFound)

		exists, err = backend.Exists("TEST_KEY")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		_, err := backend.Get("NON_EXISTENT")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		// Should not error when deleting non-existent key
		err := backend.Delete("NON_EXISTENT")
		assert.NoError(t, err)
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		err := backend.Set("KEY", "value1", false)
		require.NoError(t, err)

		err = backend.Set("KEY", "value2", false)
		require.NoError(t, err)

		value, err := backend.Get("KEY")
		require.NoError(t, err)
		assert.Equal(t, "value2", value)
	})

	t.Run("empty key and value", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		// Empty key should be allowed
		err := backend.Set("", "empty_key", false)
		assert.NoError(t, err)

		// Empty value should be allowed
		err = backend.Set("EMPTY_VALUE", "", false)
		assert.NoError(t, err)

		value, err := backend.Get("EMPTY_VALUE")
		require.NoError(t, err)
		assert.Equal(t, "", value)
	})

	t.Run("list empty backend", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		keys, err := backend.List()
		require.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("encryption flag", func(t *testing.T) {
		backend := NewMemoryBackend()
		defer backend.Close()

		// For now, encryption flag is ignored in memory backend
		err := backend.Set("ENCRYPTED", "value", true)
		require.NoError(t, err)

		value, err := backend.Get("ENCRYPTED")
		require.NoError(t, err)
		assert.Equal(t, "value", value)
	})
}

func TestMemoryBackendConcurrency(t *testing.T) {
	backend := NewMemoryBackend()
	defer backend.Close()

	// Number of concurrent operations
	numGoroutines := 100
	numOpsPerGoroutine := 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOpsPerGoroutine)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("KEY_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				if err := backend.Set(key, value, false); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("KEY_%d_%d", id, j)
				expectedValue := fmt.Sprintf("value_%d_%d", id, j)
				
				// Wait for value to be set
				for attempts := 0; attempts < 10; attempts++ {
					if value, err := backend.Get(key); err == nil {
						if value != expectedValue {
							errors <- fmt.Errorf("got %s, want %s", value, expectedValue)
						}
						break
					}
				}
			}
		}(i)
	}

	// Concurrent exists checks
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("KEY_%d_%d", id, j)
				if _, err := backend.Exists(key); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	// Concurrent lists
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := backend.List(); err != nil {
					errors <- err
				}
			}
		}()
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines/4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine/2; j++ {
				key := fmt.Sprintf("KEY_%d_%d", id, j)
				if err := backend.Delete(key); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify final state
	keys, err := backend.List()
	require.NoError(t, err)
	t.Logf("Final number of keys: %d", len(keys))
}

func BenchmarkMemoryBackendSet(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		value := fmt.Sprintf("value_%d", i)
		_ = backend.Set(key, value, false)
	}
}

func BenchmarkMemoryBackendGet(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		value := fmt.Sprintf("value_%d", i)
		backend.Set(key, value, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("KEY_%d", i%1000)
		_, _ = backend.Get(key)
	}
}

func BenchmarkMemoryBackendList(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		value := fmt.Sprintf("value_%d", i)
		backend.Set(key, value, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = backend.List()
	}
}

func BenchmarkMemoryBackendConcurrentSet(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("KEY_%d", i)
			value := fmt.Sprintf("value_%d", i)
			_ = backend.Set(key, value, false)
			i++
		}
	})
}

func BenchmarkMemoryBackendConcurrentGet(b *testing.B) {
	backend := NewMemoryBackend()
	defer backend.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("KEY_%d", i)
		value := fmt.Sprintf("value_%d", i)
		backend.Set(key, value, false)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("KEY_%d", i%1000)
			_, _ = backend.Get(key)
			i++
		}
	})
}

func TestMemoryBackendStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	backend := NewMemoryBackend()
	defer backend.Close()

	// Create many keys
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("STRESS_KEY_%05d", i)
		value := fmt.Sprintf("stress_value_%05d", i)
		err := backend.Set(key, value, false)
		require.NoError(t, err)
	}

	// Verify all keys exist
	keys, err := backend.List()
	require.NoError(t, err)
	assert.Len(t, keys, numKeys)

	// Delete half the keys
	for i := 0; i < numKeys/2; i++ {
		key := fmt.Sprintf("STRESS_KEY_%05d", i)
		err := backend.Delete(key)
		require.NoError(t, err)
	}

	// Verify remaining keys
	keys, err = backend.List()
	require.NoError(t, err)
	assert.Len(t, keys, numKeys/2)

	// Verify correct keys remain
	for i := numKeys / 2; i < numKeys; i++ {
		key := fmt.Sprintf("STRESS_KEY_%05d", i)
		value, err := backend.Get(key)
		require.NoError(t, err)
		expectedValue := fmt.Sprintf("stress_value_%05d", i)
		assert.Equal(t, expectedValue, value)
	}
}