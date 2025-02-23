package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"secrets-share/internal/captcha"
	"secrets-share/internal/config"
	"secrets-share/internal/encryption"
	"secrets-share/internal/logger"
	"secrets-share/internal/models"
	"secrets-share/internal/storage/file"
	"secrets-share/internal/storage/redis"
)

var (
	// UUID regex pattern
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// SecretAPIHandler handles HTTP requests for secrets
type SecretAPIHandler struct {
	fileStore     *file.FileStore
	redisStore    *redis.RedisStore
	encryptor     *encryption.Encryptor
	captchaClient captcha.TurnstileVerifier
	config        *config.Config
}

// NewSecretAPIHandler creates a new SecretAPIHandler
func NewSecretAPIHandler(
	fileStore *file.FileStore,
	redisStore *redis.RedisStore,
	encryptor *encryption.Encryptor,
	captchaClient captcha.TurnstileVerifier,
	config *config.Config,
) *SecretAPIHandler {
	return &SecretAPIHandler{
		fileStore:     fileStore,
		redisStore:    redisStore,
		encryptor:     encryptor,
		captchaClient: captchaClient,
		config:        config,
	}
}

// APISecretResponse represents a secret in responses
type APISecretResponse struct {
	ID string `json:"id"`
}

// APISecretContentResponse represents a secret's content in responses
type APISecretContentResponse struct {
	EncryptedContent   models.EncryptedContent `json:"encryptedContent"`
	ExpiresAt          *time.Time              `json:"expiresAt,omitempty"`
	IsBurnAfterReading bool                    `json:"isBurnAfterReading"`
}

// APICreateSecretRequest represents a request to create a secret
type APICreateSecretRequest struct {
	EncryptedContent models.EncryptedContent `json:"encryptedContent" binding:"required"`
	CustomName       string                  `json:"customName,omitempty"`
	ExpiresAt        *time.Time              `json:"expiresAt,omitempty"`
	MaxViews         *int                    `json:"maxViews,omitempty"`
	CaptchaToken     string                  `json:"captchaToken" binding:"required"`
}

// APIViewSecretRequest represents a request to view a secret
type APIViewSecretRequest struct {
	CaptchaToken string `json:"captchaToken" binding:"required"`
}

// CreateSecret handles the creation of a new secret
func (h *SecretAPIHandler) CreateSecret(c *gin.Context) {
	var req APICreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate custom name if provided
	if err := models.ValidateCustomName(req.CustomName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify captcha token
	if h.config.Security.EnableCaptcha {
		result, err := h.captchaClient.Verify(req.CaptchaToken, c.ClientIP())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify captcha"})
			return
		}
		if !result.Success {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid captcha"})
			return
		}
	}

	// Create secret input
	input := &models.SecretInput{
		EncryptedContent:   req.EncryptedContent,
		CustomName:         req.CustomName,
		ExpiresAt:          req.ExpiresAt,
		IsBurnAfterReading: req.MaxViews != nil && *req.MaxViews == 1,
		CaptchaToken:       req.CaptchaToken,
	}

	// Create secret model
	secret := models.NewSecret(input)

	// Handle expiry time based on whether it's a burn-after-reading secret
	if input.IsBurnAfterReading {
		// Burn-after-reading secrets have no expiry time
		secret.ExpiresAt = nil
	} else {
		// Calculate allowed expiry times
		now := time.Now()
		allowedExpiryTimes := map[time.Duration]bool{
			10 * time.Minute:   true,
			30 * time.Minute:   true,
			1 * time.Hour:      true,
			24 * time.Hour:     true,
			7 * 24 * time.Hour: true,
		}

		if secret.ExpiresAt == nil {
			// Set default expiry (10 minutes)
			defaultExpiry := now.Add(10 * time.Minute)
			secret.ExpiresAt = &defaultExpiry
		} else {
			// Calculate the duration between now and the requested expiry time
			duration := secret.ExpiresAt.Sub(now)

			// Check if the duration matches any of the allowed options
			isAllowedDuration := false
			for allowedDuration := range allowedExpiryTimes {
				// Allow for 1-second precision to handle slight timing differences
				if duration >= allowedDuration-time.Second && duration <= allowedDuration+time.Second {
					isAllowedDuration = true
					// Normalize the expiry time to exact duration
					exactExpiry := now.Add(allowedDuration)
					secret.ExpiresAt = &exactExpiry
					break
				}
			}

			if !isAllowedDuration {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiry time. Allowed values are: 10 minutes, 30 minutes, 1 hour, 1 day, or 7 days"})
				return
			}
		}
	}

	// Combine all client-side encrypted data into a single string
	combinedData := fmt.Sprintf("%s.%s.%s",
		input.EncryptedContent.Encrypted,
		input.EncryptedContent.Salt,
		input.EncryptedContent.IV,
	)

	// Server-side encryption of the combined data
	if h.config.Security.ServerSideEncryption {
		encryptedData, err := h.encryptor.Encrypt([]byte(combinedData), "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt data"})
			return
		}
		secret.EncryptedData = []byte(encryption.EncodeToString(encryptedData))
	} else {
		secret.EncryptedData = []byte(combinedData)
	}

	// Store the secret
	if err := h.fileStore.Store(secret); err != nil {
		if strings.Contains(err.Error(), "already taken") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store secret"})
		return
	}

	c.JSON(http.StatusOK, APISecretResponse{ID: secret.ID.String()})
}

