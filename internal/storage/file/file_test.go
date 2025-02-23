package file

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"secrets-share/internal/logger"
	"secrets-share/internal/models"

	"github.com/google/uuid"
)

func setupTestDir(t *testing.T) (string, func()) {
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

	dir, err := os.MkdirTemp("", "secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

func TestFileStore(t *testing.T) {
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	store, err := NewFileStore(testDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	// Create a test secret
	secret := &models.Secret{
		ID:         uuid.New(),
		CustomName: "test-secret",
		CreatedAt:  time.Now(),
	}

	// Test storing a secret
	t.Run("Store secret", func(t *testing.T) {
		err := store.Store(secret)
		if err != nil {
			t.Fatalf("Failed to store secret: %v", err)
		}

		// Verify file exists
		filePath := filepath.Join(testDir, secret.ID.String()+".json")
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Secret file was not created")
		}
	})

	// Test retrieving a secret by ID
	t.Run("Get secret by ID", func(t *testing.T) {
		retrieved, err := store.Get(secret.ID.String())
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Secret not found")
		}
		if retrieved.ID != secret.ID {
			t.Error("Retrieved secret ID does not match")
		}
		if retrieved.CustomName != secret.CustomName {
			t.Error("Retrieved secret custom name does not match")
		}
	})

	// Test retrieving a secret by custom name
	t.Run("Get secret by custom name", func(t *testing.T) {
		retrieved, err := store.GetByCustomName(secret.CustomName)
		if err != nil {
			t.Fatalf("Failed to get secret by custom name: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Secret not found by custom name")
		}
		if retrieved.ID != secret.ID {
			t.Error("Retrieved secret ID does not match")
		}
	})

	// Test deleting a secret
	t.Run("Delete secret", func(t *testing.T) {
		err := store.Delete(secret.ID.String())
		if err != nil {
			t.Fatalf("Failed to delete secret: %v", err)
		}

		// Verify file is deleted
		filePath := filepath.Join(testDir, secret.ID.String()+".json")
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Error("Secret file was not deleted")
		}

		// Try to get deleted secret
		retrieved, err := store.Get(secret.ID.String())
		if err != nil {
			t.Fatalf("Unexpected error when getting deleted secret: %v", err)
		}
		if retrieved != nil {
			t.Error("Deleted secret should not be retrievable")
		}
	})
}

func TestExpiredSecrets(t *testing.T) {
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	store, err := NewFileStore(testDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	// Create an expired secret
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredSecret := &models.Secret{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		ExpiresAt: &expiredTime,
	}

	// Store the expired secret
	if err := store.Store(expiredSecret); err != nil {
		t.Fatalf("Failed to store expired secret: %v", err)
	}

	// Test cleaning expired secrets
	t.Run("Clean expired secrets", func(t *testing.T) {
		err := store.CleanExpired()
		if err != nil {
			t.Fatalf("Failed to clean expired secrets: %v", err)
		}

		// Verify expired secret is deleted
		retrieved, err := store.Get(expiredSecret.ID.String())
		if err != nil {
			t.Fatalf("Unexpected error when getting expired secret: %v", err)
		}
		if retrieved != nil {
			t.Error("Expired secret should have been deleted")
		}
	})
}

func TestBurnAfterReading(t *testing.T) {
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	store, err := NewFileStore(testDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	secret := &models.Secret{
		ID:                 uuid.New(),
		CreatedAt:          time.Now(),
		IsBurnAfterReading: true,
		EncryptedData:      []byte("test-data"),
	}

	// Store the secret
	if err := store.Store(secret); err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// First read should succeed
	retrieved, err := store.Get(secret.ID.String())
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Secret should be retrievable on first read")
	}

	// Delete the secret immediately after reading
	if err := store.Delete(secret.ID.String()); err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	// Second read should return nil (secret should be deleted)
	retrieved, err = store.Get(secret.ID.String())
	if err != nil {
		t.Fatalf("Unexpected error when getting deleted secret: %v", err)
	}
	if retrieved != nil {
		t.Error("Burn-after-reading secret should have been deleted after first read")
	}
}

func TestCustomNameUniqueness(t *testing.T) {
	testDir, cleanup := setupTestDir(t)
	defer cleanup()

	store, err := NewFileStore(testDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}

	// Create first secret with custom name
	firstSecret := &models.Secret{
		ID:         uuid.New(),
		CustomName: "unique-name",
		CreatedAt:  time.Now(),
	}

	// Store first secret
	if err := store.Store(firstSecret); err != nil {
		t.Fatalf("Failed to store first secret: %v", err)
	}

	// Check if custom name is taken
	taken, err := store.IsCustomNameTaken("unique-name")
	if err != nil {
		t.Fatalf("Failed to check custom name: %v", err)
	}
	if !taken {
		t.Error("Expected custom name to be taken")
	}

	// Try to store second secret with same custom name
	secondSecret := &models.Secret{
		ID:         uuid.New(),
		CustomName: "unique-name",
		CreatedAt:  time.Now(),
	}

	err = store.Store(secondSecret)
	if err == nil {
		t.Error("Expected error when storing secret with duplicate custom name")
	}
	if err.Error() != `custom name "unique-name" is already taken` {
		t.Errorf("Expected error message %q, got %q", `custom name "unique-name" is already taken`, err.Error())
	}

	// Verify empty custom name is allowed
	emptyNameSecret := &models.Secret{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
	}

	if err := store.Store(emptyNameSecret); err != nil {
		t.Errorf("Failed to store secret with empty custom name: %v", err)
	}

	// Verify different custom names are allowed
	differentNameSecret := &models.Secret{
		ID:         uuid.New(),
		CustomName: "different-name",
		CreatedAt:  time.Now(),
	}

	if err := store.Store(differentNameSecret); err != nil {
		t.Errorf("Failed to store secret with different custom name: %v", err)
	}
}
