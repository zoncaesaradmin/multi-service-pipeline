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
	"syscall"
	"testgomodule/impl"
	"testgomodule/steps"
	"time"

	"github.com/cucumber/godog"
	"github.com/joho/godotenv"
)

var testStatus = "in_progress" // possible values: in_progress, complete
var textReportPath string

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

	// http.HandleFunc("/report/json", func(w http.ResponseWriter, r *http.Request) {
	// 	jsonReportPath := filepath.Join(os.Getenv("SERVICE_LOG_DIR"), "test_execution_report.json")
	// 	data, err := ioutil.ReadFile(jsonReportPath)
	// 	if err != nil {
	// 		w.WriteHeader(http.StatusNotFound)
	// 		w.Write([]byte("JSON report not found"))
	// 		return
	// 	}
	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.Write(data)
	// })

	port := os.Getenv("REPORT_PORT")
	if port == "" {
		port = "4478"
	}
	fmt.Println("Starting report server on port:", port)
	// Start HTTP server in a goroutine and signal when it's done
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
		fmt.Println("🔧 Global setup: runs ONCE before all features")
		// Add global setup logic here
	})
	ctx.AfterSuite(func() {
		fmt.Println("🧹 Global cleanup: runs ONCE after all features")
		// Add global cleanup logic here
	})
}

// Scenario step registration
func InitializeScenario(ctx *godog.ScenarioContext) {
	logDir := os.Getenv("SERVICE_LOG_DIR")
	logger, err := logging.NewLogger(&logging.LoggerConfig{
		Level:         logging.DebugLevel,
		FilePath:      filepath.Join(logDir, "testrunner.log"),
		LoggerName:    "testrunner",
		ComponentName: "test-component",
		ServiceName:   "cratos",
	})
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	suiteCtx := &impl.CustomContext{L: logger}

	steps.InitializeCommonSteps(ctx, suiteCtx)
	steps.InitializeRulesSteps(ctx, suiteCtx)
}
