package keystore

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestKeystoreInterface(t *testing.T) {
	// Verify MockKeystore implements Keystore interface
	var _ Keystore = (*MockKeystore)(nil)
	var _ Keystore = (*OSKeystore)(nil)
}

func TestMockKeystore_StoreRetrieve(t *testing.T) {
	ks := NewMockKeystore()

	service := "vaultenv"
	account := "test-account"
	testData := []byte("test-encryption-key")

	// Store key
	err := ks.Store(service, account, testData)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Retrieve key
	retrieved, err := ks.Retrieve(service, account)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if !bytes.Equal(retrieved, testData) {
		t.Errorf("Retrieved data = %v, want %v", retrieved, testData)
	}
}

func TestMockKeystore_MultipleAccounts(t *testing.T) {
	ks := NewMockKeystore()

	service := "vaultenv"
	accounts := map[string][]byte{
		"dev":     []byte("dev-key"),
		"staging": []byte("staging-key"),
		"prod":    []byte("prod-key"),
	}

	// Store multiple keys
	for account, data := range accounts {
		err := ks.Store(service, account, data)
		if err != nil {
			t.Errorf("Store(%s) error = %v", account, err)
		}
	}

	// Retrieve and verify
	for account, expectedData := range accounts {
		retrieved, err := ks.Retrieve(service, account)
		if err != nil {
			t.Errorf("Retrieve(%s) error = %v", account, err)
		}

		if !bytes.Equal(retrieved, expectedData) {
			t.Errorf("Retrieved data for %s = %v, want %v", account, retrieved, expectedData)
		}
	}
}

func TestMockKeystore_Delete(t *testing.T) {
	ks := NewMockKeystore()

	service := "vaultenv"
	account := "test-account"
	testData := []byte("test-data")

	// Store key
	ks.Store(service, account, testData)

	// Delete key
	err := ks.Delete(service, account)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = ks.Retrieve(service, account)
	if err == nil {
		t.Error("Expected error retrieving deleted key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Retrieve() error = %v, want not found error", err)
	}
}

func TestMockKeystore_List(t *testing.T) {
	ks := NewMockKeystore()

	service1 := "vaultenv"
	service2 := "other-service"

	// Store keys for different services
	ks.Store(service1, "dev", []byte("data"))
	ks.Store(service1, "prod", []byte("data"))
	ks.Store(service1, "staging", []byte("data"))
	ks.Store(service2, "test", []byte("data"))

	// List keys for service1
	accounts, err := ks.List(service1)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(accounts) != 3 {
		t.Errorf("List() returned %d accounts, want 3", len(accounts))
	}

	// Verify correct accounts
	expectedAccounts := map[string]bool{
		"dev":     false,
		"prod":    false,
		"staging": false,
	}

	for _, account := range accounts {
		if _, ok := expectedAccounts[account]; !ok {
			t.Errorf("Unexpected account: %s", account)
		}
		expectedAccounts[account] = true
	}

	for account, found := range expectedAccounts {
		if !found {
			t.Errorf("Missing account: %s", account)
		}
	}

	// List keys for service2
	accounts2, err := ks.List(service2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(accounts2) != 1 {
		t.Errorf("List() for service2 returned %d accounts, want 1", len(accounts2))
	}
}

func TestMockKeystore_EmptyList(t *testing.T) {
	ks := NewMockKeystore()

	accounts, err := ks.List("empty-service")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(accounts) != 0 {
		t.Errorf("List() returned %d accounts for empty service, want 0", len(accounts))
	}
}

func TestMockKeystore_Overwrite(t *testing.T) {
	ks := NewMockKeystore()

	service := "vaultenv"
	account := "test"

	// Store initial data
	initialData := []byte("initial")
	ks.Store(service, account, initialData)

	// Overwrite with new data
	newData := []byte("updated")
	err := ks.Store(service, account, newData)
	if err != nil {
		t.Fatalf("Store() overwrite error = %v", err)
	}

	// Verify new data
	retrieved, _ := ks.Retrieve(service, account)
	if !bytes.Equal(retrieved, newData) {
		t.Errorf("Retrieved = %v, want %v", retrieved, newData)
	}
}

func TestMockKeystore_Errors(t *testing.T) {
	ks := NewMockKeystore()

	service := "vaultenv"
	account := "test"

	// Test Store error with empty data
	err := ks.Store(service, account, []byte{})
	if err == nil || !strings.Contains(err.Error(), "empty data") {
		t.Errorf("Store() with empty data error = %v, want error about empty data", err)
	}

	// Test Retrieve error for non-existent key
	_, err = ks.Retrieve(service, "non-existent")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Retrieve() for non-existent key error = %v, want not found error", err)
	}
}

