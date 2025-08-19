package types

import "time"

// TestResult represents the result of a test scenario execution
type TestResult struct {
	ScenarioName string        `json:"scenario_name"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	Details      interface{}   `json:"details,omitempty"`
}
