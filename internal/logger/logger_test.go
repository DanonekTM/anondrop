package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// testWriter is a simple io.Writer for testing
type testWriter struct {
	buffer bytes.Buffer
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	return tw.buffer.Write(p)
}

func (tw *testWriter) String() string {
	return tw.buffer.String()
}

func setupTestLogger(t *testing.T) (*Logger, *testWriter, func()) {
	// Create temp directory for test logs
	tmpDir, err := os.MkdirTemp("", "logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create test writer
	tw := &testWriter{}

	// Create logger config
	cfg := &Config{
		Enabled:       true,
		ConsoleOutput: true,
		Directory:     tmpDir,
		ArchiveDir:    filepath.Join(tmpDir, "archive"),
		Files: map[string]FileConfig{
			"application": {Filename: "app.log"},
			"error":       {Filename: "error.log"},
			"access":      {Filename: "access.log"},
		},
	}

	// Create logger
	logger, err := NewLogger(cfg, false)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Replace writers with test writer
	logger.writers = map[string]io.Writer{
		"application": tw,
		"error":       tw,
		"access":      tw,
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return logger, tw, cleanup
}

func TestLogger(t *testing.T) {
	logger, tw, cleanup := setupTestLogger(t)
	defer cleanup()

	t.Run("Debug logging", func(t *testing.T) {
		testMessage := "test debug message"
		testData := map[string]string{"key": "value"}

		logger.log(DebugLevel, "application", testMessage, testData)

		// Parse the log entry
		var entry LogEntry
		if err := json.Unmarshal([]byte(tw.String()), &entry); err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		// Verify log entry
		if entry.Level != "DEBUG" {
			t.Errorf("Expected level DEBUG, got %s", entry.Level)
		}
		if entry.Message != testMessage {
			t.Errorf("Expected message %q, got %q", testMessage, entry.Message)
		}
		if entry.Type != "application" {
			t.Errorf("Expected type application, got %s", entry.Type)
		}
	})

	t.Run("Error logging", func(t *testing.T) {
		tw.buffer.Reset() // Clear previous logs

		testMessage := "test error message"
		testData := map[string]string{"error": "something went wrong"}

		logger.log(ErrorLevel, "error", testMessage, testData)

		var entry LogEntry
		if err := json.Unmarshal([]byte(tw.String()), &entry); err != nil {
			t.Fatalf("Failed to parse log entry: %v", err)
		}

		if entry.Level != "ERROR" {
			t.Errorf("Expected level ERROR, got %s", entry.Level)
		}
	})

	t.Run("Log levels", func(t *testing.T) {
		levels := []struct {
			level LogLevel
			str   string
		}{
			{DebugLevel, "DEBUG"},
			{InfoLevel, "INFO"},
			{WarnLevel, "WARN"},
			{ErrorLevel, "ERROR"},
		}

		for _, l := range levels {
			if l.level.String() != l.str {
				t.Errorf("Expected %s for level %d, got %s", l.str, l.level, l.level.String())
			}
		}
	})
}

func TestLoggerInitialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logger-init-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	archiveDir := filepath.Join(tmpDir, "archive")
	// Create archive directory
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("Failed to create archive directory: %v", err)
	}

	cfg := &Config{
		Enabled:        true,
		ConsoleOutput:  true,
		Directory:      tmpDir,
		ArchiveDir:     archiveDir,
		RotationSizeMB: 10,
		RetentionDays:  7,
		Files: map[string]FileConfig{
			"application": {Filename: "app.log"},
		},
	}

	logger, err := NewLogger(cfg, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if len(logger.writers) != len(cfg.Files) {
		t.Errorf("Expected %d writers, got %d", len(cfg.Files), len(logger.writers))
	}

	// Check if log directories were created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		t.Errorf("Archive directory %s was not created", archiveDir)
	}
}

func TestLoggerDisabled(t *testing.T) {
	logger, tw, cleanup := setupTestLogger(t)
	defer cleanup()

	// Disable logging
	logger.config.Enabled = false

	// Try to log something
	logger.log(DebugLevel, "application", "test message", nil)

	// Buffer should be empty when logging is disabled
	if tw.String() != "" {
		t.Error("Expected no output when logging is disabled")
	}
}
