package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"secrets-share/internal/captcha"
	"secrets-share/internal/config"
	"secrets-share/internal/encryption"
	"secrets-share/internal/models"
	"secrets-share/internal/storage/file"

	"secrets-share/internal/logger"

	"github.com/google/uuid"
)

const testServerKey = "test-server-key-32-bytes-long-key!!"

// MockTurnstileClient is a mock implementation of the TurnstileVerifier interface
type MockTurnstileClient struct {
	mock.Mock
}

func (m *MockTurnstileClient) Verify(token, remoteIP string) (*captcha.TurnstileResponse, error) {
	args := m.Called(token, remoteIP)
	return args.Get(0).(*captcha.TurnstileResponse), args.Error(1)
}

func setupTestEnvironment(t *testing.T) (*gin.Engine, *SecretAPIHandler, *MockTurnstileClient, func()) {
	// Create a test logger configuration
	cfg := &logger.Config{
		Enabled:       false, // Disable logging during tests
		ConsoleOutput: false,
		Directory:     t.TempDir(),
		ArchiveDir:    t.TempDir(),
		Files: map[string]logger.FileConfig{
			"application": {Filename: "app.log"},
		},
	}

	// Initialize logger
	err := logger.Init(cfg, false)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a temporary directory for the test
	testDir, err := os.MkdirTemp("", "secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	testConfig := &config.Config{
		Security: config.SecurityConfig{
			EnableCaptcha:        true,
			ServerSideEncryption: true,
		},
		Secrets: config.SecretsConfig{
			MaxSizeBytes:         500,
			StoragePath:          testDir,
			MaxExpiryDays:        7,
			DefaultExpiryMinutes: 60,
			MaxCustomNameLength:  32,
		},
	}

	// Initialize the encryptor with a test key
	encryptor := encryption.NewEncryptor(testServerKey)

	fileStore, err := file.NewFileStore(testConfig.Secrets.StoragePath)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	mockTurnstileClient := new(MockTurnstileClient)

	handler := NewSecretAPIHandler(
		fileStore,
		nil,
		encryptor,
		mockTurnstileClient,
		testConfig,
	)

	router.POST("/api/secrets", handler.CreateSecret)
	router.POST("/api/secrets/:id", handler.GetSecret)
	router.POST("/api/secrets/name/:name", handler.GetSecretByName)

	cleanup := func() {
		os.RemoveAll(testDir)
	}

	return router, handler, mockTurnstileClient, cleanup
}

