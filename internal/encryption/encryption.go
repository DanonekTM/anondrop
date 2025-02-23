package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"secrets-share/internal/logger"

	"golang.org/x/crypto/pbkdf2"
)

const (
	keySize    = 32 // AES-256
	saltSize   = 16
	iterations = 10000
)

type Encryptor struct {
	serverKey []byte
}

func NewEncryptor(serverKey string) *Encryptor {
	return &Encryptor{
		serverKey: []byte(serverKey),
	}
}

func (e *Encryptor) Encrypt(data []byte, password string) ([]byte, error) {
	logger.Debug("Encrypting data", map[string]interface{}{
		"data_length": len(data),
		"password":    password,
	})
	// Generate a random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	logger.Debug("Generated salt", map[string]interface{}{
		"salt": fmt.Sprintf("%x", salt),
	})

	// Derive key from password and salt
	key := e.deriveKey(password, salt)
	logger.Debug("Derived key", map[string]interface{}{
		"key": fmt.Sprintf("%x", key),
	})

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine salt + nonce + ciphertext
	result := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func (e *Encryptor) Decrypt(encrypted []byte, password string) ([]byte, error) {
	logger.Debug("Decrypting data", map[string]interface{}{
		"data_length": len(encrypted),
		"password":    password,
	})
	if len(encrypted) < saltSize+12 { // 12 is the minimum nonce size for GCM
		return nil, fmt.Errorf("encrypted data is too short")
	}

	// Extract salt
	salt := encrypted[:saltSize]
	logger.Debug("Extracted salt", map[string]interface{}{
		"salt": fmt.Sprintf("%x", salt),
	})

	// Derive key from password and salt
	key := e.deriveKey(password, salt)
	logger.Debug("Derived key", map[string]interface{}{
		"key": fmt.Sprintf("%x", key),
	})

	// Create cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < saltSize+nonceSize {
		return nil, fmt.Errorf("encrypted data is too short")
	}

	// Extract nonce and ciphertext
	nonce := encrypted[saltSize : saltSize+nonceSize]
	ciphertext := encrypted[saltSize+nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (e *Encryptor) deriveKey(password string, salt []byte) []byte {
	// Combine password with server key for additional security
	combinedPassword := append([]byte(password), e.serverKey...)
	return pbkdf2.Key(combinedPassword, salt, iterations, keySize, sha256.New)
}

// EncodeToString encodes the encrypted data to a base64 string
func EncodeToString(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeString decodes the base64 string back to bytes
func DecodeString(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
