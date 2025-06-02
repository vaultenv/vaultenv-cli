package encryption

import (
	"strings"
	"testing"
)

func TestChaChaEncryptor_Algorithm(t *testing.T) {
	enc := NewChaChaEncryptor()
	if got := enc.Algorithm(); got != "chacha20-poly1305" {
		t.Errorf("Algorithm() = %v, want %v", got, "chacha20-poly1305")
	}
}

func TestChaChaEncryptor_NotImplemented(t *testing.T) {
	enc := NewChaChaEncryptor()
	key := make([]byte, 32)

	// Test all methods return not implemented error
	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "GenerateSalt",
			fn: func() error {
				_, err := enc.GenerateSalt()
				return err
			},
		},
		{
			name: "Encrypt",
			fn: func() error {
				_, err := enc.Encrypt([]byte("test"), key)
				return err
			},
		},
		{
			name: "Decrypt",
			fn: func() error {
				_, err := enc.Decrypt([]byte("test"), key)
				return err
			},
		},
		{
			name: "EncryptString",
			fn: func() error {
				_, err := enc.EncryptString("test", key)
				return err
			},
		},
		{
			name: "DecryptString",
			fn: func() error {
				_, err := enc.DecryptString("test", key)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
				t.Errorf("%s() error = %v, want not implemented error", tt.name, err)
			}
		})
	}
}

func TestChaChaEncryptor_GenerateKey(t *testing.T) {
	enc := NewChaChaEncryptor()

	// GenerateKey returns nil (stub implementation)
	key := enc.GenerateKey("password", []byte("salt"))
	if key != nil {
		t.Errorf("GenerateKey() = %v, want nil", key)
	}
}
