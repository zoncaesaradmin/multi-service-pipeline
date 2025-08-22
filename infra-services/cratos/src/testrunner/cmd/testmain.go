package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testgomodule/steps"

	"github.com/cucumber/godog"
)

func InitializeScenario(ctx *godog.ScenarioContext) {
	steps.InitializeCommonSteps(ctx)
	steps.InitializeRulesSteps(ctx)
}

var testStatus = "in_progress" // possible values: in_progress, complete
func serveReports() {
	http.HandleFunc("/report/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "` + testStatus + `"}`))
	})

	http.HandleFunc("/report/text", func(w http.ResponseWriter, r *http.Request) {
		textReportPath := filepath.Join(os.Getenv("SERVICE_LOG_DIR"), "test_report.txt")
		data, err := ioutil.ReadFile(textReportPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Text report not found"))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(data)
	})

	http.HandleFunc("/report/json", func(w http.ResponseWriter, r *http.Request) {
		jsonReportPath := filepath.Join(os.Getenv("SERVICE_LOG_DIR"), "test_report.json")
		data, err := ioutil.ReadFile(jsonReportPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("JSON report not found"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	port := os.Getenv("REPORT_PORT")
	if port == "" {
		port = "8081"
	}
	go http.ListenAndServe(":"+port, nil)
}

func main() {
	// Use SERVICE_LOG_DIR for logs if set
	logDir := os.Getenv("SERVICE_LOG_DIR")
	if logDir == "" {
		logDir = "./logs"
	}
	os.MkdirAll(logDir, 0755)

	serveReports()

	testStatus = "in_progress"

	featurePaths := getFeaturePaths()

	// Run Godog for text report

	textReportPath := filepath.Join(logDir, "test_report.txt")
	textFile, err := os.Create(textReportPath)
	if err != nil {
		panic("Failed to create text report file: " + err.Error())
	}
	defer textFile.Close()
	optsText := godog.Options{
		Format: "pretty",
		Paths:  featurePaths,
		Output: textFile,
	}
	godog.TestSuite{
		Name:                "infra-testrunner",
		ScenarioInitializer: InitializeScenario,
		Options:             &optsText,
	}.Run()
	testStatus = "complete"

	// // Run Godog for JSON report
	// jsonReportPath := "../test_report.json"
	// jsonFile, err := os.Create(jsonReportPath)
	// if err != nil {
	// 	panic("Failed to create JSON report file: " + err.Error())
	// }
	// defer jsonFile.Close()
	// optsJson := godog.Options{
	// 	Format: "cucumber",
	// 	Paths:  featurePaths,
	// 	Output: jsonFile,
	// }
	// godog.TestSuite{
	// 	Name:                "infra-testrunner",
	// 	ScenarioInitializer: InitializeScenario,
	// 	Options:             &optsJson,
	// }.Run()
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
	// ...existing code...
}
