package types

import (
	"encoding/json"
	"time"
)

// Response represents a standard API response structure
type Response struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(data interface{}, message string) *Response {
	return &Response{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(error string, message string) *Response {
	return &Response{
		Success:   false,
		Message:   message,
		Error:     error,
		Timestamp: time.Now(),
	}
}

// ToJSON converts the response to JSON bytes
func (r *Response) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	*Response
	Pagination *PaginationInfo `json:"pagination,omitempty"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, pagination *PaginationInfo, message string) *PaginatedResponse {
	return &PaginatedResponse{
		Response:   NewSuccessResponse(data, message),
		Pagination: pagination,
	}
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Uptime    time.Duration     `json:"uptime"`
	Checks    map[string]string `json:"checks,omitempty"`
}

// ServiceMetrics represents basic service metrics
type ServiceMetrics struct {
	RequestCount    int64         `json:"request_count"`
	ErrorCount      int64         `json:"error_count"`
	AverageResponse time.Duration `json:"average_response_time"`
	Uptime          time.Duration `json:"uptime"`
	MemoryUsage     uint64        `json:"memory_usage_bytes"`
	ActiveUsers     int           `json:"active_users"`
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// NewValidationErrors creates a new validation errors collection
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}

// Add adds a validation error to the collection
func (ve *ValidationErrors) Add(field, value, message, code string) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		Code:    code,
	})
}

// HasErrors returns true if there are validation errors
func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Error implements the error interface
func (ve *ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}

	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, err.Message)
	}

	result, _ := json.Marshal(messages)
	return string(result)
}
