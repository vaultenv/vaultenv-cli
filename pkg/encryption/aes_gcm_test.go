package encryption

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestAESGCMEncryptor_Algorithm(t *testing.T) {
	enc := NewAESGCMEncryptor()
	if got := enc.Algorithm(); got != "aes-gcm-256" {
		t.Errorf("Algorithm() = %v, want %v", got, "aes-gcm-256")
	}
}

func TestAESGCMEncryptor_GenerateSalt(t *testing.T) {
	enc := NewAESGCMEncryptor()

	// Test salt generation
	salt1, err := enc.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if len(salt1) != 32 {
		t.Errorf("GenerateSalt() length = %v, want 32", len(salt1))
	}

	// Test that salts are unique
	salt2, err := enc.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("GenerateSalt() generated identical salts")
	}
}

func TestAESGCMEncryptor_GenerateKey(t *testing.T) {
	enc := NewAESGCMEncryptor()

	salt, _ := enc.GenerateSalt()
	password := "test-password-123"

	// Test key generation
	key1 := enc.GenerateKey(password, salt)
	if len(key1) != 32 {
		t.Errorf("GenerateKey() length = %v, want 32", len(key1))
	}

	// Test deterministic key generation
	key2 := enc.GenerateKey(password, salt)
	if !bytes.Equal(key1, key2) {
		t.Error("GenerateKey() not deterministic for same password and salt")
	}

	// Test different password produces different key
	key3 := enc.GenerateKey("different-password", salt)
	if bytes.Equal(key1, key3) {
		t.Error("GenerateKey() produced same key for different passwords")
	}

	// Test different salt produces different key
	salt2, _ := enc.GenerateSalt()
	key4 := enc.GenerateKey(password, salt2)
	if bytes.Equal(key1, key4) {
		t.Error("GenerateKey() produced same key for different salts")
	}
}

func TestAESGCMEncryptor_EncryptDecrypt(t *testing.T) {
	enc := NewAESGCMEncryptor()

	// Generate test key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello")},
		{"medium", []byte("The quick brown fox jumps over the lazy dog")},
		{"large", bytes.Repeat([]byte("a"), 10000)},
		{"binary", []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe, 0xfd}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Ciphertext should be different from plaintext
			if bytes.Equal(ciphertext, tt.plaintext) && len(tt.plaintext) > 0 {
				t.Error("Encrypt() ciphertext equals plaintext")
			}

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify decryption
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestAESGCMEncryptor_EncryptDecryptString(t *testing.T) {
	enc := NewAESGCMEncryptor()

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
		{"multiline", "line1\nline2\rline3\r\nline4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.EncryptString(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptString() error = %v", err)
			}

			// Verify base64 encoding
			if _, err := base64.StdEncoding.DecodeString(ciphertext); err != nil {
				t.Errorf("EncryptString() produced invalid base64: %v", err)
			}

			// Decrypt
			decrypted, err := enc.DecryptString(ciphertext, key)
			if err != nil {
				t.Fatalf("DecryptString() error = %v", err)
			}

			// Verify decryption
			if decrypted != tt.plaintext {
				t.Errorf("DecryptString() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestAESGCMEncryptor_InvalidKey(t *testing.T) {
	enc := NewAESGCMEncryptor()
	plaintext := []byte("test data")

	tests := []struct {
		name string
		key  []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"too_short", make([]byte, 16)},
		{"too_long", make([]byte, 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption with invalid key
			_, err := enc.Encrypt(plaintext, tt.key)
			if err != ErrInvalidKey {
				t.Errorf("Encrypt() error = %v, want %v", err, ErrInvalidKey)
			}

			// Test decryption with invalid key
			_, err = enc.Decrypt(plaintext, tt.key)
			if err != ErrInvalidKey {
				t.Errorf("Decrypt() error = %v, want %v", err, ErrInvalidKey)
			}
		})
	}
}

func TestAESGCMEncryptor_InvalidCiphertext(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)

	tests := []struct {
		name       string
		ciphertext []byte
		wantErr    error
	}{
		{"empty", []byte{}, ErrInvalidData},
		{"too_short", make([]byte, 5), ErrInvalidData},
		{"corrupted", make([]byte, 50), ErrDecryptionFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext, key)
			if err != tt.wantErr {
				t.Errorf("Decrypt() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAESGCMEncryptor_DecryptString_InvalidBase64(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)

	// Test invalid base64
	_, err := enc.DecryptString("not-valid-base64!", key)
	if err == nil || !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("DecryptString() error = %v, want invalid base64 error", err)
	}
}

func TestAESGCMEncryptor_EncryptWithNonce(t *testing.T) {
	enc := NewAESGCMEncryptor()

	// Generate test key and nonce
	key := make([]byte, 32)
	nonce := make([]byte, 12) // GCM standard nonce size
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("Failed to generate test nonce: %v", err)
	}

	plaintext := []byte("test data")

	// Test encryption with nonce
	ciphertext1, err := enc.EncryptWithNonce(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("EncryptWithNonce() error = %v", err)
	}

	// Test deterministic encryption with same nonce
	ciphertext2, err := enc.EncryptWithNonce(plaintext, key, nonce)
	if err != nil {
		t.Fatalf("EncryptWithNonce() error = %v", err)
	}

	if !bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("EncryptWithNonce() not deterministic with same nonce")
	}

	// Test decryption
	decrypted, err := enc.Decrypt(ciphertext1, key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() = %v, want %v", decrypted, plaintext)
	}

	// Test invalid nonce size
	invalidNonce := make([]byte, 16)
	_, err = enc.EncryptWithNonce(plaintext, key, invalidNonce)
	if err == nil || !strings.Contains(err.Error(), "invalid nonce size") {
		t.Errorf("EncryptWithNonce() error = %v, want invalid nonce size error", err)
	}
}

func TestAESGCMEncryptor_UniqueEncryption(t *testing.T) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	plaintext := []byte("test data")

	// Encrypt same data multiple times
	ciphertext1, _ := enc.Encrypt(plaintext, key)
	ciphertext2, _ := enc.Encrypt(plaintext, key)

	// Each encryption should produce different ciphertext (due to random nonce)
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("Encrypt() produced identical ciphertexts for same plaintext")
	}
}

func BenchmarkAESGCMEncryptor_Encrypt(b *testing.B) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	rand.Read(key)

	sizes := []int{32, 256, 1024, 4096, 16384}

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

func BenchmarkAESGCMEncryptor_Decrypt(b *testing.B) {
	enc := NewAESGCMEncryptor()
	key := make([]byte, 32)
	rand.Read(key)

	sizes := []int{32, 256, 1024, 4096, 16384}

	for _, size := range sizes {
		plaintext := make([]byte, size)
		rand.Read(plaintext)
		ciphertext, _ := enc.Encrypt(plaintext, key)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := enc.Decrypt(ciphertext, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkAESGCMEncryptor_GenerateKey(b *testing.B) {
	enc := NewAESGCMEncryptor()
	salt, _ := enc.GenerateSalt()
	password := "benchmark-password-123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = enc.GenerateKey(password, salt)
	}
}
