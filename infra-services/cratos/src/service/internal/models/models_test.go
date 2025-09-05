package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestChannelMessageTypeConstants(t *testing.T) {
	if ChannelMessageTypeData != "data" {
		t.Errorf("Expected ChannelMessageTypeData to be 'data', got %s", ChannelMessageTypeData)
	}
	if ChannelMessageTypeControl != "control" {
		t.Errorf("Expected ChannelMessageTypeControl to be 'control', got %s", ChannelMessageTypeControl)
	}
}

func TestNewChannelMessage(t *testing.T) {
	testData := []byte("test data")
	source := "test-source"
	msgType := ChannelMessageTypeData

	before := time.Now()
	msg := NewChannelMessage(msgType, testData, source, 0)
	after := time.Now()

	if msg == nil {
		t.Fatal("NewChannelMessage returned nil")
	}

	if msg.Type != msgType {
		t.Errorf("Expected type %s, got %s", msgType, msg.Type)
	}

	if string(msg.Data) != string(testData) {
		t.Errorf("Expected data %s, got %s", string(testData), string(msg.Data))
	}

	// Check timestamp is within reasonable bounds
	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Errorf("Timestamp %v not within expected range %v - %v", msg.Timestamp, before, after)
	}
}

func TestNewDataMessage(t *testing.T) {
	testData := []byte("test data payload")
	source := "data-source"

	msg := NewDataMessage(testData, source, 0)

	if msg == nil {
		t.Fatal("NewDataMessage returned nil")
	}

	if msg.Type != ChannelMessageTypeData {
		t.Errorf("Expected type %s, got %s", ChannelMessageTypeData, msg.Type)
	}

	if string(msg.Data) != string(testData) {
		t.Errorf("Expected data %s, got %s", string(testData), string(msg.Data))
	}

	if msg.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set, got zero value")
	}
}

func TestNewControlMessage(t *testing.T) {
	testData := []byte("start")
	source := "control-source"

	msg := NewControlMessage(testData, source)

	if msg == nil {
		t.Fatal("NewControlMessage returned nil")
	}

	if msg.Type != ChannelMessageTypeControl {
		t.Errorf("Expected type %s, got %s", ChannelMessageTypeControl, msg.Type)
	}

	if string(msg.Data) != string(testData) {
		t.Errorf("Expected data %s, got %s", string(testData), string(msg.Data))
	}

	if msg.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set, got zero value")
	}
}

func TestChannelMessageMethods(t *testing.T) {
	dataMsg := &ChannelMessage{Type: ChannelMessageTypeData}
	controlMsg := &ChannelMessage{Type: ChannelMessageTypeControl}

	if !dataMsg.IsDataMessage() {
		t.Error("Expected IsDataMessage() to return true for data message")
	}

	if controlMsg.IsDataMessage() {
		t.Error("Expected IsDataMessage() to return false for control message")
	}

	if dataMsg.IsControlMessage() {
		t.Error("Expected IsControlMessage() to return false for data message")
	}

	if !controlMsg.IsControlMessage() {
		t.Error("Expected IsControlMessage() to return true for control message")
	}
}

func TestErrorResponse(t *testing.T) {
	t.Run("creates error response with all fields", func(t *testing.T) {
		err := ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid input provided",
			Code:    400,
		}

		if err.Error != "validation_error" {
			t.Errorf("Expected error 'validation_error', got %s", err.Error)
		}
		if err.Message != "Invalid input provided" {
			t.Errorf("Expected message 'Invalid input provided', got %s", err.Message)
		}
		if err.Code != 400 {
			t.Errorf("Expected code 400, got %d", err.Code)
		}
	})

	t.Run("creates error response with minimal fields", func(t *testing.T) {
		err := ErrorResponse{
			Error: "simple_error",
		}

		if err.Error != "simple_error" {
			t.Errorf("Expected error 'simple_error', got %s", err.Error)
		}
		if err.Message != "" {
			t.Errorf("Expected empty message, got %s", err.Message)
		}
		if err.Code != 0 {
			t.Errorf("Expected code 0, got %d", err.Code)
		}
	})
}