func TestCreateSecret(t *testing.T) {
	router, _, mockTurnstileClient, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Mock successful captcha verification
	mockTurnstileClient.On("Verify", mock.Anything, mock.Anything).Return(&captcha.TurnstileResponse{Success: true}, nil)

	t.Run("Create secret with valid data", func(t *testing.T) {
		// Create test data with base64 encoded values
		encryptedContent := models.EncryptedContent{
			Encrypted: base64.StdEncoding.EncodeToString([]byte("test-data")),
			Salt:      base64.StdEncoding.EncodeToString([]byte("test-salt")),
			IV:        base64.StdEncoding.EncodeToString([]byte("test-iv")),
		}

		reqBody := APICreateSecretRequest{
			EncryptedContent: encryptedContent,
			CustomName:       "test123", // Valid alphanumeric name
			CaptchaToken:     "valid-token",
		}

		jsonData, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response APISecretResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.ID)
	})

	t.Run("Create secret with invalid data", func(t *testing.T) {
		reqBody := APICreateSecretRequest{
			EncryptedContent: models.EncryptedContent{
				Encrypted: "",
				Salt:      "",
				IV:        "",
			},
			CustomName:   "test-invalid-name", // Contains invalid character
			CaptchaToken: "valid-token",
		}

		jsonData, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetSecret(t *testing.T) {
	router, handler, mockTurnstileClient, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Mock successful captcha verification
	mockTurnstileClient.On("Verify", mock.Anything, mock.Anything).Return(&captcha.TurnstileResponse{Success: true}, nil)

	// Create a test secret first
	secret := &models.Secret{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
	}

	// Prepare the encrypted content with base64 encoded values
	encryptedContent := models.EncryptedContent{
		Encrypted: base64.StdEncoding.EncodeToString([]byte("test-data")),
		Salt:      base64.StdEncoding.EncodeToString([]byte("test-salt")),
		IV:        base64.StdEncoding.EncodeToString([]byte("test-iv")),
	}

	// Combine the encrypted content
	combinedData := fmt.Sprintf("%s.%s.%s",
		encryptedContent.Encrypted,
		encryptedContent.Salt,
		encryptedContent.IV,
	)

	// Apply server-side encryption if enabled
	if handler.config.Security.ServerSideEncryption {
		encryptedData, err := handler.encryptor.Encrypt([]byte(combinedData), "")
		assert.NoError(t, err)
		secret.EncryptedData = []byte(encryption.EncodeToString(encryptedData))
	} else {
		secret.EncryptedData = []byte(combinedData)
	}

	err := handler.fileStore.Store(secret)
	assert.NoError(t, err)

	t.Run("Get secret with valid data", func(t *testing.T) {
		reqBody := APIViewSecretRequest{
			CaptchaToken: "valid-token",
		}

		jsonData, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/secrets/%s", secret.ID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response APISecretContentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, encryptedContent, response.EncryptedContent)
	})

	t.Run("Get non-existent secret", func(t *testing.T) {
		nonExistentID := uuid.New()
		reqBody := APIViewSecretRequest{
			CaptchaToken: "valid-token",
		}

		jsonData, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/secrets/%s", nonExistentID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestGetSecretByName(t *testing.T) {
	router, handler, mockTurnstileClient, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Mock successful captcha verification
	mockTurnstileClient.On("Verify", mock.Anything, mock.Anything).Return(&captcha.TurnstileResponse{Success: true}, nil)

	// Create a test secret first
	secret := &models.Secret{
		ID:         uuid.New(),
		CustomName: "test123", // Valid alphanumeric name
		CreatedAt:  time.Now(),
	}

	// Prepare the encrypted content with base64 encoded values
	encryptedContent := models.EncryptedContent{
		Encrypted: base64.StdEncoding.EncodeToString([]byte("test-data")),
		Salt:      base64.StdEncoding.EncodeToString([]byte("test-salt")),
		IV:        base64.StdEncoding.EncodeToString([]byte("test-iv")),
	}

	// Combine the encrypted content
	combinedData := fmt.Sprintf("%s.%s.%s",
		encryptedContent.Encrypted,
		encryptedContent.Salt,
		encryptedContent.IV,
	)

	// Apply server-side encryption if enabled
	if handler.config.Security.ServerSideEncryption {
		encryptedData, err := handler.encryptor.Encrypt([]byte(combinedData), "")
		assert.NoError(t, err)
		secret.EncryptedData = []byte(encryption.EncodeToString(encryptedData))
	} else {
		secret.EncryptedData = []byte(combinedData)
	}

	err := handler.fileStore.Store(secret)
	assert.NoError(t, err)

	t.Run("Get secret by name with valid data", func(t *testing.T) {
		reqBody := APIViewSecretRequest{
			CaptchaToken: "valid-token",
		}

		jsonData, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", fmt.Sprintf("/api/secrets/name/%s", secret.CustomName), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response APISecretContentResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, encryptedContent, response.EncryptedContent)
	})

	t.Run("Get non-existent secret by name", func(t *testing.T) {
		reqBody := APIViewSecretRequest{
			CaptchaToken: "valid-token",
		}

		jsonData, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/secrets/name/nonexistent", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestServerSideEncryption(t *testing.T) {
	router, handler, mockTurnstileClient, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Mock successful captcha verification
	mockTurnstileClient.On("Verify", mock.Anything, mock.Anything).Return(&captcha.TurnstileResponse{Success: true}, nil)

	tests := []struct {
		name                 string
		serverSideEncryption bool
		wantEncrypted        bool
	}{
		{
			name:                 "With server-side encryption",
			serverSideEncryption: true,
			wantEncrypted:        true,
		},
		{
			name:                 "Without server-side encryption",
			serverSideEncryption: false,
			wantEncrypted:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set encryption mode for this test
			handler.config.Security.ServerSideEncryption = tt.serverSideEncryption

			// Create test data
			originalData := "test-secret-data"
			encryptedContent := models.EncryptedContent{
				Encrypted: base64.StdEncoding.EncodeToString([]byte(originalData)),
				Salt:      base64.StdEncoding.EncodeToString([]byte("test-salt")),
				IV:        base64.StdEncoding.EncodeToString([]byte("test-iv")),
			}

			reqBody := APICreateSecretRequest{
				EncryptedContent: encryptedContent,
				CaptchaToken:     "valid-token",
			}

			jsonData, err := json.Marshal(reqBody)
			assert.NoError(t, err)

			// Create the secret
			req := httptest.NewRequest("POST", "/api/secrets", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var response APISecretResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.NotEmpty(t, response.ID)

			// Verify the stored secret encryption
			secret, err := handler.fileStore.Get(response.ID)
			assert.NoError(t, err)
			assert.NotNil(t, secret)

			// Check if data is encrypted as expected
			combinedOriginal := fmt.Sprintf("%s.%s.%s",
				encryptedContent.Encrypted,
				encryptedContent.Salt,
				encryptedContent.IV,
			)

			if tt.wantEncrypted {
				assert.NotEqual(t, combinedOriginal, string(secret.EncryptedData), "Data should be server-side encrypted")
			} else {
				assert.Equal(t, combinedOriginal, string(secret.EncryptedData), "Data should not be server-side encrypted")
			}

			// Verify the secret can be retrieved and decrypted
			viewReqBody := APIViewSecretRequest{
				CaptchaToken: "valid-token",
			}

			jsonData, err = json.Marshal(viewReqBody)
			assert.NoError(t, err)

			req = httptest.NewRequest("POST", fmt.Sprintf("/api/secrets/%s", response.ID), bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var viewResponse APISecretContentResponse
			err = json.Unmarshal(w.Body.Bytes(), &viewResponse)
			assert.NoError(t, err)

			// Verify the content matches the original regardless of server-side encryption
			assert.Equal(t, encryptedContent.Encrypted, viewResponse.EncryptedContent.Encrypted)
			assert.Equal(t, encryptedContent.Salt, viewResponse.EncryptedContent.Salt)
			assert.Equal(t, encryptedContent.IV, viewResponse.EncryptedContent.IV)
		})
	}
}
