package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sharedgomodule/logging"
	"sharedgomodule/utils"
	"strconv"
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
	testStatus     = "in_progress" // possible values: in_progress, complete
	textReportPath string
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
		data, err := os.ReadFile(textReportPath)
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

	// Run tests using unified approach (automatically handles parallel vs non-parallel)
	runTestSuites(logDir)

	time.Sleep(40 * time.Second)
	testStatus = "complete"

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
		data, err := os.ReadFile(featureListFile)
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

// Check if parallel execution is requested
func shouldRunParallel() bool {
	return os.Getenv("PARALLEL_EXECUTION") == "true"
}

// Get concurrency level for parallel execution
func getConcurrency() int {
	if concStr := os.Getenv("TEST_CONCURRENCY"); concStr != "" {
		if conc, err := strconv.Atoi(concStr); err == nil && conc > 0 {
			return conc
		}
	}
	return 1 // Default to sequential execution
}

// Get maximum parallel groups from environment
func getMaxParallelGroups() int {
	if groupsStr := os.Getenv("MAX_PARALLEL_GROUPS"); groupsStr != "" {
		if groups, err := strconv.Atoi(groupsStr); err == nil && groups > 0 {
			return groups
		}
	}
	return 2 // Default to 2 parallel groups
}

// Run test suites (handles both parallel and non-parallel execution)
func runTestSuites(logDir string) int {
	featurePaths := getFeaturePaths()

	// Group features for execution (automatically handles parallel vs non-parallel)
	featureGroups := groupFeaturesForExecution(featurePaths)

	var wg sync.WaitGroup
	results := make(chan int, len(featureGroups))

	if len(featureGroups) > 1 {
		fmt.Printf("Running %d feature groups in parallel\n", len(featureGroups))
	} else {
		fmt.Printf("Running %d feature group\n", len(featureGroups))
	}

	for i, group := range featureGroups {
		wg.Add(1)
		go func(groupIndex int, features []string) {
			defer wg.Done()
			result := runFeatureGroup(logDir, groupIndex, features)
			results <- result
		}(i, group)
	}

	// Wait for all suites to complete
	wg.Wait()
	close(results)

	// Aggregate results
	finalResult := 0
	groupCount := 0
	for result := range results {
		groupCount++
		if result != 0 {
			finalResult = result // If any suite fails, overall result is failure
		}
	}

	if len(featureGroups) > 1 {
		fmt.Printf("Parallel execution completed: %d groups processed\n", groupCount)
	} else {
		fmt.Printf("Test execution completed: %d group processed\n", groupCount)
	}

	// Merge all reports
	mergeReports(logDir, len(featureGroups), finalResult)

	return finalResult
}

// Group features for execution (handles both parallel and non-parallel)
func groupFeaturesForExecution(featurePaths []string) [][]string {
	// Check if parallel execution is requested
	if !shouldRunParallel() {
		// Non-parallel execution: return all features as a single group (group 0)
		return [][]string{featurePaths}
	}

	// Parallel execution: distribute features across multiple groups
	maxGroups := getMaxParallelGroups()

	if len(featurePaths) <= maxGroups {
		// Each feature gets its own group
		groups := make([][]string, len(featurePaths))
		for i, path := range featurePaths {
			groups[i] = []string{path}
		}
		return groups
	}

	// Distribute features across groups
	groups := make([][]string, maxGroups)
	for i, path := range featurePaths {
		groupIndex := i % maxGroups
		groups[groupIndex] = append(groups[groupIndex], path)
	}

	return groups
}

// Run a specific group of features
func runFeatureGroup(logDir string, groupIndex int, features []string) int {
	// Create group-specific report file (always use group_N format)
	groupReportPath := filepath.Join(logDir, fmt.Sprintf("test_execution_report_%d.txt", groupIndex))

	textFile, err := os.Create(groupReportPath)
	if err != nil {
		log.Printf("Failed to create group report file: %v", err)
		return 1
	}
	defer textFile.Close()

	// Create group-specific logger (always use group_N format)
	logFileName := fmt.Sprintf("testrunner_%d.log", groupIndex)
	loggerName := fmt.Sprintf("testrunner-%d", groupIndex)
	suiteName := fmt.Sprintf("service-testrunner-%d", groupIndex)

	groupLogger, err := logging.NewLogger(&logging.LoggerConfig{
		Level:         logging.DebugLevel,
		FilePath:      filepath.Join(logDir, logFileName),
		LoggerName:    loggerName,
		ComponentName: "test-component",
		ServiceName:   "cratos",
	})
	if err != nil {
		log.Printf("Failed to create group logger: %v", err)
		return 1
	}

	groupLogger.Infow("Starting feature group", "group", groupIndex, "features", features)

	// Get concurrency for this group (Godog built-in concurrency)
	concurrency := getConcurrency()

	optsText := godog.Options{
		Format:      "pretty",
		Paths:       features,
		Output:      textFile,
		Concurrency: concurrency,
	}

	// Create test suite with group-based initializers (treats non-parallel as single group)
	result := godog.TestSuite{
		Name:                 suiteName,
		ScenarioInitializer:  scenarioInitializer(groupLogger, groupIndex),
		TestSuiteInitializer: suiteInitializer(groupLogger, groupIndex),
		Options:              &optsText,
	}.Run()

	groupLogger.Infow("Completed feature group", "group", groupIndex, "result", result)
	return result
}