func (h *SecretAPIHandler) decryptAndPrepareSecret(secret *models.Secret) (*APISecretContentResponse, error) {
	var combinedData string

	if h.config.Security.ServerSideEncryption {
		// Decode the base64-encoded encrypted data
		encryptedBytes, err := encryption.DecodeString(string(secret.EncryptedData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
		}

		// Decrypt using server key
		decryptedBytes, err := h.encryptor.Decrypt(encryptedBytes, "")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt server-side encryption: %w", err)
		}
		combinedData = string(decryptedBytes)
	} else {
		combinedData = string(secret.EncryptedData)
	}

	// Split the combined data into its components
	parts := strings.Split(combinedData, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid data format")
	}

	return &APISecretContentResponse{
		EncryptedContent: models.EncryptedContent{
			Encrypted: parts[0],
			Salt:      parts[1],
			IV:        parts[2],
		},
		ExpiresAt:          secret.ExpiresAt,
		IsBurnAfterReading: secret.IsBurnAfterReading,
	}, nil
}

// GetSecret retrieves a secret by ID
func (h *SecretAPIHandler) GetSecret(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing secret ID"})
		return
	}

	// Validate UUID format
	if !uuidPattern.MatchString(strings.ToLower(id)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret ID format"})
		return
	}

	var req APIViewSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Verify captcha
	resp, err := h.captchaClient.Verify(req.CaptchaToken, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify captcha"})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid captcha"})
		return
	}

	// Get secret
	secret, err := h.fileStore.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get secret"})
		return
	}
	if secret == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Secret not found"})
		return
	}

	// Check if secret is expired
	if secret.IsExpired() {
		if err := h.fileStore.Delete(id); err != nil {
			logger.Error("Failed to delete expired secret", map[string]interface{}{
				"error": err.Error(),
				"id":    id,
			})
		}
		c.JSON(http.StatusGone, gin.H{"error": "Secret has expired"})
		return
	}

	// Prepare the response
	response, err := h.decryptAndPrepareSecret(secret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete the secret if it's burn-after-reading
	if secret.IsBurnAfterReading {
		if err := h.fileStore.Delete(secret.ID.String()); err != nil {
			logger.Error("Failed to delete burn-after-reading secret", map[string]interface{}{
				"error": err.Error(),
				"id":    secret.ID,
			})
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetSecretByName retrieves a secret by custom name
func (h *SecretAPIHandler) GetSecretByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing secret name"})
		return
	}

	// Validate name format (alphanumeric only)
	if !models.CustomNameRegex.MatchString(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Secret name can only contain letters and numbers (A-Z, a-z, 0-9)"})
		return
	}

	var req APIViewSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Verify captcha
	resp, err := h.captchaClient.Verify(req.CaptchaToken, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify captcha"})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid captcha"})
		return
	}

	// Get secret by name
	secret, err := h.fileStore.GetByCustomName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get secret"})
		return
	}
	if secret == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Secret not found"})
		return
	}

	// Check if secret is expired
	if secret.IsExpired() {
		if err := h.fileStore.Delete(secret.ID.String()); err != nil {
			logger.Error("Failed to delete expired secret", map[string]interface{}{
				"error": err.Error(),
				"id":    secret.ID,
			})
		}
		c.JSON(http.StatusGone, gin.H{"error": "Secret has expired"})
		return
	}

	// Prepare the response
	response, err := h.decryptAndPrepareSecret(secret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete the secret if it's burn-after-reading
	if secret.IsBurnAfterReading {
		if err := h.fileStore.Delete(secret.ID.String()); err != nil {
			logger.Error("Failed to delete burn-after-reading secret", map[string]interface{}{
				"error": err.Error(),
				"id":    secret.ID,
			})
		}
	}

	c.JSON(http.StatusOK, response)
}
