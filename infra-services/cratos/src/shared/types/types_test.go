package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	usernameShortMessage = "Username too short"
	emailRequiredMessage = "Email is required"
)

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	message := "Operation successful"

	response := NewSuccessResponse(data, message)

	assert.True(t, response.Success)
	assert.Equal(t, message, response.Message)
	assert.Equal(t, data, response.Data)
	assert.Empty(t, response.Error)
	assert.False(t, response.Timestamp.IsZero())
}

func TestNewErrorResponse(t *testing.T) {
	errorMsg := "Something went wrong"
	message := "Operation failed"

	response := NewErrorResponse(errorMsg, message)

	assert.False(t, response.Success)
	assert.Equal(t, message, response.Message)
	assert.Equal(t, errorMsg, response.Error)
	assert.Nil(t, response.Data)
	assert.False(t, response.Timestamp.IsZero())
}

func TestResponseToJSON(t *testing.T) {
	response := NewSuccessResponse("test data", "success")

	jsonBytes, err := response.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Verify it's valid JSON
	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)
	assert.Equal(t, true, decoded["success"])
	assert.Equal(t, "test data", decoded["data"])
}

func TestNewPaginatedResponse(t *testing.T) {
	data := []string{"item1", "item2"}
	pagination := &PaginationInfo{
		Page:       1,
		PageSize:   10,
		Total:      2,
		TotalPages: 1,
	}
	message := "Items retrieved"

	response := NewPaginatedResponse(data, pagination, message)

	assert.True(t, response.Success)
	assert.Equal(t, message, response.Message)
	assert.Equal(t, data, response.Data)
	assert.Equal(t, pagination, response.Pagination)
}

func TestValidationErrors(t *testing.T) {
	ve := NewValidationErrors()

	// Initially no errors
	assert.False(t, ve.HasErrors())
	assert.Equal(t, "no validation errors", ve.Error())

	// Add an error
	ve.Add("username", "ab", usernameShortMessage, "MIN_LENGTH")

	assert.True(t, ve.HasErrors())
	assert.Len(t, ve.Errors, 1)
	assert.Equal(t, "username", ve.Errors[0].Field)
	assert.Equal(t, "ab", ve.Errors[0].Value)
	assert.Equal(t, usernameShortMessage, ve.Errors[0].Message)
	assert.Equal(t, "MIN_LENGTH", ve.Errors[0].Code)

	// Add another error
	ve.Add("email", "", emailRequiredMessage, "REQUIRED")

	assert.True(t, ve.HasErrors())
	assert.Len(t, ve.Errors, 2)

	// Error string should be JSON array of messages
	errorStr := ve.Error()
	assert.Contains(t, errorStr, usernameShortMessage)
	assert.Contains(t, errorStr, emailRequiredMessage)
}

func TestValidationErrorsJSON(t *testing.T) {
	ve := NewValidationErrors()
	ve.Add("username", "ab", usernameShortMessage, "MIN_LENGTH")
	ve.Add("email", "", emailRequiredMessage, "REQUIRED")

	// Should be able to marshal to JSON
	jsonBytes, err := json.Marshal(ve)
	require.NoError(t, err)

	// Should be able to unmarshal back
	var decoded ValidationErrors
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Errors, 2)
	assert.Equal(t, "username", decoded.Errors[0].Field)
	assert.Equal(t, "email", decoded.Errors[1].Field)
}

func TestPaginationInfo(t *testing.T) {
	pagination := &PaginationInfo{
		Page:       2,
		PageSize:   20,
		Total:      100,
		TotalPages: 5,
	}

	// Should marshal to JSON properly
	jsonBytes, err := json.Marshal(pagination)
	require.NoError(t, err)

	var decoded PaginationInfo
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 2, decoded.Page)
	assert.Equal(t, 20, decoded.PageSize)
	assert.Equal(t, 100, decoded.Total)
	assert.Equal(t, 5, decoded.TotalPages)
}

func TestHealthStatus(t *testing.T) {
	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Hour,
		Checks: map[string]string{
			"database": "ok",
			"cache":    "ok",
		},
	}

	// Should marshal to JSON properly
	jsonBytes, err := json.Marshal(status)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	var decoded HealthStatus
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "healthy", decoded.Status)
	assert.Equal(t, "1.0.0", decoded.Version)
	assert.Equal(t, "ok", decoded.Checks["database"])
}

func TestServiceMetrics(t *testing.T) {
	metrics := &ServiceMetrics{
		RequestCount:    1000,
		ErrorCount:      5,
		AverageResponse: 100 * time.Millisecond,
		Uptime:          24 * time.Hour,
		MemoryUsage:     1024 * 1024 * 100, // 100MB
		ActiveUsers:     50,
	}

	// Should marshal to JSON properly
	jsonBytes, err := json.Marshal(metrics)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	var decoded ServiceMetrics
	err = json.Unmarshal(jsonBytes, &decoded)
	require.NoError(t, err)

	assert.Equal(t, int64(1000), decoded.RequestCount)
	assert.Equal(t, int64(5), decoded.ErrorCount)
	assert.Equal(t, 50, decoded.ActiveUsers)
}
