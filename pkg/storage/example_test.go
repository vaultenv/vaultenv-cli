package storage_test

import (
	"fmt"
	"log"

	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

// ExampleBackend demonstrates basic usage of the storage backend
func ExampleBackend() {
	// Get a backend for the "development" environment
	backend, err := storage.GetBackend("development")
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	// Set a variable
	err = backend.Set("API_KEY", "secret-key-value", false)
	if err != nil {
		log.Fatal(err)
	}

	// Get a variable
	value, err := backend.Get("API_KEY")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(value)
	// Output: secret-key-value
}

// ExampleBackendOptions demonstrates creating a backend with options
func ExampleBackendOptions() {
	// Create an encrypted SQLite backend
	opts := storage.BackendOptions{
		Environment: "production",
		Type:        "sqlite",
		Password:    "my-secure-password",
		BasePath:    "/var/vaultenv",
	}

	backend, err := storage.GetBackendWithOptions(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	// Store encrypted data
	err = backend.Set("DATABASE_URL", "postgresql://...", true)
	if err != nil {
		log.Fatal(err)
	}

	// List all variables
	keys, err := backend.List()
	if err != nil {
		log.Fatal(err)
	}

	for _, key := range keys {
		fmt.Println(key)
	}
}

// ExampleMemoryBackend demonstrates using the memory backend for testing
func ExampleMemoryBackend() {
	// Create an in-memory backend
	backend := storage.NewMemoryBackend()

	// Set some test data
	backend.Set("TEST_VAR", "test-value", false)
	backend.Set("ANOTHER_VAR", "another-value", false)

	// List all variables
	keys, err := backend.List()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d variables\n", len(keys))
	// Output: Found 2 variables
}

// ExampleEncryptedBackend demonstrates using encryption
func ExampleEncryptedBackend() {
	// Create a memory backend for this example
	memBackend := storage.NewMemoryBackend()

	// Wrap it with encryption
	encBackend, err := storage.NewEncryptedBackend(memBackend, "my-password")
	if err != nil {
		log.Fatal(err)
	}

	// Store sensitive data encrypted
	err = encBackend.Set("SECRET_TOKEN", "super-secret-value", true)
	if err != nil {
		log.Fatal(err)
	}

	// Store non-sensitive data in plain text
	err = encBackend.Set("APP_NAME", "MyApp", false)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve data (automatically decrypted if needed)
	secret, _ := encBackend.Get("SECRET_TOKEN")
	appName, _ := encBackend.Get("APP_NAME")

	fmt.Printf("App: %s\n", appName)
	fmt.Printf("Secret: %s\n", secret)
	// Output:
	// App: MyApp
	// Secret: super-secret-value
}

// ExampleHistoryBackend demonstrates using the history features
func ExampleHistoryBackend() {
	// SQLite backend supports history
	backend, err := storage.NewSQLiteBackend(".vaultenv", "development")
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	// Set and update a value multiple times
	backend.Set("VERSION", "1.0.0", false)
	backend.Set("VERSION", "1.0.1", false)
	backend.Set("VERSION", "1.1.0", false)

	// Get history
	history, err := backend.GetHistory("VERSION", 10)
	if err != nil {
		log.Fatal(err)
	}

	// Display version history
	for _, h := range history {
		fmt.Printf("Version %d: %s (changed at %s)\n",
			h.Version, h.Value, h.ChangedAt.Format("2006-01-02 15:04:05"))
	}
}

// ExampleSetTestBackend demonstrates using a test backend
func ExampleSetTestBackend() {
	// Create a mock backend for testing
	testBackend := storage.NewMemoryBackend()
	testBackend.Set("TEST_MODE", "true", false)

	// Set it as the test backend
	storage.SetTestBackend(testBackend)

	// Now all calls to GetBackend will return the test backend
	backend, _ := storage.GetBackend("any-environment")
	value, _ := backend.Get("TEST_MODE")

	fmt.Println(value)
	// Output: true

	// Reset when done testing
	storage.ResetTestBackend()
}