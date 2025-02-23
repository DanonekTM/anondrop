package models

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var (
	// CustomNameRegex defines the allowed characters for custom names (alphanumeric only)
	CustomNameRegex = regexp.MustCompile("^[a-zA-Z0-9]+$")
)

// ValidateCustomName checks if the custom name is valid
func ValidateCustomName(name string) error {
	if name == "" {
		return nil // Empty name is valid (optional field)
	}

	if !CustomNameRegex.MatchString(name) {
		return fmt.Errorf("custom name can only contain letters and numbers (A-Z, a-z, 0-9)")
	}

	return nil
}

type Secret struct {
	ID                 uuid.UUID  `json:"id"`
	CustomName         string     `json:"custom_name,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	IsBurnAfterReading bool       `json:"is_burn_after_reading"`
	EncryptedData      []byte     `json:"encrypted_data"` // Server-encrypted data
}

type EncryptedContent struct {
	Encrypted string `json:"encrypted" binding:"required"`
	Salt      string `json:"salt" binding:"required"`
	IV        string `json:"iv" binding:"required"`
}

type SecretInput struct {
	EncryptedContent   EncryptedContent `json:"encryptedContent" binding:"required"`
	CustomName         string           `json:"customName,omitempty"`
	ExpiresAt          *time.Time       `json:"expires_at,omitempty"`
	IsBurnAfterReading bool             `json:"isBurnAfterReading"`
	CaptchaToken       string           `json:"captchaToken" binding:"required"`
}

type SecretView struct {
	EncryptedContent EncryptedContent `json:"encryptedContent"`
	ExpiresAt        *time.Time       `json:"expires_at,omitempty"`
	MaxViews         *int             `json:"max_views,omitempty"`
	ViewCount        int              `json:"view_count"`
	CustomName       string           `json:"custom_name,omitempty"`
}

type SecretAccess struct {
	CaptchaToken string `json:"captchaToken" binding:"required"`
}

func NewSecret(input *SecretInput) *Secret {
	return &Secret{
		ID:                 uuid.New(),
		CustomName:         input.CustomName,
		CreatedAt:          time.Now(),
		ExpiresAt:          input.ExpiresAt,
		IsBurnAfterReading: input.IsBurnAfterReading,
	}
}

func (s *Secret) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}
