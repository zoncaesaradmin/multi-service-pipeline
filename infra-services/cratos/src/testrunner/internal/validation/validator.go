package validation

import (
	"reflect"

	"testgomodule/internal/config"
)

// ValidationResult represents the result of output validation
type ValidationResult struct {
	Success bool        `json:"success"`
	Details interface{} `json:"details,omitempty"`
}

// Validator handles output validation
type Validator struct {
	config config.ValidationConfig
}

// NewValidator creates a new validator with the given configuration
func NewValidator(cfg config.ValidationConfig) *Validator {
	return &Validator{
		config: cfg,
	}
}

// ValidateOutput validates the actual output against expected output
func (v *Validator) ValidateOutput(actual, expected map[string]interface{}) (ValidationResult, error) {
	result := ValidationResult{
		Success: true,
		Details: make(map[string]interface{}),
	}

	// Compare the outputs
	if !v.deepEqual(actual, expected) {
		result.Success = false
		result.Details = map[string]interface{}{
			"actual":   actual,
			"expected": expected,
			"error":    "output mismatch",
		}
		return result, nil
	}

	result.Details = map[string]interface{}{
		"message": "validation passed",
	}

	return result, nil
}

// deepEqual performs deep comparison of two interface{} values
func (v *Validator) deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}
