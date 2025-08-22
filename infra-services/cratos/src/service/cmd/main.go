package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"servicegomodule/internal/api"
	"servicegomodule/internal/app"
	"servicegomodule/internal/config"
	"sharedgomodule/logging"
	"sharedgomodule/utils"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file for local development (ignored in production)
	loadEnvFile()

	// Log environment info
	logEnvironmentInfo()
	cfg := loadConfig()
	if cfg == nil {
		log.Fatal("Failed to load configuration, exiting")
	}

	logger := initLoggerSettings(cfg)
	defer logger.Close()

	// Create application instance
	application := app.NewApplication(cfg, logger)

	// Start the application and its processing pipeline
	if err := application.Start(); err != nil {
		logger.Fatalf("Failed to start application: %v", err)
	}

	// Initialize handlers and setup HTTP mux
	mux := setupRouter(logger)

	// Start server
	startServer(mux, cfg, application)
}

func setupRouter(logger logging.Logger) *http.ServeMux {

	handler := api.NewHandler(logger)
	mux := http.NewServeMux()

	// Setup routes
	handler.SetupRoutes(mux)

	return mux
}

func startServer(mux *http.ServeMux, cfg *config.RawConfig, application *app.Application) {
	logger := application.Logger()

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Debugf("Starting http server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down application ...")

	// Give outstanding requests a 10-second deadline to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	// Shutdown the application
	if err := application.Shutdown(); err != nil {
		logger.Errorf("Application shutdown error: %v", err)
	}

	logger.Info("Server exited")
}

func loadConfig() *config.RawConfig {
	// Load configuration using absolute paths based on SERVICE_HOME environment variable
	homeDir := os.Getenv("SERVICE_HOME")
	if homeDir == "" {
		log.Fatal("SERVICE_HOME environment variable is required and must point to the repository root")
	}

	// Load configuration from the centralized config file
	configPath := filepath.Join(homeDir, "conf", "config.yaml")

	return config.LoadConfigWithDefaults(configPath)
}

func initLoggerSettings(cfg *config.RawConfig) logging.Logger {
	// create the log directory path if it does not exist
	os.MkdirAll(utils.GetEnv("SERVICE_LOG_DIR", ""), 0755)

	// update all log file paths to absolute paths
	logDir := os.Getenv("SERVICE_LOG_DIR")
	if logDir != "" && !filepath.IsAbs(cfg.Logging.FileName) {
		cfg.Logging.FileName = filepath.Join(logDir, cfg.Logging.FileName)
		cfg.Processing.PloggerConfig.FileName = filepath.Join(logDir, cfg.Processing.PloggerConfig.FileName)
		cfg.Processing.Processor.RuleProcConfig.Logging.FileName = filepath.Join(logDir, cfg.Processing.Processor.RuleProcConfig.Logging.FileName)
	}

	// Convert config logging configuration to logger config
	loggerConfig := cfg.Logging.ConvertToLoggerConfig()

	logger, err := logging.NewLogger(&loggerConfig)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	return logger
}

// loadEnvFile loads .env file for local development
// In production (Docker/K8s), environment variables are set directly
func loadEnvFile() {
	// Determine if we're in a containerized environment
	if isRunningInContainer() {
		log.Println("Running in container - using system environment variables")
		return
	}

	// Try to load .env file from workspace root for local development
	envPaths := []string{
		".env",             // Current directory
		"../../../.env",    // From service/cmd back to workspace root
		"../../../../.env", // Alternative path
		filepath.Join(os.Getenv("SERVICE_HOME"), ".env"), // Using SERVICE_HOME if set
	}

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err == nil {
				log.Printf("✅ Loaded environment from: %s", envPath)
				continue
			} else {
				log.Printf("❌ Failed to load .env from %s: %v", envPath, err)
			}
		}
	}

	// If no .env file found, that's fine for production
	log.Println("ℹ️  No .env file found - using system environment variables")
}

// isRunningInContainer detects if the application is running in a container
func isRunningInContainer() bool {
	// Check for common container environment indicators
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true // Running in Kubernetes
	}
	if os.Getenv("CONTAINER") == "true" {
		return true // Explicit container flag
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true // Docker container
	}
	return false
}

// logEnvironmentInfo logs information about the current environment
func logEnvironmentInfo() {
	appEnv := getEnvWithDefault("APP_ENV", "production")
	appName := getEnvWithDefault("APP_NAME", "cratos")
	appVersion := getEnvWithDefault("APP_VERSION", "unknown")

	log.Printf("🚀 Starting %s v%s in %s environment", appName, appVersion, appEnv)

	if isRunningInContainer() {
		log.Println("📦 Running in containerized environment")
	} else {
		log.Println("💻 Running in local development environment")
	}
}

// getEnvWithDefault gets an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
