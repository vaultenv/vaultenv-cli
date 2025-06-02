package encryption

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		algorithm string
		wantType  string
		wantErr   bool
	}{
		{"aes-gcm-256", "*encryption.AESGCMEncryptor", false},
		{"aes-gcm-256-deterministic", "*encryption.DeterministicEncryptor", false},
		{"chacha20-poly1305", "*encryption.ChaChaEncryptor", false},
		{"unknown-algorithm", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.algorithm, func(t *testing.T) {
			enc, err := NewEncryptor(tt.algorithm)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotType := fmt.Sprintf("%T", enc)
				if gotType != tt.wantType {
					t.Errorf("NewEncryptor() type = %v, want %v", gotType, tt.wantType)
				}
			}
		})
	}
}

func TestDefaultEncryptor(t *testing.T) {
	enc := DefaultEncryptor()

	// Should return AES-GCM encryptor
	if enc.Algorithm() != "aes-gcm-256" {
		t.Errorf("DefaultEncryptor() algorithm = %v, want aes-gcm-256", enc.Algorithm())
	}

	// Test it works
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := []byte("test default encryptor")
	encrypted, err := enc.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestEncryptor_Interface(t *testing.T) {
	// Test that all implementations satisfy the Encryptor interface
	encryptors := []Encryptor{
		NewAESGCMEncryptor(),
		NewDeterministicEncryptor(),
		NewChaChaEncryptor(),
	}

	for _, enc := range encryptors {
		t.Run(enc.Algorithm(), func(t *testing.T) {
			// Verify interface methods exist
			_ = enc.Algorithm()
			_ = enc.GenerateKey("password", []byte("salt"))

			// Skip further tests for unimplemented ChaCha
			if enc.Algorithm() == "chacha20-poly1305" {
				return
			}

			// Test full encryption cycle
			salt, err := enc.GenerateSalt()
			if err != nil {
				t.Fatalf("GenerateSalt() error = %v", err)
			}

			key := enc.GenerateKey("test-password", salt)

			plaintext := "test data"
			encrypted, err := enc.EncryptString(plaintext, key)
			if err != nil {
				t.Fatalf("EncryptString() error = %v", err)
			}

			decrypted, err := enc.DecryptString(encrypted, key)
			if err != nil {
				t.Fatalf("DecryptString() error = %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("DecryptString() = %v, want %v", decrypted, plaintext)
			}
		})
	}
}

func TestEncryption_CrossAlgorithm(t *testing.T) {
	// Test that different algorithms produce different outputs
	password := "test-password"
	plaintext := []byte("test data for cross-algorithm comparison")

	algorithms := []string{"aes-gcm-256", "aes-gcm-256-deterministic"}
	results := make(map[string][]byte)

	for _, algo := range algorithms {
		enc, err := NewEncryptor(algo)
		if err != nil {
			t.Fatalf("NewEncryptor(%s) error = %v", algo, err)
		}

		salt, _ := enc.GenerateSalt()
		key := enc.GenerateKey(password, salt)

		encrypted, err := enc.Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("%s Encrypt() error = %v", algo, err)
		}

		results[algo] = encrypted
	}

	// Verify different algorithms produce different outputs
	if bytes.Equal(results["aes-gcm-256"], results["aes-gcm-256-deterministic"]) {
		t.Error("Different algorithms produced identical ciphertext")
	}
}

func TestEncryption_KeyDerivation(t *testing.T) {
	enc := NewAESGCMEncryptor()

	password := "test-password-123"
	salt1, _ := enc.GenerateSalt()
	salt2, _ := enc.GenerateSalt()

	// Test that same password with different salts produces different keys
	key1 := enc.GenerateKey(password, salt1)
	key2 := enc.GenerateKey(password, salt2)

	if bytes.Equal(key1, key2) {
		t.Error("Same password with different salts produced identical keys")
	}

	// Test that different passwords with same salt produce different keys
	key3 := enc.GenerateKey("different-password", salt1)

	if bytes.Equal(key1, key3) {
		t.Error("Different passwords with same salt produced identical keys")
	}
}

func TestEncryption_LargeData(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	rand.Read(key)

	// Test with various sizes
	sizes := []int{
		1024,        // 1 KB
		1024 * 100,  // 100 KB
		1024 * 1024, // 1 MB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			plaintext := make([]byte, size)
			rand.Read(plaintext)

			encrypted, err := enc.Encrypt(plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			decrypted, err := enc.Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, plaintext) {
				t.Error("Decryption failed for large data")
			}
		})
	}
}

func TestEncryption_EdgeCases(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	rand.Read(key)

	// Test empty data
	encrypted, err := enc.Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := enc.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("Decrypt() length = %v, want 0", len(decrypted))
	}

	// Test single byte
	singleByte := []byte{0x42}
	encrypted, err = enc.Encrypt(singleByte, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err = enc.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, singleByte) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, singleByte)
	}
}

func TestMetadata_Struct(t *testing.T) {
	// Test that Metadata struct is properly defined
	metadata := Metadata{
		Algorithm: "test-algo",
		Version:   1,
		Salt:      []byte("test-salt"),
		Nonce:     []byte("test-nonce"),
		CreatedAt: 1234567890,
	}

	if metadata.Algorithm != "test-algo" {
		t.Errorf("Algorithm = %v, want test-algo", metadata.Algorithm)
	}

	if metadata.Version != 1 {
		t.Errorf("Version = %v, want 1", metadata.Version)
	}
}

func TestEncryptedData_Struct(t *testing.T) {
	// Test that EncryptedData struct is properly defined
	data := EncryptedData{
		Metadata: Metadata{
			Algorithm: "aes-gcm-256",
			Version:   1,
		},
		Ciphertext: []byte("encrypted-data"),
	}

	if data.Metadata.Algorithm != "aes-gcm-256" {
		t.Errorf("Metadata.Algorithm = %v, want aes-gcm-256", data.Metadata.Algorithm)
	}

	if !bytes.Equal(data.Ciphertext, []byte("encrypted-data")) {
		t.Errorf("Ciphertext = %v, want encrypted-data", data.Ciphertext)
	}
}
