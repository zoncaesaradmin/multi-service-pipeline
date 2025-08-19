package processing

import (
	"context"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"testing"
	"time"
)

// mockConsumer implements the messagebus.Consumer interface for testing
type mockConsumer struct {
	subscribedTopics []string
	pollMessage      *messagebus.Message
	pollError        error
	commitError      error
	closeError       error
	closed           bool
	subscribeError   error
}

func (m *mockConsumer) Subscribe(topics []string) error {
	if m.subscribeError != nil {
		return m.subscribeError
	}
	m.subscribedTopics = topics
	return nil
}

func (m *mockConsumer) Poll(timeout time.Duration) (*messagebus.Message, error) {
	if m.pollError != nil {
		return nil, m.pollError
	}
	return m.pollMessage, nil
}

func (m *mockConsumer) Commit(ctx context.Context, message *messagebus.Message) error {
	return m.commitError
}

func (m *mockConsumer) Close() error {
	m.closed = true
	return m.closeError
}

// mockProducerForInput implements the messagebus.Producer interface for input tests
type mockProducerForInput struct{}

func (m *mockProducerForInput) Send(ctx context.Context, message *messagebus.Message) (partition int32, offset int64, err error) {
	return 0, 0, nil
}

func (m *mockProducerForInput) SendAsync(ctx context.Context, message *messagebus.Message) <-chan messagebus.SendResult {
	resultCh := make(chan messagebus.SendResult, 1)
	resultCh <- messagebus.SendResult{Partition: 0, Offset: 0, Error: nil}
	close(resultCh)
	return resultCh
}

func (m *mockProducerForInput) Close() error {
	return nil
}

// mockLoggerForInput implements the logging.Logger interface for input tests
type mockLoggerForInput struct{}

func (m *mockLoggerForInput) SetLevel(level logging.Level)                           { /* mock */ }
func (m *mockLoggerForInput) GetLevel() logging.Level                                { return logging.InfoLevel }
func (m *mockLoggerForInput) IsLevelEnabled(level logging.Level) bool                { return true }
func (m *mockLoggerForInput) Debug(msg string)                                       { /* mock */ }
func (m *mockLoggerForInput) Info(msg string)                                        { /* mock */ }
func (m *mockLoggerForInput) Warn(msg string)                                        { /* mock */ }
func (m *mockLoggerForInput) Error(msg string)                                       { /* mock */ }
func (m *mockLoggerForInput) Fatal(msg string)                                       { /* mock */ }
func (m *mockLoggerForInput) Panic(msg string)                                       { /* mock */ }
func (m *mockLoggerForInput) Debugf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForInput) Infof(format string, args ...interface{})               { /* mock */ }
func (m *mockLoggerForInput) Warnf(format string, args ...interface{})               { /* mock */ }
func (m *mockLoggerForInput) Errorf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForInput) Fatalf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForInput) Panicf(format string, args ...interface{})              { /* mock */ }
func (m *mockLoggerForInput) Debugw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForInput) Infow(msg string, fields ...interface{})                { /* mock */ }
func (m *mockLoggerForInput) Warnw(msg string, fields ...interface{})                { /* mock */ }
func (m *mockLoggerForInput) Errorw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForInput) Fatalw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForInput) Panicw(msg string, fields ...interface{})               { /* mock */ }
func (m *mockLoggerForInput) WithFields(fields logging.Fields) logging.Logger        { return m }
func (m *mockLoggerForInput) WithField(key string, value interface{}) logging.Logger { return m }
func (m *mockLoggerForInput) WithError(err error) logging.Logger                     { return m }
func (m *mockLoggerForInput) WithContext(ctx context.Context) logging.Logger         { return m }
func (m *mockLoggerForInput) Log(level logging.Level, msg string)                    { /* mock */ }
func (m *mockLoggerForInput) Logf(level logging.Level, format string, args ...interface{}) { /* mock */
}
func (m *mockLoggerForInput) Logw(level logging.Level, msg string, keysAndValues ...interface{}) { /* mock */
}
func (m *mockLoggerForInput) Clone() logging.Logger { return &mockLoggerForInput{} }
func (m *mockLoggerForInput) Close() error          { return nil }

