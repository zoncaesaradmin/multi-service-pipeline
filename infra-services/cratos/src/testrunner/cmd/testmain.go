package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sharedgomodule/logging"
	"strings"
	"sync"
	"syscall"
	"testgomodule/impl"
	"testgomodule/steps"
	"time"

	"github.com/cucumber/godog"
	"github.com/joho/godotenv"
)

var (
	currentFeature   string
	featureSetupDone = make(map[string]bool)
	featureMutex     sync.Mutex

	testStatus     = "in_progress" // possible values: in_progress, complete
	textReportPath string

	// Global logger and context for the entire test suite
	suiteLogger logging.Logger
	suiteCtx    *impl.CustomContext
)

// Context keys for passing data through Go context
type contextKey string

const (
	loggerContextKey contextKey = "suite-logger"
	suiteContextKey  contextKey = "suite-context"
)

func serveReports() *http.Server {
	http.HandleFunc("/report/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "` + testStatus + `"}`))
	})

	http.HandleFunc("/report/text", func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadFile(textReportPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Text report not found"))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
	})

	port := os.Getenv("REPORT_PORT")
	if port == "" {
		port = "4478"
	}
	fmt.Println("Starting report server on port:", port)
	server := &http.Server{Addr: ":" + port}

	return server
}

// loadEnvironmentFiles attempts to load .env files from various locations
func loadEnvironmentFiles() {
	envPaths := []string{
		".env",         // Current directory
		"./../.env",    // From component workspace root
		"./../../.env", // From service workspace root
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
}

// setupLogDirectory creates and returns the log directory path
func setupLogDirectory() string {
	logDir := os.Getenv("SERVICE_LOG_DIR")
	if logDir == "" {
		logDir = "./logs"
	}
	fmt.Printf("Log and report dir : %s\n", logDir)
	os.MkdirAll(logDir, 0755)
	return logDir
}

// runTestSuite executes the Godog test suite and returns the result
func runTestSuite(logDir string) int {
	featurePaths := getFeaturePaths()

	textReportPath = filepath.Join(logDir, "test_execution_report.txt")
	textFile, err := os.Create(textReportPath)
	if err != nil {
		log.Panicf("Failed to create text report file: %v", err)
	}
	defer textFile.Close()

	optsText := godog.Options{
		Format: "pretty",
		Paths:  featurePaths,
		Output: textFile,
	}

	return godog.TestSuite{
		Name:                 "service-testrunner",
		ScenarioInitializer:  InitializeScenario,
		TestSuiteInitializer: InitializeTestSuite,
		Options:              &optsText,
	}.Run()
}

// appendSuiteResult writes the final test result to the report file
func appendSuiteResult(suiteResult int) {
	resultText := "SUITE RESULT: "
	if suiteResult == 0 {
		resultText += "SUCCESS"
	} else {
		resultText += "FAILURE"
	}

	f, err := os.OpenFile(textReportPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString("\n" + resultText + "\n")
		f.Close()
	}
}

// handleServerShutdown manages the server based on environment configuration
func handleServerShutdown(server *http.Server) {
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	fmt.Println("Test execution is complete")
	if os.Getenv("KEEP_REPORT_SERVER") == "true" {
		fmt.Println("Report server is still running as KEEP_REPORT_SERVER is set")
		keepRunningSever(server)
	} else {
		fmt.Println("Exiting as KEEP_REPORT_SERVER is not set")
	}
}

func main() {
	loadEnvironmentFiles()

	logDir := setupLogDirectory()
	server := serveReports()
	testStatus = "in_progress"

	suiteResult := runTestSuite(logDir)
	testStatus = "complete"

	appendSuiteResult(suiteResult)
	handleServerShutdown(server)
}

func keepRunningSever(server *http.Server) {
	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down testrunner application ...")

	// Give outstanding requests a 1-second deadline to complete
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v", err)
	}
}

func getFeaturePaths() []string {
	// Feature selection logic
	var featurePaths []string
	// Highest priority: FEATURE_LIST_FILE env var (file with list of feature files)
	featureListFile := os.Getenv("FEATURE_LIST_FILE")
	if featureListFile != "" {
		data, err := ioutil.ReadFile(featureListFile)
		if err == nil {
			// Each line is a feature file path
			lines := []string{}
			for _, line := range splitLines(string(data)) {
				trimmed := trim(line)
				if trimmed != "" {
					lines = append(lines, trimmed)
				}
			}
			featurePaths = lines
		}
	} else {
		// Next priority: FEATURES env var (comma-separated list)
		selected := os.Getenv("FEATURES")
		if selected != "" {
			featurePaths = splitAndTrim(selected, ",")
		} else {
			featurePaths = []string{"features"}
		}
	}
	return featurePaths
}

func splitLines(s string) []string {
	var out []string
	curr := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			if curr != "" {
				out = append(out, curr)
				curr = ""
			}
		} else {
			curr += string(s[i])
		}
	}
	if curr != "" {
		out = append(out, curr)
	}
	return out
}

