package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"secrets-share/internal/config"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	config     *Config
	writers    map[string]io.Writer
	mu         sync.Mutex
	production bool
}

type Config struct {
	Enabled        bool
	ConsoleOutput  bool
	Directory      string
	ArchiveDir     string
	RotationSizeMB int
	RetentionDays  int
	Files          map[string]FileConfig
}

type FileConfig struct {
	Filename string
}

type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
}

var (
	defaultLogger *Logger
	once          sync.Once
)

func Init(cfg *Config, production bool) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(cfg, production)
	})
	return err
}

func NewLogger(cfg *Config, production bool) (*Logger, error) {
	// Get the project root directory (where the config.yaml is located)
	projectRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %v", err)
	}

	// Clean the paths and make them absolute from the project root
	logDir := filepath.Clean(filepath.Join(projectRoot, cfg.Directory))
	archiveDir := filepath.Clean(filepath.Join(projectRoot, cfg.ArchiveDir))

	// Create log directories with proper permissions
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %v", err)
	}
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %v", err)
	}

	l := &Logger{
		config:     cfg,
		writers:    make(map[string]io.Writer),
		production: production,
	}

	// Configure writers for each log file
	for name, fileCfg := range cfg.Files {
		logPath := filepath.Join(logDir, fileCfg.Filename)
		writer := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    cfg.RotationSizeMB,
			MaxAge:     cfg.RetentionDays,
			MaxBackups: 10, // Keep at most 10 old files
			Compress:   true,
			LocalTime:  true,
		}
		l.writers[name] = writer
	}

	return l, nil
}

func (l *Logger) log(level LogLevel, logType string, message string, data interface{}) {
	if !l.config.Enabled {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Type:      logType,
		Data:      data,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling log entry: %v\n", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to appropriate log file
	if writer, ok := l.writers[logType]; ok {
		if _, err := writer.Write(append(jsonData, '\n')); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
		}
	}

	// Write to console in development mode
	if !l.production && l.config.ConsoleOutput {
		fmt.Println(string(jsonData))
	}
}

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Helper functions for the default logger
func Debug(message string, data interface{}) {
	defaultLogger.log(DebugLevel, "application", message, data)
}

func Info(message string, data interface{}) {
	defaultLogger.log(InfoLevel, "application", message, data)
}

func Warn(message string, data interface{}) {
	defaultLogger.log(WarnLevel, "application", message, data)
}

func Error(message string, data interface{}) {
	defaultLogger.log(ErrorLevel, "error", message, data)
}

func Access(message string, data interface{}) {
	defaultLogger.log(InfoLevel, "access", message, data)
}

func RateLimit(message string, data interface{}) {
	defaultLogger.log(InfoLevel, "ratelimit", message, data)
}

// LogStartupInfo prints server startup information in a nice ASCII format
func LogStartupInfo(cfg interface{}, redisConnected bool, envVars map[string]string) {
	banner := `
    _                      ____                   
   / \   _ __   ___  _ __ |  _ \ _ __ ___  _ __  
  / _ \ | '_ \ / _ \| '_ \| | | | '__/ _ \| '_ \ 
 / ___ \| | | | (_) | | | | |_| | | | (_) | |_) |
/_/   \_\_| |_|\___/|_| |_|____/|_|  \___/| .__/ 
                                          |_|.link 
`
	fmt.Print("\033[1;36m", banner, "\033[0m") // Cyan color for banner

	serverCfg := cfg.(*config.Config)
	goVersion := runtime.Version()
	osInfo := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	hostname, _ := os.Hostname()

	// Format environment variables
	var envList []string
	sensitiveKeys := []string{"key", "secret", "password", "token"}
	for k, v := range envVars {
		shouldRedact := false
		lowerKey := strings.ToLower(k)
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(lowerKey, sensitive) {
				shouldRedact = true
				break
			}
		}
		if v != "" { // Only show set variables
			if shouldRedact {
				envList = append(envList, fmt.Sprintf("%s=********", k))
			} else {
				envList = append(envList, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	// Format the output
	fmt.Printf("\033[1;33m=== Server Information ===\033[0m\n")
	fmt.Printf("Environment: \033[1;32m%s\033[0m\n", serverCfg.Server.Env)
	fmt.Printf("Host: %s:%d\n", serverCfg.Server.Host, serverCfg.Server.Port)
	fmt.Printf("Go Version: %s\n", goVersion)
	fmt.Printf("OS/Arch: %s\n", osInfo)
	fmt.Printf("Hostname: %s\n\n", hostname)

	fmt.Printf("\033[1;33m=== Features ===\033[0m\n")
	fmt.Printf("Redis: %s\n", formatStatus(redisConnected))
	fmt.Printf("Rate Limiting: %s\n", formatStatus(serverCfg.RateLimit.Enabled))
	fmt.Printf("Captcha: %s\n", formatStatus(serverCfg.Security.EnableCaptcha))
	fmt.Printf("Server-side Encryption: %s\n\n", formatStatus(serverCfg.Security.ServerSideEncryption))

	fmt.Printf("\033[1;33m=== Environment Variables ===\033[0m\n")
	for _, env := range envList {
		fmt.Printf("%s\n", env)
	}
	fmt.Println()

	fmt.Printf("\033[1;32m=== Server is ready ===\033[0m\n\n")
}

// formatStatus returns a colored status string
func formatStatus(enabled bool) string {
	if enabled {
		return "\033[1;32menabled\033[0m" // Green
	}
	return "\033[1;31mdisabled\033[0m" // Red
}

// GinLogger returns a gin.HandlerFunc (middleware) that logs requests using our custom logger
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		data := map[string]interface{}{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"ip":         c.ClientIP(),
			"latency":    time.Since(start).String(),
			"user_agent": c.Request.UserAgent(),
		}

		Access(fmt.Sprintf("%s %s", c.Request.Method, path), data)
	}
}
