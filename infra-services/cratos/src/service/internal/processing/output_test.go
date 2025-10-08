package processing

import (
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"testing"
	"time"
)

const testComponentName = "output-handler"

// createTestLogger creates a mock logger for testing
func createTestLogger() *logging.MockLogger {
	mockLogger := logging.NewMockLogger()
	mockLogger.SetComponentName(testComponentName)
	return mockLogger
}

func TestOutputConfig(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "test-output-topic",
		BatchSize:         10,
		FlushTimeout:      5 * time.Second,
		ChannelBufferSize: 100,
	}

	if config.OutputTopic != "test-output-topic" {
		t.Errorf("Expected output topic to be 'test-output-topic', got %s", config.OutputTopic)
	}
	if config.BatchSize != 10 {
		t.Errorf("Expected batch size to be 10, got %d", config.BatchSize)
	}
	if config.FlushTimeout != 5*time.Second {
		t.Errorf("Expected flush timeout to be 5s, got %v", config.FlushTimeout)
	}
	if config.ChannelBufferSize != 100 {
		t.Errorf("Expected channel buffer size to be 100, got %d", config.ChannelBufferSize)
	}
}

func TestOutputHandlerGetOutputChannel(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "test-topic",
		BatchSize:         5,
		FlushTimeout:      1 * time.Second,
		ChannelBufferSize: 50,
	}
	mockLogger := createTestLogger()

	handler := NewOutputHandler(config, mockLogger, nil)
	channel := handler.GetOutputChannel()

	if channel == nil {
		t.Fatal("Expected output channel to be returned, got nil")
	}

	// Test that we can send a message to the channel
	testMsg := models.NewDataMessage([]byte("test"), "test", 0)
	select {
	case channel <- testMsg:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Could not send message to output channel")
	}
}

func TestOutputHandlerStartSuccess(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "start-topic",
		BatchSize:         5,
		FlushTimeout:      100 * time.Millisecond,
		ChannelBufferSize: 10,
	}
	mockLogger := createTestLogger()

	handler := NewOutputHandler(config, mockLogger, nil)

	err := handler.Start()
	if err != nil {
		t.Fatalf("Expected no error starting output handler, got %v", err)
	}

	// Test that the handler state shows it's running
	stats := handler.GetStatus()
	if stats["status"] != "running" {
		t.Errorf("Expected handler status to be 'running', got %v", stats["status"])
	}

	// Verify that the mock logger captured some log entries
	if mockLogger.GetLogCount() == 0 {
		t.Error("Expected some log entries to be captured by mock logger")
	}

	// Verify component name is included in log entries
	if !mockLogger.HasLogEntryWithField(logging.InfoLevel, "component", testComponentName) {
		t.Error("Expected component field in log entries")
	}

	// Clean up
	handler.Stop()
}

func TestOutputHandlerStop(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "stop-topic",
		BatchSize:         5,
		FlushTimeout:      100 * time.Millisecond,
		ChannelBufferSize: 10,
	}
	mockLogger := createTestLogger()

	handler := NewOutputHandler(config, mockLogger, nil)

	// Start and then stop
	err := handler.Start()
	if err != nil {
		t.Fatalf("Expected no error starting, got %v", err)
	}

	err = handler.Stop()
	if err != nil {
		t.Fatalf("Expected no error stopping output handler, got %v", err)
	}

	// Note: Producer closure is now handled internally by OutputHandler
}

func TestOutputHandlerBatching(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "batch-topic",
		BatchSize:         3, // Small batch for easy testing
		FlushTimeout:      100 * time.Millisecond,
		ChannelBufferSize: 10,
	}
	mockLogger := createTestLogger()

	handler := NewOutputHandler(config, mockLogger, nil)
	err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer handler.Stop()

	// Send multiple messages to trigger batching
	channel := handler.GetOutputChannel()
	for i := 0; i < 5; i++ {
		testData := []byte("test message")
		testMsg := models.NewDataMessage(testData, "test", 0)
		channel <- testMsg
	}

	// Wait for batching to occur
	time.Sleep(200 * time.Millisecond)

	// Since producer is now internal, we test that the handler
	// can process messages without errors
	// Note: Integration tests should verify actual message sending
}