// splitAndTrim splits a string by sep and trims whitespace
func splitAndTrim(s, sep string) []string {
	var out []string
	for _, part := range split(s, sep) {
		trimmed := trim(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func split(s, sep string) []string {
	var out []string
	curr := ""
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			out = append(out, curr)
			curr = ""
		} else {
			curr += string(s[i])
		}
	}
	out = append(out, curr)
	return out
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// --- HOOKS INTEGRATION ---

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		fmt.Println("Starting test suite - global setup")

		suiteLogger, err := getDefaultLogger()
		if err != nil {
			log.Fatalf("Failed to create suite logger: %v", err)
		}

		// Initialize the global suite context once
		suiteCtx = &impl.CustomContext{
			L:           suiteLogger,
			ExampleData: make(map[string]string),
		}

		suiteLogger.Info("Test suite initialized with global logger and context")
	})

	ctx.AfterSuite(func() {
		if suiteLogger != nil {
			suiteLogger.Info("Ending test suite - global cleanup")
		} else {
			fmt.Println("Ending test suite - global cleanup")
		}
		// Cleanup any remaining feature resources
		cleanupAllFeatureResources()
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(goCtx context.Context, sc *godog.Scenario) (context.Context, error) {
		featureMutex.Lock()
		defer featureMutex.Unlock()

		// Add logger and suite context to the Go context
		goCtx = context.WithValue(goCtx, loggerContextKey, suiteLogger)
		goCtx = context.WithValue(goCtx, suiteContextKey, suiteCtx)

		// Get logger from context (with fallback)
		logger := getLoggerFromContext(goCtx)

		// Reset scenario-specific data while keeping the same context instance
		suiteCtx.CurrentScenario = sc.Name
		suiteCtx.ExampleData = make(map[string]string)

		featureName := getFeatureNameFromScenario(sc)

		// Check if this is the first scenario in a new feature
		if currentFeature != featureName {
			// Cleanup previous feature if exists
			if currentFeature != "" {
				logger.Infow("Ending feature", "feature", currentFeature)
				cleanupFeatureResources(currentFeature)
			}

			// Setup new feature
			currentFeature = featureName
			logger.Infow("Starting feature", "feature", featureName)
			if err := setupFeatureResources(logger, featureName); err != nil {
				return goCtx, fmt.Errorf("failed to setup feature %s: %w", featureName, err)
			}
			featureSetupDone[featureName] = true
		}

		logger.Infow("Starting scenario", "scenario", sc.Name, "feature", featureName)
		return goCtx, nil
	})

	ctx.After(func(goCtx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Get logger from context
		logger := getLoggerFromContext(goCtx)
		logger.Infow("Cleaning up scenario", "scenario", sc.Name)

		// Get suite context from Go context
		scenarioCtx := getSuiteContextFromContext(goCtx)
		if scenarioCtx == nil {
			logger.Warn("Suite context not found in Go context, using global")
			scenarioCtx = suiteCtx
		}

		featureName := getFeatureNameFromScenario(sc)

		// handle per scenario cleanup only for scenarios NOT using feature-level cleanup
		if featureName == "basic_tests" {
			// Clean up scenario-level resources for any specific case
			logger.Info("cleanup per scenario resources")
			if scenarioCtx.ConsHandler != nil {
				scenarioCtx.ConsHandler.Stop()
			}
			if scenarioCtx.ProducerHandler != nil {
				scenarioCtx.ProducerHandler.Stop()
			}
		} else {
			// For features using feature level resource setup,
			// just reset consumer state but don't stop the handlers
			if scenarioCtx.ConsHandler != nil {
				// Reset consumer state between scenarios (if your ConsumerHandler has a Reset method)
				scenarioCtx.ConsHandler.Reset()
			}
			logger.Debugw("Kafka resources maintained at feature level, skipping scenario cleanup", "feature", featureName)
		}

		// Always reset scenario context
		scenarioCtx.CurrentScenario = ""
		scenarioCtx.ExampleData = make(map[string]string)

		return goCtx, nil
	})

	// Register step definitions with the global context
	steps.InitializeCommonSteps(ctx, suiteCtx)
	steps.InitializeRulesSteps(ctx, suiteCtx)
}

// Helper function to get logger from Go context with fallback
func getLoggerFromContext(ctx context.Context) logging.Logger {
	if logger, ok := ctx.Value(loggerContextKey).(logging.Logger); ok {
		return logger
	}
	// Fallback to global logger
	if suiteLogger != nil {
		return suiteLogger
	}
	// Last resort: create a basic logger
	log.Printf("Warning: No logger found in context, creating fallback logger")
	dLogger, err := getDefaultLogger()
	if err != nil {
		log.Fatalf("Failed to create fallback logger: %v", err)
	}
	return dLogger
}

func getDefaultLogger() (logging.Logger, error) {
	// Initialize the global logger once for the entire test suite
	logDir := os.Getenv("SERVICE_LOG_DIR")
	if logDir == "" {
		logDir = "./logs"
	}

	return logging.NewLogger(&logging.LoggerConfig{
		Level:         logging.DebugLevel,
		FilePath:      filepath.Join(logDir, "testrunner.log"),
		LoggerName:    "testrunner",
		ComponentName: "test-component",
		ServiceName:   "cratos",
	})
}