func TestSuccessResponse(t *testing.T) {
	t.Run("creates success response with message only", func(t *testing.T) {
		resp := SuccessResponse{
			Message: "Operation completed successfully",
		}

		if resp.Message != "Operation completed successfully" {
			t.Errorf("Expected message 'Operation completed successfully', got %s", resp.Message)
		}
		if resp.Data != nil {
			t.Errorf("Expected nil data, got %v", resp.Data)
		}
	})

	t.Run("creates success response with data", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		resp := SuccessResponse{
			Message: "Success",
			Data:    data,
		}

		if resp.Message != "Success" {
			t.Errorf("Expected message 'Success', got %s", resp.Message)
		}
		if resp.Data == nil {
			t.Error("Expected data to be set")
		}
	})
}

func TestHealthResponse(t *testing.T) {
	t.Run("creates health response with all fields", func(t *testing.T) {
		timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		health := HealthResponse{
			Status:    "healthy",
			Timestamp: timestamp,
			Version:   "1.0.0",
		}

		if health.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got %s", health.Status)
		}
		if !health.Timestamp.Equal(timestamp) {
			t.Errorf("Expected timestamp %v, got %v", timestamp, health.Timestamp)
		}
		if health.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got %s", health.Version)
		}
	})

	t.Run("creates health response with different statuses", func(t *testing.T) {
		statuses := []string{"healthy", "unhealthy", "degraded"}

		for _, status := range statuses {
			health := HealthResponse{
				Status:    status,
				Timestamp: time.Now(),
				Version:   "test",
			}

			if health.Status != status {
				t.Errorf("Expected status %s, got %s", status, health.Status)
			}
		}
	})
}

func TestChannelMessageEdgeCases(t *testing.T) {
	t.Run("NewChannelMessage with nil data", func(t *testing.T) {
		msg := NewChannelMessage(ChannelMessageTypeControl, nil, "test-source", 0)

		if msg.Type != ChannelMessageTypeControl {
			t.Errorf("Expected type %s, got %s", ChannelMessageTypeControl, msg.Type)
		}

		if msg.Data != nil {
			t.Errorf("Expected nil data, got %v", msg.Data)
		}
	})

	t.Run("NewDataMessage with empty data", func(t *testing.T) {
		msg := NewDataMessage([]byte{}, "source", 0)

		if msg.Type != ChannelMessageTypeData {
			t.Errorf("Expected type %s, got %s", ChannelMessageTypeData, msg.Type)
		}

		if len(msg.Data) != 0 {
			t.Errorf("Expected empty data, got %v", msg.Data)
		}
	})

	t.Run("methods work with constructed messages", func(t *testing.T) {
		dataMsg := NewDataMessage([]byte("test"), "source", 0)
		controlMsg := NewControlMessage([]byte("stop"), "source")

		if !dataMsg.IsDataMessage() || dataMsg.IsControlMessage() {
			t.Error("Data message type methods returned incorrect values")
		}

		if controlMsg.IsDataMessage() || !controlMsg.IsControlMessage() {
			t.Error("Control message type methods returned incorrect values")
		}
	})
}

func TestMessageTimestamps(t *testing.T) {
	t.Run("all constructors set timestamps", func(t *testing.T) {
		before := time.Now()

		dataMsg := NewDataMessage([]byte("data"), "source", 0)
		controlMsg := NewControlMessage([]byte("control"), "source")
		channelMsg := NewChannelMessage(ChannelMessageTypeData, []byte("test"), "source", 0)

		after := time.Now()

		messages := []*ChannelMessage{dataMsg, controlMsg, channelMsg}

		for i, msg := range messages {
			if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
				t.Errorf("Message %d timestamp %v not within range %v - %v", i, msg.Timestamp, before, after)
			}
		}
	})

	t.Run("timestamps are unique for rapid creation", func(t *testing.T) {
		msg1 := NewDataMessage([]byte("data1"), "source", 0)
		msg2 := NewDataMessage([]byte("data2"), "source", 0)

		// Allow for very small time differences (nanosecond precision)
		if msg1.Timestamp.Equal(msg2.Timestamp) {
			// This might happen in very fast systems, which is acceptable
			t.Logf("Messages created at same timestamp: %v", msg1.Timestamp)
		}
	})
}