// Create group-specific scenario initializer (no global state, each group is independent)
func scenarioInitializer(groupLogger logging.Logger, groupIndex int) func(*godog.ScenarioContext) {
	// Create group-specific context - each group has its own isolated state
	var groupCurrentFeature string
	var groupFeatureSetupDone = make(map[string]bool)
	var groupFeatureMutex sync.Mutex

	groupSuiteCtx := &impl.CustomContext{
		L:           groupLogger,
		ExampleData: make(map[string]string),
	}

	// Each group maintains its own isolated state - no global variables needed

	return func(ctx *godog.ScenarioContext) {
		ctx.Before(func(goCtx context.Context, sc *godog.Scenario) (context.Context, error) {
			groupFeatureMutex.Lock()
			defer groupFeatureMutex.Unlock()

			// Add context
			goCtx = context.WithValue(goCtx, loggerContextKey, groupLogger)
			goCtx = context.WithValue(goCtx, suiteContextKey, groupSuiteCtx)

			// Reset scenario-specific data
			groupSuiteCtx.CurrentScenario = sc.Name
			groupSuiteCtx.ExampleData = make(map[string]string)

			featureName := getFeatureNameFromScenario(sc)

			// Check if this is the first scenario in a new feature for this group
			if groupCurrentFeature != featureName {
				// Cleanup previous feature if exists
				if groupCurrentFeature != "" {
					groupLogger.Infow("Ending feature in group", "feature", groupCurrentFeature, "group", groupIndex)
					cleanupGroupFeatureResources(groupSuiteCtx, groupCurrentFeature, groupLogger)
				}

				// Setup new feature for this group
				groupCurrentFeature = featureName
				groupLogger.Infow("Starting feature in group", "feature", featureName, "group", groupIndex)

				if err := setupGroupFeatureResources(groupSuiteCtx, groupLogger, featureName, groupIndex); err != nil {
					return goCtx, fmt.Errorf("failed to setup feature %s in group %d: %w", featureName, groupIndex, err)
				}
				groupFeatureSetupDone[featureName] = true
			}

			groupLogger.Infow("Starting scenario in group", "scenario", sc.Name, "feature", featureName, "group", groupIndex)
			return goCtx, nil
		})

		ctx.After(func(goCtx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
			featureName := getFeatureNameFromScenario(sc)

			// Handle cleanup based on feature type
			if featureName == "basic_tests" {
				// Clean up scenario-level resources
				groupLogger.Infow("Cleaning up scenario resources in group", "scenario", sc.Name, "group", groupIndex)
				if groupSuiteCtx.ConsHandler != nil {
					groupSuiteCtx.ConsHandler.Stop()
				}
				if groupSuiteCtx.ProducerHandler != nil {
					groupSuiteCtx.ProducerHandler.Stop()
				}
			} else {
				// For features using feature level resource setup, just reset consumer state
				if groupSuiteCtx.ConsHandler != nil {
					groupSuiteCtx.ConsHandler.Reset()
				}
				groupLogger.Debugw("Kafka resources maintained at feature level in group, skipping scenario cleanup", "feature", featureName, "group", groupIndex)
			}

			// Reset scenario context
			groupSuiteCtx.CurrentScenario = ""
			groupSuiteCtx.ExampleData = make(map[string]string)

			return goCtx, nil
		})

		// Register step definitions with the group context
		steps.InitializeCommonSteps(ctx, groupSuiteCtx)
		steps.InitializeRulesSteps(ctx, groupSuiteCtx)
	}
}

// Create group-specific suite initializer
func suiteInitializer(groupLogger logging.Logger, groupIndex int) func(*godog.TestSuiteContext) {
	return func(ctx *godog.TestSuiteContext) {
		ctx.BeforeSuite(func() {
			groupLogger.Infow("Starting test suite for group", "group", groupIndex)
		})

		ctx.AfterSuite(func() {
			groupLogger.Infow("Ending test suite for group", "group", groupIndex)
			// Each group cleans up its own resources during scenario transitions
		})
	}
}

// Setup feature resources for a specific group
func setupGroupFeatureResources(groupCtx *impl.CustomContext, logger logging.Logger, featureName string, groupIndex int) error {
	logger.Infow("Setting up group feature resources", "feature", featureName, "group", groupIndex)

	switch featureName {
	case "basic_tests":
		// basic_tests uses scenario-level setup, no feature-level setup needed
		logger.Debugw("Basic tests use scenario-level setup", "group", groupIndex)
		return nil

	default:
		// Setup Kafka producer and consumer for feature by default
		logger.Debugw("Setting up feature resources with Kafka", "group", groupIndex)
		return setupGroupKafkaResources(groupCtx, logger, groupIndex)

	}
}