func TestInputConfig(t *testing.T) {
	config := InputConfig{
		Topics:            []string{"test-topic-1", "test-topic-2"},
		PollTimeout:       5 * time.Second,
		ChannelBufferSize: 100,
	}

	if len(config.Topics) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(config.Topics))
	}
	if config.Topics[0] != "test-topic-1" {
		t.Errorf("Expected first topic to be 'test-topic-1', got %s", config.Topics[0])
	}
	if config.PollTimeout != 5*time.Second {
		t.Errorf("Expected poll timeout to be 5s, got %v", config.PollTimeout)
	}
	if config.ChannelBufferSize != 100 {
		t.Errorf("Expected channel buffer size to be 100, got %d", config.ChannelBufferSize)
	}
}

func TestNewInputHandler(t *testing.T) {
	config := InputConfig{
		Topics:            []string{"input-topic"},
		PollTimeout:       1 * time.Second,
		ChannelBufferSize: 50,
	}
	logger := &mockLoggerForInput{}

	handler := NewInputHandler(config, logger)

	if handler == nil {
		t.Fatal("Expected input handler to be created, got nil")
	}
	if len(handler.config.Topics) != 1 {
		t.Errorf("Expected 1 topic in config, got %d", len(handler.config.Topics))
	}
	if handler.config.Topics[0] != "input-topic" {
		t.Errorf("Expected topic to be 'input-topic', got %s", handler.config.Topics[0])
	}
	if handler.config.PollTimeout != 1*time.Second {
		t.Errorf("Expected poll timeout to be 1s, got %v", handler.config.PollTimeout)
	}
	if handler.config.ChannelBufferSize != 50 {
		t.Errorf("Expected buffer size to be 50, got %d", handler.config.ChannelBufferSize)
	}
}

func TestInputHandlerGetInputChannel(t *testing.T) {
	config := InputConfig{
		Topics:            []string{"test-topic"},
		PollTimeout:       1 * time.Second,
		ChannelBufferSize: 10,
	}
	logger := &mockLoggerForInput{}

	handler := NewInputHandler(config, logger)
	channel := handler.GetInputChannel()

	if channel == nil {
		t.Fatal("Expected input channel to be available, got nil")
	}

	// Test that the channel is read-only
	select {
	case <-channel:
		// This is expected behavior - reading from empty channel should not block in select
	default:
		// Channel is empty, which is expected
	}
}

func TestInputHandlerGetStats(t *testing.T) {
	config := InputConfig{
		Topics:            []string{"topic1", "topic2"},
		PollTimeout:       3 * time.Second,
		ChannelBufferSize: 200,
	}
	logger := &mockLoggerForInput{}

	handler := NewInputHandler(config, logger)
	stats := handler.GetStats()

	if stats == nil {
		t.Fatal("Expected stats to be returned, got nil")
	}

	if status, ok := stats["status"]; !ok || status != "running" {
		t.Errorf("Expected status to be 'running', got %v", status)
	}

	if topics, ok := stats["topics"]; !ok {
		t.Error("Expected topics in stats")
	} else {
		topicsSlice, ok := topics.([]string)
		if !ok || len(topicsSlice) != 2 {
			t.Errorf("Expected 2 topics in stats, got %v", topics)
		}
	}

	if pollTimeout, ok := stats["poll_timeout"]; !ok {
		t.Error("Expected poll_timeout in stats")
	} else if pollTimeout != "3s" {
		t.Errorf("Expected poll_timeout to be '3s', got %v", pollTimeout)
	}

	if bufferSize, ok := stats["channel_buffer_size"]; !ok {
		t.Error("Expected channel_buffer_size in stats")
	} else if bufferSize != 200 {
		t.Errorf("Expected channel_buffer_size to be 200, got %v", bufferSize)
	}
}

