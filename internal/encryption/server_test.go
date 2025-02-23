package encryption

import (
	"bytes"
	"testing"
)

func TestServerSideEncryption(t *testing.T) {
	cleanup := setupTestLogger(t)
	defer cleanup()

	// Test with different server keys
	testCases := []struct {
		name      string
		serverKey string
		data      []byte
	}{
		{
			name:      "Normal server key",
			serverKey: "test-server-key-32-bytes-long-key!!",
			data:      []byte("Hello, World!"),
		},
		{
			name:      "Empty server key",
			serverKey: "",
			data:      []byte("Hello, World!"),
		},
		{
			name:      "Different server keys should produce different results",
			serverKey: "different-server-key-32-bytes-!!!!!",
			data:      []byte("Hello, World!"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encryptor := NewEncryptor(tc.serverKey)

			// Test server-side encryption (empty password)
			encrypted1, err := encryptor.Encrypt(tc.data, "")
			if err != nil {
				t.Fatalf("Server-side encryption failed: %v", err)
			}

			// Verify the data is actually encrypted
			if bytes.Equal(encrypted1, tc.data) {
				t.Error("Encrypted data is identical to original data")
			}

			// Test decryption with the same server key
			decrypted1, err := encryptor.Decrypt(encrypted1, "")
			if err != nil {
				t.Fatalf("Server-side decryption failed: %v", err)
			}

			// Verify decrypted data matches original
			if !bytes.Equal(decrypted1, tc.data) {
				t.Error("Decrypted data does not match original data")
			}

			// Create a second encryptor with a different server key
			differentKey := "completely-different-server-key!!!!"
			if differentKey == tc.serverKey {
				differentKey = "another-completely-different-key!!!"
			}
			encryptor2 := NewEncryptor(differentKey)

			// Try to decrypt with different server key
			_, err = encryptor2.Decrypt(encrypted1, "")
			if err == nil {
				t.Error("Expected decryption to fail with different server key, but it succeeded")
			}

			// Verify that encryption produces different results with different server keys
			encrypted2, err := encryptor2.Encrypt(tc.data, "")
			if err != nil {
				t.Fatalf("Second encryption failed: %v", err)
			}

			if bytes.Equal(encrypted1, encrypted2) {
				t.Error("Encryptions with different server keys produced identical results")
			}
		})
	}
}

func TestServerAndClientEncryption(t *testing.T) {
	cleanup := setupTestLogger(t)
	defer cleanup()

	serverKey := "test-server-key-32-bytes-long-key!!"
	clientPassword := "test-client-password"
	data := []byte("Hello, World!")

	encryptor := NewEncryptor(serverKey)

	// First, encrypt with client password
	clientEncrypted, err := encryptor.Encrypt(data, clientPassword)
	if err != nil {
		t.Fatalf("Client-side encryption failed: %v", err)
	}

	// Then, encrypt the client-encrypted data with server key
	serverEncrypted, err := encryptor.Encrypt(clientEncrypted, "")
	if err != nil {
		t.Fatalf("Server-side encryption failed: %v", err)
	}

	// Decrypt in reverse order
	serverDecrypted, err := encryptor.Decrypt(serverEncrypted, "")
	if err != nil {
		t.Fatalf("Server-side decryption failed: %v", err)
	}

	clientDecrypted, err := encryptor.Decrypt(serverDecrypted, clientPassword)
	if err != nil {
		t.Fatalf("Client-side decryption failed: %v", err)
	}

	// Verify the final result matches the original data
	if !bytes.Equal(clientDecrypted, data) {
		t.Error("Final decrypted data does not match original data")
	}
}
