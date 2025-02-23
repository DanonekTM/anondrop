package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"secrets-share/internal/logger"
	"secrets-share/internal/models"
)

type FileStore struct {
	basePath string
	mu       sync.RWMutex
	// Add metrics
	cleanupStats struct {
		lastRun        time.Time
		secretsCleaned int
		errors         int
	}
}

// CleanupStats represents cleanup operation statistics
type CleanupStats struct {
	LastRun        time.Time
	SecretsCleaned int
	Errors         int
}

// GetCleanupStats returns the current cleanup statistics
func (s *FileStore) GetCleanupStats() CleanupStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CleanupStats{
		LastRun:        s.cleanupStats.lastRun,
		SecretsCleaned: s.cleanupStats.secretsCleaned,
		Errors:         s.cleanupStats.errors,
	}
}

func NewFileStore(basePath string) (*FileStore, error) {
	// Ensure the base path exists
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStore{
		basePath: basePath,
	}, nil
}

func (s *FileStore) Store(secret *models.Secret) error {
	// Check if custom name is taken before acquiring write lock
	if secret.CustomName != "" {
		taken, err := s.IsCustomNameTaken(secret.CustomName)
		if err != nil {
			return fmt.Errorf("failed to check custom name: %w", err)
		}
		if taken {
			// Check if we're updating the same secret
			existingSecret, err := s.GetByCustomName(secret.CustomName)
			if err != nil {
				return fmt.Errorf("failed to get existing secret: %w", err)
			}
			if existingSecret.ID != secret.ID {
				return fmt.Errorf("custom name %q is already taken", secret.CustomName)
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create file path
	filePath := filepath.Join(s.basePath, secret.ID.String()+".json")

	// Marshal secret to JSON
	data, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write secret file: %w", err)
	}

	return nil
}

func (s *FileStore) Get(id string) (*models.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create file path
	filePath := filepath.Join(s.basePath, id+".json")

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read secret file: %w", err)
	}

	// Unmarshal JSON
	var secret models.Secret
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return &secret, nil
}

func (s *FileStore) GetByCustomName(name string) (*models.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// List all files in the directory
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Search for a secret with matching custom name
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var secret models.Secret
		if err := json.Unmarshal(data, &secret); err != nil {
			continue
		}

		if secret.CustomName == name {
			return &secret, nil
		}
	}

	return nil, nil
}

func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.basePath, id+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete secret file: %w", err)
	}

	return nil
}

func (fs *FileStore) CleanExpired() error {
	files, err := ioutil.ReadDir(fs.basePath)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	var deletedCount int
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(fs.basePath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			logger.Error("Failed to read secret file", map[string]interface{}{
				"file":  file.Name(),
				"error": err.Error(),
			})
			continue
		}

		var secret models.Secret
		if err := json.Unmarshal(data, &secret); err != nil {
			logger.Error("Failed to unmarshal secret", map[string]interface{}{
				"file":  file.Name(),
				"error": err.Error(),
			})
			continue
		}

		if secret.IsExpired() {
			if err := os.Remove(filePath); err != nil {
				logger.Error("Failed to delete expired secret", map[string]interface{}{
					"file":  filePath,
					"error": err.Error(),
				})
				continue
			}
			deletedCount++
		}
	}

	logger.Debug("Cleaned up expired secrets", map[string]interface{}{
		"deleted_count": deletedCount,
	})

	fs.mu.Lock()
	fs.cleanupStats.secretsCleaned = deletedCount
	fs.cleanupStats.lastRun = time.Now()
	fs.mu.Unlock()

	return nil
}

// IsCustomNameTaken checks if a custom name is already in use
func (s *FileStore) IsCustomNameTaken(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Skip check if name is empty
	if name == "" {
		return false, nil
	}

	// List all files in the directory
	files, err := os.ReadDir(s.basePath)
	if err != nil {
		return false, fmt.Errorf("failed to read directory: %w", err)
	}

	// Search for a secret with matching custom name
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var secret models.Secret
		if err := json.Unmarshal(data, &secret); err != nil {
			continue
		}

		if secret.CustomName == name {
			return true, nil
		}
	}

	return false, nil
}