// Helper function to get suite context from Go context with fallback
func getSuiteContextFromContext(ctx context.Context) *impl.CustomContext {
	if suiteContext, ok := ctx.Value(suiteContextKey).(*impl.CustomContext); ok {
		return suiteContext
	}
	// Fallback to global suite context
	return suiteCtx
}

// Helper function to extract feature name from scenario
func getFeatureNameFromScenario(sc *godog.Scenario) string {
	// Godog scenarios contain the feature URI
	if sc.Uri != "" {
		// Extract filename without extension
		parts := strings.Split(sc.Uri, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			return strings.TrimSuffix(filename, ".feature")
		}
	}
	return "unknown_feature"
}

// Setup Kafka producer and consumer at feature level
func setupKafkaResourcesForFeature(logger logging.Logger) error {
	logger.Info("Starting Kafka producer and consumer for feature")

	// Initialize consumer handler
	suiteCtx.ConsHandler = impl.NewConsumerHandler(logger)
	if err := suiteCtx.ConsHandler.Start(); err != nil {
		return fmt.Errorf("failed to start consumer handler: %w", err)
	}
	logger.Debug("Feature-level consumer handler started successfully")

	// Initialize producer handler
	suiteCtx.ProducerHandler = impl.NewProducerHandler(logger)
	if err := suiteCtx.ProducerHandler.Start(); err != nil {
		// If producer fails, cleanup consumer
		if stopErr := suiteCtx.ConsHandler.Stop(); stopErr != nil {
			logger.Errorw("Failed to stop consumer during producer setup failure", "error", stopErr)
		}
		suiteCtx.ConsHandler = nil
		return fmt.Errorf("failed to start producer handler: %w", err)
	}
	logger.Debug("Feature-level producer handler started successfully")

	suiteCtx.InConfigTopic = "cisco_nir-alertRules"
	suiteCtx.InDataTopic = "cisco_nir-anomalies"
	suiteCtx.OutDataTopic = "cisco_nir-prealerts"

	logger.Info("Kafka resources initialized successfully for feature")
	return nil
}

// Cleanup Kafka producer and consumer at feature level
func cleanupKafkaResourcesForFeature() {
	if suiteLogger == nil {
		return
	}

	suiteLogger.Info("Cleaning up Kafka resources for feature")

	// Cleanup consumer
	if suiteCtx.ConsHandler != nil {
		if err := suiteCtx.ConsHandler.Stop(); err != nil {
			suiteLogger.Errorw("Failed to stop feature-level consumer handler", "error", err)
		} else {
			suiteLogger.Debug("Feature-level consumer handler stopped successfully")
		}
		suiteCtx.ConsHandler = nil
	}

	// Cleanup producer
	if suiteCtx.ProducerHandler != nil {
		if err := suiteCtx.ProducerHandler.Stop(); err != nil {
			suiteLogger.Errorw("Failed to stop feature-level producer handler", "error", err)
		} else {
			suiteLogger.Debug("Feature-level producer handler stopped successfully")
		}
		suiteCtx.ProducerHandler = nil
	}

	suiteLogger.Info("Kafka resources cleanup completed for feature")
}

// Feature-level setup
func setupFeatureResources(logger logging.Logger, featureName string) error {
	logger.Infow("Setting up feature-level resources if any", "feature", featureName)

	switch featureName {
	case "basic_tests":
		// Setup Kafka producer and consumer for basic_tests.feature
		//logger.Debug("Setting up basic_tests feature resources with Kafka")
		//return setupKafkaResourcesForFeature(logger)

	case "one_to_one_tests":
		// Setup Kafka producer and consumer for one_to_one_tests.feature
		logger.Debug("Setting up one_to_one_tests feature resources with Kafka")
		return setupKafkaResourcesForFeature(logger)

	default:
		// Default feature setup
		logger.Debug("Setting up default feature resources")
		return nil
	}
	return nil
}

// Feature-level cleanup
func cleanupFeatureResources(featureName string) {
	if suiteLogger != nil {
		suiteLogger.Infow("Cleaning up resources for feature", "feature", featureName)
	}

	switch featureName {
	case "basic_tests":
		// if suiteLogger != nil {
		// 	suiteLogger.Debug("Cleaning up basic_tests feature resources with Kafka")
		// }
		// cleanupKafkaResourcesForFeature()

	case "one_to_one_tests":
		if suiteLogger != nil {
			suiteLogger.Debug("Cleaning up one_to_one_tests feature resources with Kafka")
		}
		cleanupKafkaResourcesForFeature()

	default:
		if suiteLogger != nil {
			suiteLogger.Debug("Cleaning up default feature resources")
		}
	}
}

// Cleanup all feature resources (called in AfterSuite)
func cleanupAllFeatureResources() {
	if currentFeature != "" {
		cleanupFeatureResources(currentFeature)
	}
}
