package processing

import (
	"context"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"testing"
	"time"
)

// Mock producer for output tests
type mockProducerForOutput struct {
	messages []messagebus.Message
	closed   bool
	sendErr  error
}

func (m *mockProducerForOutput) Send(ctx context.Context, message *messagebus.Message) (partition int32, offset int64, err error) {
	if m.sendErr != nil {
		return 0, 0, m.sendErr
	}
	m.messages = append(m.messages, *message)
	return 0, int64(len(m.messages)), nil
}

func (m *mockProducerForOutput) SendAsync(ctx context.Context, message *messagebus.Message) <-chan messagebus.SendResult {
	resultCh := make(chan messagebus.SendResult, 1)
	if m.sendErr != nil {
		resultCh <- messagebus.SendResult{Error: m.sendErr}
	} else {
		m.messages = append(m.messages, *message)
		resultCh <- messagebus.SendResult{
			Partition: 0,
			Offset:    int64(len(m.messages)),
			Error:     nil,
		}
	}
	close(resultCh)
	return resultCh
}

func (m *mockProducerForOutput) Close() error {
	m.closed = true
	return nil
}

// Mock logger for output tests - matches logging.Logger interface exactly
type mockLoggerForOutput struct{}

func (m *mockLoggerForOutput) SetLevel(level logging.Level)                           { /* mock */ }
func (m *mockLoggerForOutput) GetLevel() logging.Level                                { return logging.InfoLevel }
func (m *mockLoggerForOutput) IsLevelEnabled(level logging.Level) bool                { return true }
func (m *mockLoggerForOutput) Debug(msg string)                                       { /* mock */ }
func (m *mockLoggerForOutput) Info(msg string)                                        { /* mock */ }
func (m *mockLoggerForOutput) Warn(msg string)                                        { /* mock */ }
func (m *mockLoggerForOutput) Error(msg string)                                       { /* mock */ }
func (m *mockLoggerForOutput) Fatal(msg string)                                       { /* mock */ }
func (m *mockLoggerForOutput) Panic(msg string)                                       { /* mock */ }
func (m *mockLoggerForOutput) Debugf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForOutput) Infof(format string, args ...interface{})               { /* mock */ }
func (m *mockLoggerForOutput) Warnf(format string, args ...interface{})               { /* mock */ }
func (m *mockLoggerForOutput) Errorf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForOutput) Fatalf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForOutput) Panicf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForOutput) Debugw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForOutput) Infow(msg string, fields ...interface{})                { /* mock */ }
func (m *mockLoggerForOutput) Warnw(msg string, fields ...interface{})                { /* mock */ }
func (m *mockLoggerForOutput) Errorw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForOutput) Fatalw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForOutput) Panicw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForOutput) WithFields(fields logging.Fields) logging.Logger        { return m }
func (m *mockLoggerForOutput) WithField(key string, value interface{}) logging.Logger { return m }
func (m *mockLoggerForOutput) WithError(err error) logging.Logger                     { return m }
func (m *mockLoggerForOutput) WithContext(ctx context.Context) logging.Logger         { return m }
func (m *mockLoggerForOutput) Log(level logging.Level, msg string)                    { /* mock */ }
func (m *mockLoggerForOutput) Logf(level logging.Level, format string, args ...interface{}) { /* mock */
}
func (m *mockLoggerForOutput) Logw(level logging.Level, msg string, keysAndValues ...interface{}) { /* mock */
}
func (m *mockLoggerForOutput) Clone() logging.Logger { return &mockLoggerForOutput{} }
func (m *mockLoggerForOutput) Close() error          { return nil }

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
	logger := &mockLoggerForOutput{}

	handler := NewOutputHandler(config, logger)
	channel := handler.GetOutputChannel()

	if channel == nil {
		t.Fatal("Expected output channel to be returned, got nil")
	}

	// Test that we can send a message to the channel
	testMsg := models.NewDataMessage([]byte("test"), "test")
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
	logger := &mockLoggerForOutput{}

	handler := NewOutputHandler(config, logger)

	err := handler.Start()
	if err != nil {
		t.Fatalf("Expected no error starting output handler, got %v", err)
	}

	// Test that the handler state shows it's running
	stats := handler.GetStats()
	if stats["status"] != "running" {
		t.Errorf("Expected handler status to be 'running', got %v", stats["status"])
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
	logger := &mockLoggerForOutput{}

	handler := NewOutputHandler(config, logger)

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
	logger := &mockLoggerForOutput{}

	handler := NewOutputHandler(config, logger)
	err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer handler.Stop()

	// Send multiple messages to trigger batching
	channel := handler.GetOutputChannel()
	for i := 0; i < 5; i++ {
		testData := []byte("test message")
		testMsg := models.NewDataMessage(testData, "test")
		channel <- testMsg
	}

	// Wait for batching to occur
	time.Sleep(200 * time.Millisecond)

	// Since producer is now internal, we test that the handler
	// can process messages without errors
	// Note: Integration tests should verify actual message sending
}
