package encryption

import (
	"bytes"
	"secrets-share/internal/logger"
	"testing"
)

func setupTestLogger(t *testing.T) func() {
	// Create a test logger configuration
	cfg := &logger.Config{
		Enabled:        false, // Disable logging during tests
		ConsoleOutput:  false,
		Directory:      t.TempDir(),
		ArchiveDir:     t.TempDir(),
		RotationSizeMB: 10,
		RetentionDays:  7,
		Files: map[string]logger.FileConfig{
			"application": {Filename: "app.log"},
		},
	}

	// Initialize logger
	err := logger.Init(cfg, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Return cleanup function that does nothing since we don't need to clean up anymore
	return func() {}
}

func TestEncryption(t *testing.T) {
	cleanup := setupTestLogger(t)
	defer cleanup()

	testCases := []struct {
		name     string
		data     []byte
		password string
	}{
		{
			name:     "Simple text",
			data:     []byte("Hello, World!"),
			password: "test-password-123",
		},
		{
			name:     "Empty string",
			data:     []byte(""),
			password: "test-password-123",
		},
		{
			name:     "Long text",
			data:     bytes.Repeat([]byte("A"), 1000),
			password: "test-password-123",
		},
		{
			name:     "Special characters",
			data:     []byte("!@#$%^&*()_+-=[]{}|;:,.<>?"),
			password: "test-password-123",
		},
		{
			name:     "Unicode characters",
			data:     []byte("Hello, 世界! Привет, мир! 👋"),
			password: "test-password-123",
		},
	}

	encryptor := NewEncryptor("test-server-key")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test encryption
			encrypted, err := encryptor.Encrypt(tc.data, tc.password)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Debug logging
			t.Logf("Original data: %q", tc.data)
			t.Logf("Password: %q", tc.password)
			t.Logf("Encrypted data length: %d", len(encrypted))

			// Verify encrypted data is different from original
			if bytes.Equal(encrypted, tc.data) {
				t.Error("Encrypted data is identical to original data")
			}

			// Test decryption
			decrypted, err := encryptor.Decrypt(encrypted, tc.password)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Debug logging
			t.Logf("Decrypted data: %q", decrypted)

			// Verify decrypted data matches original
			if !bytes.Equal(decrypted, tc.data) {
				t.Error("Decrypted data does not match original data")
			}
		})
	}
}

func TestDecryptionWithWrongPassword(t *testing.T) {
	cleanup := setupTestLogger(t)
	defer cleanup()

	encryptor := NewEncryptor("test-server-key")
	data := []byte("Hello, World!")
	password := "correct-password"
	wrongPassword := "wrong-password"

	// Encrypt with correct password
	encrypted, err := encryptor.Encrypt(data, password)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Debug logging
	t.Logf("Original data: %q", data)
	t.Logf("Correct password: %q", password)
	t.Logf("Wrong password: %q", wrongPassword)
	t.Logf("Encrypted data length: %d", len(encrypted))

	// Try to decrypt with wrong password
	decrypted, err := encryptor.Decrypt(encrypted, wrongPassword)
	if err == nil {
		t.Error("Expected decryption to fail with wrong password, but it succeeded")
		t.Logf("Decrypted data with wrong password: %q", decrypted)
	} else {
		t.Logf("Decryption error: %v", err)
	}
}

func TestEncodeDecodeString(t *testing.T) {
	cleanup := setupTestLogger(t)
	defer cleanup()

	testData := []byte("Hello, World!")

	// Test encoding
	encoded := EncodeToString(testData)
	if encoded == "" {
		t.Error("Expected non-empty encoded string")
	}

	// Test decoding
	decoded, err := DecodeString(encoded)
	if err != nil {
		t.Fatalf("Decoding failed: %v", err)
	}

	if !bytes.Equal(decoded, testData) {
		t.Error("Decoded data does not match original data")
	}
}

func TestDecryptionWithInvalidData(t *testing.T) {
	cleanup := setupTestLogger(t)
	defer cleanup()

	encryptor := NewEncryptor("test-server-key")
	invalidData := []byte("invalid-data")
	password := "test-password"

	// Try to decrypt invalid data
	_, err := encryptor.Decrypt(invalidData, password)
	if err == nil {
		t.Error("Expected decryption to fail with invalid data, but it succeeded")
	}
}
