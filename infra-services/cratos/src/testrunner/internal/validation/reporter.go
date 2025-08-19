package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"testgomodule/internal/types"
)

// TestReport represents a complete test execution report
type TestReport struct {
	Timestamp time.Time          `json:"timestamp"`
	Results   []types.TestResult `json:"results"`
}

// Reporter handles test result reporting
type Reporter struct {
	format string
}

// NewReporter creates a new reporter with the specified format
func NewReporter(format string) *Reporter {
	return &Reporter{
		format: format,
	}
}

// GenerateReport generates a test report in the specified format
func (r *Reporter) GenerateReport(report TestReport) error {
	switch r.format {
	case "console":
		return r.generateConsoleReport(report)
	case "json":
		return r.generateJSONReport(report)
	case "junit":
		return r.generateJUnitReport(report)
	default:
		return fmt.Errorf("unsupported report format: %s", r.format)
	}
}

// generateConsoleReport generates a human-readable console report
func (r *Reporter) generateConsoleReport(report TestReport) error {
	fmt.Printf("\n=== Test Execution Report ===\n")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Printf("Total scenarios: %d\n\n", len(report.Results))

	successful := 0
	for _, result := range report.Results {
		status := "✓ PASS"
		if !result.Success {
			status = "✗ FAIL"
		} else {
			successful++
		}

		fmt.Printf("%s %s (%v)\n", status, result.ScenarioName, result.Duration)
		if result.Error != "" {
			fmt.Printf("    Error: %s\n", result.Error)
		}
	}

	successRate := float64(successful) / float64(len(report.Results)) * 100
	fmt.Printf("\nSummary: %d/%d passed (%.1f%%)\n", successful, len(report.Results), successRate)

	return nil
}

// generateJSONReport generates a JSON report
func (r *Reporter) generateJSONReport(report TestReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	filename := fmt.Sprintf("test-report-%s.json", report.Timestamp.Format("20060102-150405"))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	fmt.Printf("JSON report written to: %s\n", filename)
	return nil
}

// generateJUnitReport generates a JUnit XML report
func (r *Reporter) generateJUnitReport(report TestReport) error {
	xml := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	xml += fmt.Sprintf(`<testsuite name="Cratos Test Suite" tests="%d" failures="%d" time="%.3f" timestamp="%s">`,
		len(report.Results),
		r.countFailures(report.Results),
		r.calculateTotalTime(report.Results),
		report.Timestamp.Format(time.RFC3339))
	xml += "\n"

	for _, result := range report.Results {
		xml += fmt.Sprintf(`  <testcase name="%s" classname="Scenario" time="%.3f">`,
			result.ScenarioName, result.Duration.Seconds())
		xml += "\n"

		if !result.Success {
			xml += fmt.Sprintf(`    <failure message="%s"></failure>`, result.Error)
			xml += "\n"
		}

		xml += "  </testcase>\n"
	}

	xml += "</testsuite>\n"

	filename := fmt.Sprintf("test-report-%s.xml", report.Timestamp.Format("20060102-150405"))
	if err := os.WriteFile(filename, []byte(xml), 0644); err != nil {
		return fmt.Errorf("failed to write JUnit report: %w", err)
	}

	fmt.Printf("JUnit report written to: %s\n", filename)
	return nil
}

func (r *Reporter) countFailures(results []types.TestResult) int {
	failures := 0
	for _, result := range results {
		if !result.Success {
			failures++
		}
	}
	return failures
}

func (r *Reporter) calculateTotalTime(results []types.TestResult) float64 {
	var total time.Duration
	for _, result := range results {
		total += result.Duration
	}
	return total.Seconds()
}
