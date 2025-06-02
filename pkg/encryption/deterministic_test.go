package encryption

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestDeterministicEncryptor_Algorithm(t *testing.T) {
	enc := NewDeterministicEncryptor()
	if got := enc.Algorithm(); got != "aes-gcm-256-deterministic" {
		t.Errorf("Algorithm() = %v, want %v", got, "aes-gcm-256-deterministic")
	}
}

func TestDeterministicEncryptor_EncryptDeterministic(t *testing.T) {
	enc := NewDeterministicEncryptor()
	
	// Generate test key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	
	plaintext := []byte("test data")
	context := []byte("test-context")
	
	// Test deterministic encryption
	ciphertext1, err := enc.EncryptDeterministic(plaintext, key, context)
	if err != nil {
		t.Fatalf("EncryptDeterministic() error = %v", err)
	}
	
	// Encrypt again with same inputs
	ciphertext2, err := enc.EncryptDeterministic(plaintext, key, context)
	if err != nil {
		t.Fatalf("EncryptDeterministic() error = %v", err)
	}
	
	// Should produce identical ciphertext
	if !bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("EncryptDeterministic() not deterministic for same inputs")
	}
	
	// Test with different context
	ciphertext3, err := enc.EncryptDeterministic(plaintext, key, []byte("different-context"))
	if err != nil {
		t.Fatalf("EncryptDeterministic() error = %v", err)
	}
	
	// Should produce different ciphertext
	if bytes.Equal(ciphertext1, ciphertext3) {
		t.Error("EncryptDeterministic() produced same ciphertext for different contexts")
	}
	
	// Test decryption
	decrypted, err := enc.DecryptDeterministic(ciphertext1, key, context)
	if err != nil {
		t.Fatalf("DecryptDeterministic() error = %v", err)
	}
	
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("DecryptDeterministic() = %v, want %v", decrypted, plaintext)
	}
}

func TestDeterministicEncryptor_Encrypt(t *testing.T) {
	enc := NewDeterministicEncryptor()
	
	// Generate test key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	
	plaintext := []byte("test data")
	
	// Test encryption (uses empty context)
	ciphertext1, err := enc.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	// Should be deterministic with empty context
	ciphertext2, err := enc.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	
	if !bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Encrypt() not deterministic")
	}
	
	// Test decryption
	decrypted, err := enc.Decrypt(ciphertext1, key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}
}

func TestDeterministicEncryptor_EncryptDecryptString(t *testing.T) {
	enc := NewDeterministicEncryptor()
	
	// Generate test key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	
	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty", ""},
		{"simple", "hello world"},
		{"unicode", "Hello 世界 🌍"},
		{"special", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext1, err := enc.EncryptString(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptString() error = %v", err)
			}
			
			// Verify base64 encoding
			if _, err := base64.StdEncoding.DecodeString(ciphertext1); err != nil {
				t.Errorf("EncryptString() produced invalid base64: %v", err)
			}
			
			// Should be deterministic
			ciphertext2, err := enc.EncryptString(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptString() error = %v", err)
			}
			
			if ciphertext1 != ciphertext2 {
				t.Error("EncryptString() not deterministic")
			}
			
			// Decrypt
			decrypted, err := enc.DecryptString(ciphertext1, key)
			if err != nil {
				t.Fatalf("DecryptString() error = %v", err)
			}
			
			if decrypted != tt.plaintext {
				t.Errorf("DecryptString() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestDeterministicEncryptor_GenerateKey(t *testing.T) {
	enc := NewDeterministicEncryptor()
	
	salt, err := enc.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	
	password := "test-password"
	
	// Test key generation
	key1 := enc.GenerateKey(password, salt)
	if len(key1) != 32 {
		t.Errorf("GenerateKey() length = %v, want 32", len(key1))
	}
	
	// Should be deterministic
	key2 := enc.GenerateKey(password, salt)
	if !bytes.Equal(key1, key2) {
		t.Error("GenerateKey() not deterministic")
	}
}

func TestDeterministicEncryptor_GenerateSalt(t *testing.T) {
	enc := NewDeterministicEncryptor()
	
	salt1, err := enc.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	
	if len(salt1) != 32 {
		t.Errorf("GenerateSalt() length = %v, want 32", len(salt1))
	}
	
	// Salts should be unique
	salt2, err := enc.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	
	if bytes.Equal(salt1, salt2) {
		t.Error("GenerateSalt() generated identical salts")
	}
}

func TestDeterministicEncryptor_NonceDerivation(t *testing.T) {
	enc := NewDeterministicEncryptor()
	
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	
	tests := []struct {
		name      string
		plaintext []byte
		context   []byte
	}{
		{"empty_all", []byte{}, []byte{}},
		{"same_plaintext_diff_context", []byte("test"), []byte("ctx1")},
		{"same_plaintext_diff_context2", []byte("test"), []byte("ctx2")},
		{"diff_plaintext_same_context", []byte("test1"), []byte("ctx")},
		{"diff_plaintext_same_context2", []byte("test2"), []byte("ctx")},
	}
	
	ciphertexts := make(map[string][]byte)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := enc.EncryptDeterministic(tt.plaintext, key, tt.context)
			if err != nil {
				t.Fatalf("EncryptDeterministic() error = %v", err)
			}
			
			// Check that each combination produces unique ciphertext
			for name, existing := range ciphertexts {
				if bytes.Equal(ciphertext, existing) && name != tt.name {
					t.Errorf("EncryptDeterministic() produced same ciphertext for %v and %v", tt.name, name)
				}
			}
			
			ciphertexts[tt.name] = ciphertext
		})
	}
}

func TestDeterministicEncryptor_CrossCompatibility(t *testing.T) {
	// Test that deterministic encryptor can decrypt data from regular AES-GCM
	aesEnc := NewAESGCMEncryptor()
	detEnc := NewDeterministicEncryptor()
	
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	
	plaintext := []byte("cross compatibility test")
	
	// Encrypt with AES-GCM
	aesCiphertext, err := aesEnc.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AES Encrypt() error = %v", err)
	}
	
	// Decrypt with deterministic encryptor
	decrypted, err := detEnc.Decrypt(aesCiphertext, key)
	if err != nil {
		t.Fatalf("Deterministic Decrypt() error = %v", err)
	}
	
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Cross-decryption failed: got %v, want %v", decrypted, plaintext)
	}
}

func BenchmarkDeterministicEncryptor_Encrypt(b *testing.B) {
	enc := NewDeterministicEncryptor()
	key := make([]byte, 32)
	rand.Read(key)
	
	sizes := []int{32, 256, 1024, 4096}
	
	for _, size := range sizes {
		plaintext := make([]byte, size)
		rand.Read(plaintext)
		
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := enc.Encrypt(plaintext, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDeterministicEncryptor_EncryptDeterministic(b *testing.B) {
	enc := NewDeterministicEncryptor()
	key := make([]byte, 32)
	context := []byte("benchmark-context")
	rand.Read(key)
	
	sizes := []int{32, 256, 1024, 4096}
	
	for _, size := range sizes {
		plaintext := make([]byte, size)
		rand.Read(plaintext)
		
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				_, err := enc.EncryptDeterministic(plaintext, key, context)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}