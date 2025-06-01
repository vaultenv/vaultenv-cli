package keystore

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestKeystore(t *testing.T) {
	keystore := NewMockKeystore()
	service := "vaultenv-test"

	t.Run("StoreAndRetrieve", func(t *testing.T) {
		account := "test-account"
		data := make([]byte, 32)
		rand.Read(data)

		// Store
		err := keystore.Store(service, account, data)
		if err != nil {
			t.Fatalf("Store() error = %v", err)
		}

		// Retrieve
		retrieved, err := keystore.Retrieve(service, account)
		if err != nil {
			t.Fatalf("Retrieve() error = %v", err)
		}

		if !bytes.Equal(data, retrieved) {
			t.Error("Retrieved data does not match stored data")
		}
	})

	t.Run("StoreEmptyData", func(t *testing.T) {
		err := keystore.Store(service, "empty-account", []byte{})
		if err == nil {
			t.Error("Store() should error on empty data")
		}
	})

	t.Run("RetrieveNonExistent", func(t *testing.T) {
		_, err := keystore.Retrieve(service, "non-existent")
		if err == nil {
			t.Error("Retrieve() should error on non-existent key")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		account := "delete-test"
		data := []byte("test data")

		// Store
		keystore.Store(service, account, data)

		// Delete
		err := keystore.Delete(service, account)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Try to retrieve
		_, err = keystore.Retrieve(service, account)
		if err == nil {
			t.Error("Retrieve() should error after delete")
		}
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		// Should not error
		err := keystore.Delete(service, "non-existent")
		if err != nil {
			t.Errorf("Delete() non-existent error = %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		// Clear keystore
		keystore = NewMockKeystore()

		// Add some keys
		accounts := []string{"prod", "staging", "dev"}
		for _, acc := range accounts {
			keystore.Store(service, acc, []byte("data"))
		}

		// Add key for different service
		keystore.Store("other-service", "account", []byte("data"))

		// List
		listed, err := keystore.List(service)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(listed) != 3 {
			t.Errorf("List() returned %d accounts, want 3", len(listed))
		}

		// Check all accounts are present
		accountMap := make(map[string]bool)
		for _, acc := range listed {
			accountMap[acc] = true
		}

		for _, acc := range accounts {
			if !accountMap[acc] {
				t.Errorf("List() missing account %s", acc)
			}
		}
	})

	t.Run("ListEmpty", func(t *testing.T) {
		keystore := NewMockKeystore()
		
		listed, err := keystore.List("empty-service")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(listed) != 0 {
			t.Errorf("List() returned %d accounts, want 0", len(listed))
		}
	})

	t.Run("IsolationBetweenServices", func(t *testing.T) {
		keystore := NewMockKeystore()
		data1 := []byte("service1 data")
		data2 := []byte("service2 data")

		// Store for different services
		keystore.Store("service1", "account", data1)
		keystore.Store("service2", "account", data2)

		// Retrieve should get correct data
		retrieved1, _ := keystore.Retrieve("service1", "account")
		retrieved2, _ := keystore.Retrieve("service2", "account")

		if !bytes.Equal(retrieved1, data1) {
			t.Error("Service isolation failed for service1")
		}
		if !bytes.Equal(retrieved2, data2) {
			t.Error("Service isolation failed for service2")
		}
	})
}