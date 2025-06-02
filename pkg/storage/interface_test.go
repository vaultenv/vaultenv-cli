package storage

import (
	"testing"
)

func TestGetBackend(t *testing.T) {
	// Test with default options
	backend, err := GetBackend("test-env")
	if err != nil {
		t.Fatalf("GetBackend() error = %v", err)
	}
	defer backend.Close()

	// Should return a file backend by default
	if _, ok := backend.(*FileBackend); !ok {
		t.Error("GetBackend() did not return FileBackend")
	}
}

func TestGetBackendWithOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    BackendOptions
		wantErr bool
	}{
		{
			name: "file_backend",
			opts: BackendOptions{
				Environment: "test",
				Type:        "file",
				BasePath:    ".test",
			},
			wantErr: false,
		},
		{
			name: "sqlite_backend",
			opts: BackendOptions{
				Environment: "test",
				Type:        "sqlite",
				BasePath:    ".test",
			},
			wantErr: false,
		},
		{
			name: "git_backend",
			opts: BackendOptions{
				Environment: "test",
				Type:        "git",
				BasePath:    ".test",
			},
			wantErr: false,
		},
		{
			name: "encrypted_backend",
			opts: BackendOptions{
				Environment: "test",
				Type:        "file",
				BasePath:    ".test",
				Password:    "test-password",
			},
			wantErr: false,
		},
		{
			name: "invalid_type",
			opts: BackendOptions{
				Environment: "test",
				Type:        "invalid",
			},
			wantErr: true,
		},
		{
			name: "default_base_path",
			opts: BackendOptions{
				Environment: "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := GetBackendWithOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBackendWithOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				defer backend.Close()

				// Verify correct backend type
				switch tt.opts.Type {
				case "file", "":
					if tt.opts.Password != "" {
						if _, ok := backend.(*EncryptedBackend); !ok {
							t.Error("Expected EncryptedBackend for password option")
						}
					} else {
						if _, ok := backend.(*FileBackend); !ok {
							t.Error("Expected FileBackend")
						}
					}
				case "sqlite":
					if tt.opts.Password != "" {
						if _, ok := backend.(*EncryptedBackend); !ok {
							t.Error("Expected EncryptedBackend for password option")
						}
					} else {
						if _, ok := backend.(*SQLiteBackend); !ok {
							t.Error("Expected SQLiteBackend")
						}
					}
				case "git":
					if tt.opts.Password != "" {
						if _, ok := backend.(*EncryptedBackend); !ok {
							t.Error("Expected EncryptedBackend for password option")
						}
					} else {
						if _, ok := backend.(*GitBackend); !ok {
							t.Error("Expected GitBackend")
						}
					}
				}
			}
		})
	}
}

func TestTestBackend(t *testing.T) {
	// Create a test backend
	testBackend := NewMemoryBackend()
	testBackend.Set("TEST_KEY", "test_value", false)

	// Set test backend
	SetTestBackend(testBackend)

	// GetBackend should return test backend
	backend, err := GetBackend("any-env")
	if err != nil {
		t.Fatalf("GetBackend() with test backend error = %v", err)
	}

	// Verify it's the test backend
	value, err := backend.Get("TEST_KEY")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if value != "test_value" {
		t.Errorf("Get() = %v, want test_value", value)
	}

	// Reset test backend
	ResetTestBackend()

	// GetBackend should now return regular backend
	backend2, err := GetBackend("test-env")
	if err != nil {
		t.Fatalf("GetBackend() after reset error = %v", err)
	}
	defer backend2.Close()

	// Should not have test data
	_, err = backend2.Get("TEST_KEY")
	if err != ErrNotFound {
		t.Error("Expected ErrNotFound after resetting test backend")
	}
}

func TestBackendInterface(t *testing.T) {
	// Test that all backend implementations satisfy the Backend interface
	backends := []Backend{
		NewMemoryBackend(),
		// File, SQLite, and Git backends require paths, tested separately
	}

	for _, backend := range backends {
		// Test all interface methods exist
		_ = backend.Set("key", "value", false)
		_, _ = backend.Get("key")
		_, _ = backend.Exists("key")
		_ = backend.Delete("key")
		_, _ = backend.List()
		_ = backend.Close()
	}
}

func TestBackendErrors(t *testing.T) {
	// Test common error values
	if ErrNotFound.Error() != "variable not found" {
		t.Errorf("ErrNotFound = %v, want 'variable not found'", ErrNotFound)
	}

	if ErrAlreadyExists.Error() != "variable already exists" {
		t.Errorf("ErrAlreadyExists = %v, want 'variable already exists'", ErrAlreadyExists)
	}

	if ErrInvalidName.Error() != "invalid variable name" {
		t.Errorf("ErrInvalidName = %v, want 'invalid variable name'", ErrInvalidName)
	}
}

func TestHistoryBackendInterface(t *testing.T) {
	// Verify SQLiteBackend implements HistoryBackend
	var _ HistoryBackend = (*SQLiteBackend)(nil)
}

// TestBackendCompatibility tests that all backends work the same way
func TestBackendCompatibility(t *testing.T) {
	testData := map[string]string{
		"KEY1":      "value1",
		"KEY2":      "value with spaces",
		"EMPTY":     "",
		"UNICODE":   "Hello 世界",
		"MULTILINE": "line1\nline2\nline3",
	}

	// Create different backend types
	memBackend := NewMemoryBackend()

	backends := []struct {
		name    string
		backend Backend
	}{
		{"memory", memBackend},
	}

	for _, b := range backends {
		t.Run(b.name, func(t *testing.T) {
			backend := b.backend
			defer backend.Close()

			// Test Set/Get
			for key, value := range testData {
				err := backend.Set(key, value, false)
				if err != nil {
					t.Errorf("Set(%s) error = %v", key, err)
				}

				got, err := backend.Get(key)
				if err != nil {
					t.Errorf("Get(%s) error = %v", key, err)
				}
				if got != value {
					t.Errorf("Get(%s) = %v, want %v", key, got, value)
				}
			}

			// Test List
			keys, err := backend.List()
			if err != nil {
				t.Errorf("List() error = %v", err)
			}
			if len(keys) != len(testData) {
				t.Errorf("List() returned %d keys, want %d", len(keys), len(testData))
			}

			// Test Exists
			exists, err := backend.Exists("KEY1")
			if err != nil {
				t.Errorf("Exists() error = %v", err)
			}
			if !exists {
				t.Error("Exists() = false, want true")
			}

			exists, err = backend.Exists("NONEXISTENT")
			if err != nil {
				t.Errorf("Exists() error = %v", err)
			}
			if exists {
				t.Error("Exists() = true for non-existent key")
			}

			// Test Delete
			err = backend.Delete("KEY1")
			if err != nil {
				t.Errorf("Delete() error = %v", err)
			}

			_, err = backend.Get("KEY1")
			if err != ErrNotFound {
				t.Error("Get() after Delete() should return ErrNotFound")
			}

			// Test overwrite
			err = backend.Set("KEY2", "updated value", false)
			if err != nil {
				t.Errorf("Set() overwrite error = %v", err)
			}

			value, _ := backend.Get("KEY2")
			if value != "updated value" {
				t.Errorf("Get() after overwrite = %v, want 'updated value'", value)
			}
		})
	}
}
