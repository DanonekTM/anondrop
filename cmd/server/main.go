package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"secrets-share/internal/api/handlers"
	"secrets-share/internal/captcha"
	"secrets-share/internal/config"
	"secrets-share/internal/encryption"
	"secrets-share/internal/logger"
	"secrets-share/internal/storage/file"
	"secrets-share/internal/storage/redis"
)

func getRateLimits(c *gin.Context, cfg *config.Config) (int, int) {
	route := c.FullPath()

	routeMap := map[string]string{
		"/api/secrets":            "create_secret",
		"/api/secrets/:id":        "view_secret",
		"/api/secrets/name/:name": "view_secret_by_name",
	}

	if configKey, exists := routeMap[route]; exists {
		if limits, ok := cfg.RateLimit.Routes[configKey]; ok {
			return limits.RequestsPerHour, limits.RequestsPerMinute
		}
	}

	logger.Debug("Using default rate limits", map[string]interface{}{
		"route":  route,
		"hour":   cfg.RateLimit.Default.RequestsPerHour,
		"minute": cfg.RateLimit.Default.RequestsPerMinute,
	})
	return cfg.RateLimit.Default.RequestsPerHour, cfg.RateLimit.Default.RequestsPerMinute
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		logger.Warn("Failed to load .env file", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		logger.Error("Failed to load config", err)
		os.Exit(1)
	}

	// Initialize logger
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	loggerConfig := &logger.Config{
		Enabled:        cfg.Logging.Enabled,
		ConsoleOutput:  cfg.Logging.ConsoleOutput,
		Directory:      cfg.Logging.Directory,
		ArchiveDir:     cfg.Logging.ArchiveDirectory,
		RotationSizeMB: cfg.Logging.Rotation.SizeMB,
		RetentionDays:  cfg.Logging.Retention.Days,
		Files: map[string]logger.FileConfig{
			"error": {
				Filename: cfg.Logging.Files.Error.Filename,
			},
			"access": {
				Filename: cfg.Logging.Files.Access.Filename,
			},
			"ratelimit": {
				Filename: cfg.Logging.Files.Ratelimit.Filename,
			},
			"application": {
				Filename: cfg.Logging.Files.Application.Filename,
			},
		},
	}

	if err := logger.Init(loggerConfig, cfg.Server.Env == "production"); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Initialize storage
	fileStore, err := file.NewFileStore(cfg.Secrets.StoragePath)
	if err != nil {
		logger.Error("Failed to initialize file store", err)
		os.Exit(1)
	}

	// Perform initial cleanup of expired secrets
	logger.Info("Performing startup cleanup of expired secrets", nil)
	if err := fileStore.CleanExpired(); err != nil {
		logger.Warn("Startup cleanup failed", err)
	} else {
		stats := fileStore.GetCleanupStats()
		if stats.SecretsCleaned > 0 {
			logger.Info("Startup cleanup completed", map[string]interface{}{
				"secrets_cleaned": stats.SecretsCleaned,
			})
		} else {
			logger.Info("Startup cleanup completed: no expired secrets found", nil)
		}
	}

	// Initialize Redis store (optional)
	var redisStore *redis.RedisStore
	logger.Info("Connecting to Redis", map[string]interface{}{
		"host": cfg.Redis.Host,
		"port": cfg.Redis.Port,
	})

	redisStore, err = redis.NewRedisStore(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.Username,
		cfg.Redis.DB,
	)
	if err != nil {
		logger.Warn("Redis store not available", err)
		logger.Warn("Running without Redis features (rate limiting disabled)", nil)
	} else {
		logger.Info("Successfully connected to Redis", map[string]interface{}{
			"host": cfg.Redis.Host,
			"port": cfg.Redis.Port,
		})
		logger.Info("Rate limiting is enabled", nil)
	}

	// Initialize encryptor
	encryptor := encryption.NewEncryptor(os.Getenv("SERVER_ENCRYPTION_KEY"))

	// Initialize Turnstile client
	turnstileClient := captcha.NewTurnstileClient(os.Getenv("CAPTCHA_SECRET_KEY"))

	// Initialize secret handler
	secretHandler := handlers.NewSecretAPIHandler(fileStore, redisStore, encryptor, turnstileClient, cfg)

	// Log startup information
	envVars := map[string]string{
		"SERVER_ENCRYPTION_KEY": os.Getenv("SERVER_ENCRYPTION_KEY"),
		"CAPTCHA_SECRET_KEY":    os.Getenv("CAPTCHA_SECRET_KEY"),
		"REDIS_USERNAME":        os.Getenv("REDIS_USERNAME"),
		"REDIS_PASSWORD":        os.Getenv("REDIS_PASSWORD"),
	}
	logger.LogStartupInfo(cfg, redisStore != nil, envVars)

	// Initialize Gin router
	router := gin.New()
	router.Use(logger.GinLogger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			for _, allowed := range cfg.CORS.AllowedOrigins {
				if origin == allowed {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Rate limiting middleware (only if Redis is available)
	if cfg.RateLimit.Enabled && redisStore != nil {
		router.Use(func(c *gin.Context) {
			ip := c.ClientIP()
			route := c.FullPath()
			requestsPerHour, requestsPerMinute := getRateLimits(c, cfg)

			logger.Debug("Rate limit check", map[string]interface{}{
				"route":               route,
				"requests_per_hour":   requestsPerHour,
				"requests_per_minute": requestsPerMinute,
			})

			allowed, err := redisStore.CheckRateLimit(
				c.Request.Context(),
				ip,
				route,
				requestsPerHour,
				requestsPerMinute,
			)
			if err != nil {
				logger.Error("Rate limit check failed", err)
				c.Next()
				return
			}
			if !allowed {
				logger.RateLimit("Rate limit exceeded", map[string]interface{}{
					"route": route,
					"ip":    ip,
				})
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit exceeded. Please try again later.",
				})
				return
			}
			c.Next()
		})
	}

	// API routes
	api := router.Group("/api")
	{
		secrets := api.Group("/secrets")
		{
			secrets.POST("", secretHandler.CreateSecret)
			secrets.POST("/name/:name", secretHandler.GetSecretByName)
			secrets.POST("/:id", secretHandler.GetSecret)
		}
	}

	// Create context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create WaitGroup for cleanup goroutines
	var wg sync.WaitGroup

	// Start cleanup goroutine
	cleanupTicker := time.NewTicker(time.Duration(cfg.Secrets.CleanupIntervalSec) * time.Second)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cleanupTicker.Stop()

		logger.Info("Starting cleanup routine", map[string]interface{}{
			"interval": time.Duration(cfg.Secrets.CleanupIntervalSec) * time.Second,
		})

		for {
			select {
			case <-ctx.Done():
				logger.Info("Cleanup routine shutting down", nil)
				// Perform one final cleanup
				if err := fileStore.CleanExpired(); err != nil {
					logger.Error("Final cleanup failed", err)
				} else {
					stats := fileStore.GetCleanupStats()
					if stats.SecretsCleaned > 0 {
						logger.Info("Final cleanup completed", map[string]interface{}{
							"secrets_cleaned": stats.SecretsCleaned,
						})
					}
				}
				return
			case <-cleanupTicker.C:
				if err := fileStore.CleanExpired(); err != nil {
					logger.Error("Failed to clean expired secrets", err)
					continue
				}

				// Log cleanup statistics
				stats := fileStore.GetCleanupStats()
				if stats.SecretsCleaned > 0 || stats.Errors > 0 {
					logger.Info("Periodic cleanup completed", map[string]interface{}{
						"secrets_cleaned": stats.SecretsCleaned,
						"errors":          stats.Errors,
						"last_run":        stats.LastRun.Format(time.RFC3339),
					})
				}
			}
		}
	}()

	// Start HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", map[string]interface{}{
			"address": srv.Addr,
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("Shutdown signal received", nil)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", err)
	}

	// Close Redis connection if it exists
	if redisStore != nil {
		if err := redisStore.Close(); err != nil {
			logger.Error("Redis connection close error", err)
		}
	}

	// Wait for cleanup goroutine to finish
	wg.Wait()
	logger.Info("Server shutdown complete", nil)
}