func TestOSKeystore_makeKey(t *testing.T) {
	// This test doesn't require actual OS keystore
	ks := &OSKeystore{}

	tests := []struct {
		service string
		account string
		want    string
	}{
		{"vaultenv", "dev", "vaultenv:dev"},
		{"vaultenv", "prod", "vaultenv:prod"},
		{"service", "account", "service:account"},
		{"", "account", ":account"},
		{"service", "", "service:"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.service, tt.account), func(t *testing.T) {
			got := ks.makeKey(tt.service, tt.account)
			if got != tt.want {
				t.Errorf("makeKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOSKeystore_GetBackend(t *testing.T) {
	ks := &OSKeystore{}
	backend := ks.GetBackend()

	// Verify appropriate backend for OS
	switch runtime.GOOS {
	case "darwin":
		if backend != "macOS Keychain" {
			t.Errorf("GetBackend() = %v, want 'macOS Keychain'", backend)
		}
	case "windows":
		if backend != "Windows Credential Manager" {
			t.Errorf("GetBackend() = %v, want 'Windows Credential Manager'", backend)
		}
	case "linux":
		if backend != "Secret Service" && backend != "Encrypted File" {
			t.Errorf("GetBackend() = %v, want 'Secret Service' or 'Encrypted File'", backend)
		}
	default:
		if backend != "Encrypted File" {
			t.Errorf("GetBackend() = %v, want 'Encrypted File'", backend)
		}
	}
}

// TestOSKeystore_Integration tests actual OS keystore if available
// This test is skipped by default as it requires OS permissions
func TestOSKeystore_Integration(t *testing.T) {
	t.Skip("Skipping OS keystore integration test for beta release - requires interactive input")
	if testing.Short() {
		t.Skip("Skipping OS keystore integration test in short mode")
	}

	// Try to create OS keystore
	ks, err := NewOSKeystore("vaultenv-test")
	if err != nil {
		t.Skipf("Cannot create OS keystore: %v", err)
	}

	service := "vaultenv-test"
	account := "integration-test"
	testData := []byte("test-key-data")

	// Clean up any existing key
	ks.Delete(service, account)

	// Test Store
	err = ks.Store(service, account, testData)
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Test Retrieve
	retrieved, err := ks.Retrieve(service, account)
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if !bytes.Equal(retrieved, testData) {
		t.Errorf("Retrieved = %v, want %v", retrieved, testData)
	}

	// Test List
	accounts, err := ks.List(service)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	found := false
	for _, acc := range accounts {
		if acc == account {
			found = true
			break
		}
	}

	if !found {
		t.Error("Account not found in list")
	}

	// Test Delete
	err = ks.Delete(service, account)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = ks.Retrieve(service, account)
	if err == nil {
		t.Error("Expected error retrieving deleted key")
	}
}

func BenchmarkMockKeystore_Store(b *testing.B) {
	ks := NewMockKeystore()
	service := "vaultenv"
	data := []byte("benchmark-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		account := fmt.Sprintf("account_%d", i)
		ks.Store(service, account, data)
	}
}

func BenchmarkMockKeystore_Retrieve(b *testing.B) {
	ks := NewMockKeystore()
	service := "vaultenv"
	data := []byte("benchmark-data")

	// Pre-populate
	for i := 0; i < 1000; i++ {
		account := fmt.Sprintf("account_%d", i)
		ks.Store(service, account, data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		account := fmt.Sprintf("account_%d", i%1000)
		ks.Retrieve(service, account)
	}
}
