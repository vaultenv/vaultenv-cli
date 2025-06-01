package storage_test

import (
	"fmt"
	"log"

	"github.com/vaultenv/vaultenv-cli/pkg/storage"
)

func ExampleEncryptedBackend() {
	// Create a memory backend for this example
	memBackend := storage.NewMemoryBackend()
	
	// Wrap it with encryption using a password
	password := "my-secure-password"
	encBackend, err := storage.NewEncryptedBackend(memBackend, password)
	if err != nil {
		log.Fatal(err)
	}
	defer encBackend.Close()
	
	// Store an encrypted secret
	err = encBackend.Set("API_KEY", "sk-1234567890abcdef", true)
	if err != nil {
		log.Fatal(err)
	}
	
	// Store a non-encrypted value
	err = encBackend.Set("APP_NAME", "MyApp", false)
	if err != nil {
		log.Fatal(err)
	}
	
	// Retrieve the encrypted secret
	apiKey, err := encBackend.Get("API_KEY")
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("API Key: %s\n", apiKey)
	
	// List all keys
	keys, err := encBackend.List()
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Keys: %v\n", keys)
	
	// Output:
	// API Key: sk-1234567890abcdef
	// Keys: [API_KEY APP_NAME]
}

func ExampleGetBackendWithOptions() {
	// Create an encrypted backend using the helper function
	backend, err := storage.GetBackendWithOptions(storage.BackendOptions{
		Environment: "production",
		Password:    "secure-password-123",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()
	
	// Use the backend
	err = backend.Set("DATABASE_URL", "postgres://user:pass@host/db", true)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Encrypted value stored successfully")
	
	// Output:
	// Encrypted value stored successfully
}