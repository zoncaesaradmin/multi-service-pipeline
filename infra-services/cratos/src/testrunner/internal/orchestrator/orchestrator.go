package orchestrator

import (
	"fmt"
	"log"
	"time"

	"testgomodule/internal/config"
	"testgomodule/internal/harness"
	"testgomodule/internal/process"
	"testgomodule/internal/testdata"
	"testgomodule/internal/types"
	"testgomodule/internal/validation"
)

// Orchestrator manages the execution of test scenarios
type Orchestrator struct {
	config         *config.Config
	harness        harness.TestHarness
	processManager *process.Manager
	validator      *validation.Validator
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	// Create test harness based on configuration
	h, err := harness.NewTestHarness(cfg.MessageBus)
	if err != nil {
		return nil, fmt.Errorf("failed to create test harness: %w", err)
	}

	// Create process manager
	pm := process.NewManager(cfg.Service)

	// Create validator
	validator := validation.NewValidator(cfg.Validation)

	return &Orchestrator{
		config:         cfg,
		harness:        h,
		processManager: pm,
		validator:      validator,
	}, nil
}

// ExecuteScenario executes a single test scenario
func (o *Orchestrator) ExecuteScenario(scenario testdata.TestScenario) (types.TestResult, error) {
	log.Printf("Starting execution of scenario: %s", scenario.Name)
	start := time.Now()

	result := types.TestResult{
		ScenarioName: scenario.Name,
		Success:      false,
	}

	// Start service process
	if err := o.processManager.StartService(); err != nil {
		result.Error = fmt.Sprintf("failed to start service: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("failed to start service: %w", err)
	}

	// Ensure cleanup
	defer func() {
		if err := o.processManager.StopService(); err != nil {
			log.Printf("Warning: failed to stop service: %v", err)
		}
	}()

	// Wait for service to be ready
	if err := o.processManager.WaitForReady(); err != nil {
		result.Error = fmt.Sprintf("service not ready: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("service not ready: %w", err)
	}

	// Initialize test harness
	if err := o.harness.Initialize(); err != nil {
		result.Error = fmt.Sprintf("failed to initialize harness: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("failed to initialize harness: %w", err)
	}

	// Send input data
	if err := o.harness.SendMessage(scenario.Input); err != nil {
		result.Error = fmt.Sprintf("failed to send input: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("failed to send input: %w", err)
	}

	// Receive output data
	output, err := o.harness.ReceiveMessage(scenario.Timeout)
	if err != nil {
		result.Error = fmt.Sprintf("failed to receive output: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("failed to receive output: %w", err)
	}

	// Validate results
	validationResult, err := o.validator.ValidateOutput(output, scenario.ExpectedOutput)
	if err != nil {
		result.Error = fmt.Sprintf("validation failed: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("validation failed: %w", err)
	}

	result.Success = validationResult.Success
	result.Duration = time.Since(start)
	result.Details = validationResult.Details

	if !validationResult.Success {
		result.Error = "output validation failed"
		return result, fmt.Errorf("output validation failed")
	}

	log.Printf("Scenario '%s' completed successfully in %v", scenario.Name, result.Duration)
	return result, nil
}

// Cleanup releases any resources held by the orchestrator
func (o *Orchestrator) Cleanup() error {
	if err := o.harness.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup harness: %w", err)
	}

	if err := o.processManager.StopService(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	return nil
}