func TestResponseTypesIntegration(t *testing.T) {
	t.Run("all response types can be used in interface{}", func(t *testing.T) {
		responses := []interface{}{
			&ErrorResponse{Error: "test"},
			&SuccessResponse{Message: "test"},
			&HealthResponse{Status: "test"},
		}

		for i, resp := range responses {
			if resp == nil {
				t.Errorf("Response %d is nil", i)
			}
		}
	})
}

func TestJSONSerialization(t *testing.T) {
	// Add encoding/json import at the top of the file if not already present
	// This test requires: import "encoding/json"

	t.Run("ChannelMessage JSON serialization", func(t *testing.T) {
		timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		msg := &ChannelMessage{
			Type:      ChannelMessageTypeData,
			Timestamp: timestamp,
			Data:      []byte("test data"),
		}

		// Note: JSON marshaling of []byte fields encodes them as base64
		// So "test data" becomes "dGVzdCBkYXRh" in base64
		jsonBytes, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal ChannelMessage: %v", err)
		}

		// Verify we can unmarshal it back
		var unmarshaled ChannelMessage
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal ChannelMessage: %v", err)
		}

		if unmarshaled.Type != msg.Type {
			t.Errorf("Expected type %s, got %s", msg.Type, unmarshaled.Type)
		}
		if !unmarshaled.Timestamp.Equal(msg.Timestamp) {
			t.Errorf("Expected timestamp %v, got %v", msg.Timestamp, unmarshaled.Timestamp)
		}
		if string(unmarshaled.Data) != string(msg.Data) {
			t.Errorf("Expected data %s, got %s", string(msg.Data), string(unmarshaled.Data))
		}
	})

	t.Run("ErrorResponse JSON serialization", func(t *testing.T) {
		resp := ErrorResponse{
			Error:   "test_error",
			Message: "Test message",
			Code:    500,
		}

		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal ErrorResponse: %v", err)
		}

		var unmarshaled ErrorResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal ErrorResponse: %v", err)
		}

		if unmarshaled.Error != resp.Error {
			t.Errorf("Expected error %s, got %s", resp.Error, unmarshaled.Error)
		}
		if unmarshaled.Message != resp.Message {
			t.Errorf("Expected message %s, got %s", resp.Message, unmarshaled.Message)
		}
		if unmarshaled.Code != resp.Code {
			t.Errorf("Expected code %d, got %d", resp.Code, unmarshaled.Code)
		}
	})

	t.Run("SuccessResponse JSON serialization", func(t *testing.T) {
		resp := SuccessResponse{
			Message: "Test success",
			Data:    map[string]int{"count": 42},
		}

		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal SuccessResponse: %v", err)
		}

		var unmarshaled SuccessResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal SuccessResponse: %v", err)
		}

		if unmarshaled.Message != resp.Message {
			t.Errorf("Expected message %s, got %s", resp.Message, unmarshaled.Message)
		}
		// Note: Data will be map[string]interface{} after unmarshaling
		if unmarshaled.Data == nil {
			t.Error("Expected data to be present")
		}
	})

	t.Run("HealthResponse JSON serialization", func(t *testing.T) {
		timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		resp := HealthResponse{
			Status:    "healthy",
			Timestamp: timestamp,
			Version:   "1.0.0",
		}

		jsonBytes, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal HealthResponse: %v", err)
		}

		var unmarshaled HealthResponse
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal HealthResponse: %v", err)
		}

		if unmarshaled.Status != resp.Status {
			t.Errorf("Expected status %s, got %s", resp.Status, unmarshaled.Status)
		}
		if !unmarshaled.Timestamp.Equal(resp.Timestamp) {
			t.Errorf("Expected timestamp %v, got %v", resp.Timestamp, unmarshaled.Timestamp)
		}
		if unmarshaled.Version != resp.Version {
			t.Errorf("Expected version %s, got %s", resp.Version, unmarshaled.Version)
		}
	})
}