func TestInputHandlerStartSuccess(t *testing.T) {
	config := InputConfig{
		Topics:            []string{"test-topic"},
		PollTimeout:       100 * time.Millisecond,
		ChannelBufferSize: 10,
	}
	logger := &mockLoggerForInput{}

	handler := NewInputHandler(config, logger)

	// Mock the consumer to avoid actual Kafka calls
	mockConsumer := &mockConsumer{}
	handler.consumer = mockConsumer

	err := handler.Start()
	if err != nil {
		t.Fatalf("Expected no error starting input handler, got %v", err)
	}

	// Verify consumer was subscribed to topics
	if len(mockConsumer.subscribedTopics) != 1 {
		t.Errorf("Expected 1 subscribed topic, got %d", len(mockConsumer.subscribedTopics))
	}
	if mockConsumer.subscribedTopics[0] != "test-topic" {
		t.Errorf("Expected subscribed topic to be 'test-topic', got %s", mockConsumer.subscribedTopics[0])
	}

	// Clean up
	handler.Stop()
}

func TestInputHandlerStop(t *testing.T) {
	config := InputConfig{
		Topics:            []string{"test-topic"},
		PollTimeout:       100 * time.Millisecond,
		ChannelBufferSize: 10,
	}
	logger := &mockLoggerForInput{}

	handler := NewInputHandler(config, logger)

	// Mock the consumer
	mockConsumer := &mockConsumer{}
	handler.consumer = mockConsumer

	// Start and then stop
	err := handler.Start()
	if err != nil {
		t.Fatalf("Expected no error starting, got %v", err)
	}

	err = handler.Stop()
	if err != nil {
		t.Fatalf("Expected no error stopping input handler, got %v", err)
	}

	// Verify consumer was closed
	if !mockConsumer.closed {
		t.Error("Expected consumer to be closed")
	}
}

func TestInputConfigEdgeCases(t *testing.T) {
	testCases := []struct {
		name   string
		config InputConfig
		valid  bool
	}{
		{
			"empty_topics",
			InputConfig{
				Topics:            []string{},
				PollTimeout:       1 * time.Second,
				ChannelBufferSize: 100,
			},
			true, // Empty topics slice is technically valid for the struct
		},
		{
			"zero_poll_timeout",
			InputConfig{
				Topics:            []string{"topic1"},
				PollTimeout:       0,
				ChannelBufferSize: 100,
			},
			true, // Zero timeout is valid - means no waiting
		},
		{
			"large_buffer_size",
			InputConfig{
				Topics:            []string{"topic1"},
				PollTimeout:       1 * time.Second,
				ChannelBufferSize: 10000,
			},
			true,
		},
		{
			"many_topics",
			InputConfig{
				Topics:            []string{"topic1", "topic2", "topic3", "topic4", "topic5"},
				PollTimeout:       5 * time.Second,
				ChannelBufferSize: 500,
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := &mockLoggerForInput{}

			handler := NewInputHandler(tc.config, logger)
			if handler == nil && tc.valid {
				t.Errorf("Expected valid config to create handler, got nil")
			}

			if handler != nil {
				// Verify config was set correctly
				if len(handler.config.Topics) != len(tc.config.Topics) {
					t.Errorf("Expected %d topics, got %d", len(tc.config.Topics), len(handler.config.Topics))
				}
				if handler.config.PollTimeout != tc.config.PollTimeout {
					t.Errorf("Expected poll timeout %v, got %v", tc.config.PollTimeout, handler.config.PollTimeout)
				}
				if handler.config.ChannelBufferSize != tc.config.ChannelBufferSize {
					t.Errorf("Expected buffer size %d, got %d", tc.config.ChannelBufferSize, handler.config.ChannelBufferSize)
				}
			}
		})
	}
}
