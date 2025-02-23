package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Security  SecurityConfig  `mapstructure:"security"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Secrets   SecretsConfig   `mapstructure:"secrets"`
	Redis     RedisConfig     `mapstructure:"redis"`
	CORS      CORSConfig      `mapstructure:"cors"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Env  string `mapstructure:"env"`
}

type SecurityConfig struct {
	EnableCaptcha        bool `mapstructure:"enable_captcha"`
	ServerSideEncryption bool `mapstructure:"server_side_encryption"`
}

type RouteRateLimit struct {
	RequestsPerHour   int `mapstructure:"requests_per_hour"`
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
}

type RateLimitConfig struct {
	Enabled bool                      `mapstructure:"enabled"`
	Routes  map[string]RouteRateLimit `mapstructure:"routes"`
	Default RouteRateLimit            `mapstructure:"default"`
}

type SecretsConfig struct {
	MaxSizeBytes         int    `mapstructure:"max_size_bytes"`
	MaxCustomNameLength  int    `mapstructure:"max_custom_name_length"`
	DefaultExpiryMinutes int    `mapstructure:"default_expiry_minutes"`
	MaxExpiryDays        int    `mapstructure:"max_expiry_days"`
	StoragePath          string `mapstructure:"storage_path"`
	CleanupIntervalSec   int    `mapstructure:"cleanup_interval_sec"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	DB       int    `mapstructure:"db"`
	Password string
	Username string
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type LoggingConfig struct {
	Enabled          bool               `mapstructure:"enabled"`
	ConsoleOutput    bool               `mapstructure:"console_output"`
	Directory        string             `mapstructure:"directory"`
	ArchiveDirectory string             `mapstructure:"archive_directory"`
	Rotation         LogRotationConfig  `mapstructure:"rotation"`
	Retention        LogRetentionConfig `mapstructure:"retention"`
	Files            LogFilesConfig     `mapstructure:"files"`
}

type LogRotationConfig struct {
	SizeMB int `mapstructure:"size_mb"`
}

type LogRetentionConfig struct {
	Days int `mapstructure:"days"`
}

type LogFileConfig struct {
	Filename string `mapstructure:"filename"`
}

type LogFilesConfig struct {
	Error       LogFileConfig `mapstructure:"error"`
	Access      LogFileConfig `mapstructure:"access"`
	Ratelimit   LogFileConfig `mapstructure:"ratelimit"`
	Application LogFileConfig `mapstructure:"application"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Load environment variables
	viper.AutomaticEnv()

	// Load sensitive configuration from environment
	config.Redis.Password = os.Getenv("REDIS_PASSWORD")
	config.Redis.Username = os.Getenv("REDIS_USERNAME")

	// Ensure storage directory exists
	if err := os.MkdirAll(filepath.Join(configPath, config.Secrets.StoragePath), 0750); err != nil {
		return nil, fmt.Errorf("error creating storage directory: %w", err)
	}

	return &config, nil
}
