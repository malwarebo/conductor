package security

import (
	"testing"
)

func TestEncryptionManager_EncryptDecrypt(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	manager, err := CreateEncryptionManager(key)
	if err != nil {
		t.Fatalf("CreateEncryptionManager() error = %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "Simple text",
			plaintext: "Hello, World!",
		},
		{
			name:      "JSON data",
			plaintext: `{"user":"test","amount":1000}`,
		},
		{
			name:      "Empty string",
			plaintext: "",
		},
		{
			name:      "Long text",
			plaintext: "This is a very long text that should be encrypted and decrypted successfully without any data loss or corruption.",
		},
		{
			name:      "Special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := manager.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			if len(encrypted) == 0 {
				t.Error("Encrypt() returned empty string")
			}
			if encrypted == tt.plaintext {
				t.Error("Encrypt() returned same as plaintext")
			}

			decrypted, err := manager.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptionManager_DifferentKeys(t *testing.T) {
	plaintext := "sensitive data"

	manager1, err := CreateEncryptionManager([]byte("11111111111111111111111111111111"))
	if err != nil {
		t.Fatalf("CreateEncryptionManager() error = %v", err)
	}
	manager2, err := CreateEncryptionManager([]byte("22222222222222222222222222222222"))
	if err != nil {
		t.Fatalf("CreateEncryptionManager() error = %v", err)
	}

	encrypted, err := manager1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = manager2.Decrypt(encrypted)
	if err == nil {
		t.Error("Decrypt() expected error with different key")
	}
}

func TestEncryptionManager_InvalidCiphertext(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	manager, err := CreateEncryptionManager(key)
	if err != nil {
		t.Fatalf("CreateEncryptionManager() error = %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "Empty string",
			ciphertext: "",
		},
		{
			name:       "Too short",
			ciphertext: "short",
		},
		{
			name:       "Invalid base64",
			ciphertext: "not-valid-base64!@#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Decrypt() expected error for invalid ciphertext")
			}
		})
	}
}

func TestEncryptionManager_UniqueEncryption(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	manager, err := CreateEncryptionManager(key)
	if err != nil {
		t.Fatalf("CreateEncryptionManager() error = %v", err)
	}
	plaintext := "same text"

	encrypted1, err := manager.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	encrypted2, err := manager.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if encrypted1 == encrypted2 {
		t.Error("Encrypt() should produce different ciphertexts for same plaintext")
	}

	decrypted1, err := manager.Decrypt(encrypted1)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted1 != plaintext {
		t.Errorf("Decrypt() = %v, want %v", decrypted1, plaintext)
	}

	decrypted2, err := manager.Decrypt(encrypted2)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted2 != plaintext {
		t.Errorf("Decrypt() = %v, want %v", decrypted2, plaintext)
	}
}