// Setup Kafka resources for a specific group
func setupGroupKafkaResources(groupCtx *impl.CustomContext, logger logging.Logger, groupIndex int) error {
	logger.Infow("Starting Kafka producer and consumer for group", "group", groupIndex)

	// Initialize consumer handler for this group
	groupCtx.ConsHandler = impl.NewConsumerHandler(logger, groupIndex)
	if err := groupCtx.ConsHandler.Start(); err != nil {
		return fmt.Errorf("failed to start consumer handler for group %d: %w", groupIndex, err)
	}
	logger.Debugw("Group consumer handler started successfully", "group", groupIndex)

	// Initialize producer handler for this group
	groupCtx.ProducerHandler = impl.NewProducerHandler(logger, groupIndex)
	if err := groupCtx.ProducerHandler.Start(); err != nil {
		// If producer fails, cleanup consumer
		if stopErr := groupCtx.ConsHandler.Stop(); stopErr != nil {
			logger.Errorw("Failed to stop consumer during producer setup failure", "error", stopErr, "group", groupIndex)
		}
		groupCtx.ConsHandler = nil
		return fmt.Errorf("failed to start producer handler for group %d: %w", groupIndex, err)
	}
	logger.Debugw("Group producer handler started successfully", "group", groupIndex)

	// Set default topics for the group
	groupCtx.InConfigTopic = utils.GetEnv("PROCESSING_RULES_TOPIC", "cisco_nir-alertRules")
	groupCtx.InDataTopic = utils.GetEnv("PROCESSING_INPUT_TOPIC", "cisco_nir-anomalies")
	groupCtx.OutDataTopic = utils.GetEnv("PROCESSING_OUTPUT_TOPIC", "cisco_nir-prealerts")
	groupCtx.ExecGroupIndex = groupIndex

	logger.Infow("Kafka resources initialized successfully for group", "group", groupIndex)
	return nil
}

// Cleanup feature resources for a specific group
func cleanupGroupFeatureResources(groupCtx *impl.CustomContext, featureName string, logger logging.Logger) {
	logger.Infow("Cleaning up group feature resources", "feature", featureName)

	switch featureName {
	case "basic_tests":
		// basic_tests uses scenario-level cleanup, no feature-level cleanup needed
		logger.Debug("Basic tests use scenario-level cleanup")

	default:
		logger.Debug("Cleaning up feature resources with Kafka")
		cleanupGroupKafkaResources(groupCtx, logger)

	}
}

// Cleanup Kafka resources for a specific group
func cleanupGroupKafkaResources(groupCtx *impl.CustomContext, logger logging.Logger) {
	logger.Info("Cleaning up Kafka resources for group")

	// Cleanup consumer
	if groupCtx.ConsHandler != nil {
		if err := groupCtx.ConsHandler.Stop(); err != nil {
			logger.Errorw("Failed to stop group consumer handler", "error", err)
		} else {
			logger.Debug("Group consumer handler stopped successfully")
		}
		groupCtx.ConsHandler = nil
	}

	// Cleanup producer
	if groupCtx.ProducerHandler != nil {
		if err := groupCtx.ProducerHandler.Stop(); err != nil {
			logger.Errorw("Failed to stop group producer handler", "error", err)
		} else {
			logger.Debug("Group producer handler stopped successfully")
		}
		groupCtx.ProducerHandler = nil
	}

	logger.Info("Kafka resources cleanup completed for group")
}

// Merge group reports into a single final report
func mergeReports(logDir string, numGroups int, finalResult int) {
	finalReportPath := filepath.Join(logDir, "complete_test_execution_report.txt")
	finalFile, err := os.Create(finalReportPath)
	if err != nil {
		log.Printf("Failed to create final report file: %v", err)
		return
	}
	defer finalFile.Close()

	// Set global textReportPath so the HTTP server can serve the merged report
	textReportPath = finalReportPath

	if numGroups > 1 {
		finalFile.WriteString("=== PARALLEL TEST EXECUTION REPORT ===\n\n")
	} else {
		finalFile.WriteString("=== TEST EXECUTION REPORT ===\n\n")
	}

	for i := 0; i < numGroups; i++ {
		groupReportPath := filepath.Join(logDir, fmt.Sprintf("test_execution_report_%d.txt", i))

		finalFile.WriteString(fmt.Sprintf("=== GROUP %d RESULTS ===\n", i))

		if data, err := os.ReadFile(groupReportPath); err == nil {
			finalFile.Write(data)
		} else {
			finalFile.WriteString(fmt.Sprintf("Error reading group %d report: %v\n", i, err))
		}

		finalFile.WriteString(fmt.Sprintf("\n=== END GROUP %d ===\n\n", i))
	}

	// Add the final suite result status expected by test scripts
	resultText := "SUITE RESULT: "
	if finalResult == 0 {
		resultText += "SUCCESS"
	} else {
		resultText += "FAILURE"
	}
	finalFile.WriteString(resultText + "\n")
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
