package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestAESGCMEncryptor(t *testing.T) {
	encryptor := NewAESGCMEncryptor()

	t.Run("Algorithm", func(t *testing.T) {
		if got := encryptor.Algorithm(); got != "aes-gcm-256" {
			t.Errorf("Algorithm() = %v, want %v", got, "aes-gcm-256")
		}
	})

	t.Run("GenerateSalt", func(t *testing.T) {
		salt1, err := encryptor.GenerateSalt()
		if err != nil {
			t.Fatalf("GenerateSalt() error = %v", err)
		}
		if len(salt1) != 32 {
			t.Errorf("GenerateSalt() length = %v, want 32", len(salt1))
		}

		salt2, err := encryptor.GenerateSalt()
		if err != nil {
			t.Fatalf("GenerateSalt() error = %v", err)
		}
		if bytes.Equal(salt1, salt2) {
			t.Error("GenerateSalt() should generate unique salts")
		}
	})

	t.Run("GenerateKey", func(t *testing.T) {
		salt, _ := encryptor.GenerateSalt()
		password := "testpassword123"

		key1 := encryptor.GenerateKey(password, salt)
		if len(key1) != 32 {
			t.Errorf("GenerateKey() length = %v, want 32", len(key1))
		}

		// Same password and salt should generate same key
		key2 := encryptor.GenerateKey(password, salt)
		if !bytes.Equal(key1, key2) {
			t.Error("GenerateKey() should be deterministic")
		}

		// Different password should generate different key
		key3 := encryptor.GenerateKey("different", salt)
		if bytes.Equal(key1, key3) {
			t.Error("GenerateKey() should generate different keys for different passwords")
		}

		// Different salt should generate different key
		salt2, _ := encryptor.GenerateSalt()
		key4 := encryptor.GenerateKey(password, salt2)
		if bytes.Equal(key1, key4) {
			t.Error("GenerateKey() should generate different keys for different salts")
		}
	})

	t.Run("EncryptDecrypt", func(t *testing.T) {
		key := make([]byte, 32)
		rand.Read(key)

		testCases := []struct {
			name      string
			plaintext []byte
		}{
			{"empty", []byte{}},
			{"short", []byte("hello")},
			{"medium", []byte("This is a medium length test message")},
			{"long", make([]byte, 1024)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Encrypt
				ciphertext, err := encryptor.Encrypt(tc.plaintext, key)
				if err != nil {
					t.Fatalf("Encrypt() error = %v", err)
				}

				// Ciphertext should be different from plaintext
				if len(tc.plaintext) > 0 && bytes.Equal(ciphertext, tc.plaintext) {
					t.Error("Encrypt() ciphertext should differ from plaintext")
				}

				// Decrypt
				decrypted, err := encryptor.Decrypt(ciphertext, key)
				if err != nil {
					t.Fatalf("Decrypt() error = %v", err)
				}

				// Decrypted should match original
				if !bytes.Equal(decrypted, tc.plaintext) {
					t.Error("Decrypt() did not return original plaintext")
				}
			})
		}
	})

	t.Run("EncryptWithInvalidKey", func(t *testing.T) {
		plaintext := []byte("test")

		// Too short
		_, err := encryptor.Encrypt(plaintext, []byte("short"))
		if err != ErrInvalidKey {
			t.Errorf("Encrypt() with short key error = %v, want %v", err, ErrInvalidKey)
		}

		// Too long
		_, err = encryptor.Encrypt(plaintext, make([]byte, 33))
		if err != ErrInvalidKey {
			t.Errorf("Encrypt() with long key error = %v, want %v", err, ErrInvalidKey)
		}
	})

	t.Run("DecryptWithInvalidKey", func(t *testing.T) {
		key := make([]byte, 32)
		rand.Read(key)
		plaintext := []byte("test")

		ciphertext, _ := encryptor.Encrypt(plaintext, key)

		// Wrong key length
		_, err := encryptor.Decrypt(ciphertext, []byte("short"))
		if err != ErrInvalidKey {
			t.Errorf("Decrypt() with short key error = %v, want %v", err, ErrInvalidKey)
		}
	})

	t.Run("DecryptWithInvalidData", func(t *testing.T) {
		key := make([]byte, 32)
		rand.Read(key)

		// Too short data
		_, err := encryptor.Decrypt([]byte("short"), key)
		if err != ErrInvalidData {
			t.Errorf("Decrypt() with short data error = %v, want %v", err, ErrInvalidData)
		}

		// Corrupted data
		ciphertext := make([]byte, 50)
		rand.Read(ciphertext)
		_, err = encryptor.Decrypt(ciphertext, key)
		if err != ErrDecryptionFailed {
			t.Errorf("Decrypt() with corrupted data error = %v, want %v", err, ErrDecryptionFailed)
		}
	})

	t.Run("StringHelpers", func(t *testing.T) {
		key := make([]byte, 32)
		rand.Read(key)
		plaintext := "Hello, World! 🌍"

		// Encrypt string
		encrypted, err := encryptor.EncryptString(plaintext, key)
		if err != nil {
			t.Fatalf("EncryptString() error = %v", err)
		}

		// Should be base64
		if len(encrypted) == 0 {
			t.Error("EncryptString() returned empty string")
		}

		// Decrypt string
		decrypted, err := encryptor.DecryptString(encrypted, key)
		if err != nil {
			t.Fatalf("DecryptString() error = %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("DecryptString() = %v, want %v", decrypted, plaintext)
		}

		// Invalid base64
		_, err = encryptor.DecryptString("not-base64!", key)
		if err == nil {
			t.Error("DecryptString() should fail with invalid base64")
		}
	})
}