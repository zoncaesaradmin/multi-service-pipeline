package processing

import (
	"servicegomodule/internal/models"
	"context"
	"encoding/json"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"testing"
	"time"
)

// mockLogger implements the logging.Logger interface for testing
type mockLogger struct{}

func (m *mockLogger) SetLevel(level logging.Level)                                       { /* mock */ }
func (m *mockLogger) GetLevel() logging.Level                                            { return logging.InfoLevel }
func (m *mockLogger) IsLevelEnabled(level logging.Level) bool                            { return true }
func (m *mockLogger) Debug(msg string)                                                   { /* mock */ }
func (m *mockLogger) Info(msg string)                                                    { /* mock */ }
func (m *mockLogger) Warn(msg string)                                                    { /* mock */ }
func (m *mockLogger) Error(msg string)                                                   { /* mock */ }
func (m *mockLogger) Fatal(msg string)                                                   { /* mock */ }
func (m *mockLogger) Panic(msg string)                                                   { /* mock */ }
func (m *mockLogger) Debugf(format string, args ...interface{})                          { /* mock */ }
func (m *mockLogger) Infof(format string, args ...interface{})                           { /* mock */ }
func (m *mockLogger) Warnf(format string, args ...interface{})                           { /* mock */ }
func (m *mockLogger) Errorf(format string, args ...interface{})                          { /* mock */ }
func (m *mockLogger) Fatalf(format string, args ...interface{})                          { /* mock */ }
func (m *mockLogger) Panicf(format string, args ...interface{})                          { /* mock */ }
func (m *mockLogger) Debugw(msg string, fields ...interface{})                           { /* mock */ }
func (m *mockLogger) Infow(msg string, fields ...interface{})                            { /* mock */ }
func (m *mockLogger) Warnw(msg string, fields ...interface{})                            { /* mock */ }
func (m *mockLogger) Errorw(msg string, fields ...interface{})                           { /* mock */ }
func (m *mockLogger) Fatalw(msg string, fields ...interface{})                           { /* mock */ }
func (m *mockLogger) Panicw(msg string, fields ...interface{})                           { /* mock */ }
func (m *mockLogger) WithFields(fields logging.Fields) logging.Logger                    { return m }
func (m *mockLogger) WithField(key string, value interface{}) logging.Logger             { return m }
func (m *mockLogger) WithError(err error) logging.Logger                                 { return m }
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger                     { return m }
func (m *mockLogger) Log(level logging.Level, msg string)                                { /* mock */ }
func (m *mockLogger) Logf(level logging.Level, format string, args ...interface{})       { /* mock */ }
func (m *mockLogger) Logw(level logging.Level, msg string, keysAndValues ...interface{}) { /* mock */ }
func (m *mockLogger) Clone() logging.Logger                                              { return &mockLogger{} }
func (m *mockLogger) Close() error                                                       { return nil }

// mockProducer implements the messagebus.Producer interface for testing
type mockProducer struct {
	sentMessages []*messagebus.Message
	closed       bool
	sendError    error
}

func (m *mockProducer) Send(ctx context.Context, message *messagebus.Message) (partition int32, offset int64, err error) {
	if m.sendError != nil {
		return 0, 0, m.sendError
	}
	m.sentMessages = append(m.sentMessages, message)
	return 0, int64(len(m.sentMessages)), nil
}

func (m *mockProducer) SendAsync(ctx context.Context, message *messagebus.Message) <-chan messagebus.SendResult {
	resultCh := make(chan messagebus.SendResult, 1)
	go func() {
		if m.sendError != nil {
			resultCh <- messagebus.SendResult{Error: m.sendError}
		} else {
			m.sentMessages = append(m.sentMessages, message)
			resultCh <- messagebus.SendResult{
				Partition: 0,
				Offset:    int64(len(m.sentMessages)),
				Error:     nil,
			}
		}
		close(resultCh)
	}()
	return resultCh
}

func (m *mockProducer) Close() error {
	m.closed = true
	return nil
}

func TestConfigValidation(t *testing.T) {
	config := DefaultConfig(nil)
	err := ValidateConfig(config)
	if err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}
}

func TestNewProcessor(t *testing.T) {
	config := ProcessorConfig{
		ProcessingDelay: 10 * time.Millisecond,
		BatchSize:       100,
	}
	logger := &mockLogger{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}
	if processor.config.ProcessingDelay != config.ProcessingDelay {
		t.Errorf("Expected processing delay %v, got %v", config.ProcessingDelay, processor.config.ProcessingDelay)
	}
}

func TestProcessorApplyProcessing(t *testing.T) {
	config := ProcessorConfig{
		ProcessingDelay: 1 * time.Millisecond,
		BatchSize:       100,
	}
	logger := &mockLogger{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	input := ProcessingRecord{
		ID:        "test-record",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"string_field": "test_value",
			"number_field": 42,
		},
		Metadata: map[string]string{
			"source": "test",
		},
	}

	result, err := processor.applyProcessing(input)
	if err != nil {
		t.Fatalf("Expected no error applying processing, got %v", err)
	}

	if result.ID != "test-record" {
		t.Errorf("Expected ID to be test-record, got %s", result.ID)
	}
	if result.Data["string_field"] != "processed_test_value" {
		t.Errorf("Expected string_field to be processed_test_value, got %v", result.Data["string_field"])
	}
}

func TestNewOutputHandler(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "output-topic",
		BatchSize:         10,
		FlushTimeout:      5 * time.Second,
		ChannelBufferSize: 100,
	}
	logger := &mockLogger{}

	handler := NewOutputHandler(config, logger)

	if handler == nil {
		t.Fatal("Expected output handler to be created, got nil")
	}
	if handler.config.OutputTopic != config.OutputTopic {
		t.Errorf("Expected output topic %s, got %s", config.OutputTopic, handler.config.OutputTopic)
	}
}

func TestProcessingRecord(t *testing.T) {
	record := ProcessingRecord{
		ID:        "test-id",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"key": "value"},
		Metadata:  map[string]string{"source": "test"},
	}

	if record.ID != "test-id" {
		t.Errorf("Expected ID to be test-id, got %s", record.ID)
	}
	if record.Data["key"] != "value" {
		t.Errorf("Expected data key to be value, got %v", record.Data["key"])
	}
}

func TestProcessingRecordJSON(t *testing.T) {
	original := ProcessingRecord{
		ID:        "test-record-id",
		Timestamp: time.Now().UTC(),
		Data:      map[string]interface{}{"key": "value", "num": 42},
		Metadata:  map[string]string{"source": "test", "version": "1.0"},
	}

	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal ProcessingRecord: %v", err)
	}

	var unmarshaled ProcessingRecord
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ProcessingRecord: %v", err)
	}

	if unmarshaled.ID != original.ID {
		t.Errorf("Expected ID %s, got %s", original.ID, unmarshaled.ID)
	}
}

func TestProcessorGetStats(t *testing.T) {
	config := ProcessorConfig{
		ProcessingDelay: 1 * time.Millisecond,
		BatchSize:       100,
	}
	logger := &mockLogger{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	stats := processor.GetStats()
	if stats == nil {
		t.Error("Expected stats to be returned")
	}

	if _, ok := stats["status"]; !ok {
		t.Error("Expected status in stats")
	}
	if _, ok := stats["batch_size"]; !ok {
		t.Error("Expected batch_size in stats")
	}
	if _, ok := stats["processing_delay"]; !ok {
		t.Error("Expected processing_delay in stats")
	}
}

func TestOutputHandlerGetStats(t *testing.T) {
	config := OutputConfig{
		OutputTopic:       "output-topic",
		BatchSize:         10,
		FlushTimeout:      5 * time.Second,
		ChannelBufferSize: 100,
	}
	logger := &mockLogger{}

	handler := NewOutputHandler(config, logger)

	stats := handler.GetStats()
	if stats == nil {
		t.Error("Expected stats to be returned")
	}

	if _, ok := stats["status"]; !ok {
		t.Error("Expected status in stats")
	}
	if _, ok := stats["output_topic"]; !ok {
		t.Error("Expected output_topic in stats")
	}
	if _, ok := stats["batch_size"]; !ok {
		t.Error("Expected batch_size in stats")
	}
	if _, ok := stats["flush_timeout"]; !ok {
		t.Error("Expected flush_timeout in stats")
	}
}

func TestProcessorApplyProcessingEdgeCases(t *testing.T) {
	config := ProcessorConfig{
		ProcessingDelay: 1 * time.Millisecond,
		BatchSize:       100,
	}
	logger := &mockLogger{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	// Test with empty data
	input := ProcessingRecord{
		ID:        "empty-data",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
		Metadata:  map[string]string{},
	}

	result, err := processor.applyProcessing(input)
	if err != nil {
		t.Fatalf("Expected no error applying processing to empty data, got %v", err)
	}
	if result.ID != "empty-data" {
		t.Errorf("Expected ID to be empty-data, got %s", result.ID)
	}

	// Test with nil values
	inputWithNil := ProcessingRecord{
		ID:        "nil-test",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"nil_field": nil},
		Metadata:  map[string]string{"nil_meta": ""},
	}

	result2, err := processor.applyProcessing(inputWithNil)
	if err != nil {
		t.Fatalf("Expected no error applying processing to nil values, got %v", err)
	}
	if result2.ID != "nil-test" {
		t.Errorf("Expected ID to be nil-test, got %s", result2.ID)
	}
}

func TestSimpleNewPipeline(t *testing.T) {
	config := DefaultConfig(nil)
	logger := &mockLogger{}

	pipeline := NewPipeline(config, logger)

	if pipeline == nil {
		t.Fatal("Expected pipeline to be created with default config")
	}
}

func TestProcessingRecordValidation(t *testing.T) {
	// Test with various data types
	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{"string_data", map[string]interface{}{"key": "string_value"}},
		{"int_data", map[string]interface{}{"key": 42}},
		{"float_data", map[string]interface{}{"key": 3.14}},
		{"bool_data", map[string]interface{}{"key": true}},
		{"array_data", map[string]interface{}{"key": []string{"a", "b", "c"}}},
		{"nested_data", map[string]interface{}{"key": map[string]interface{}{"nested": "value"}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record := ProcessingRecord{
				ID:        tc.name,
				Timestamp: time.Now(),
				Data:      tc.data,
				Metadata:  map[string]string{"test": "value"},
			}

			if record.ID != tc.name {
				t.Errorf("Expected ID to be %s, got %s", tc.name, record.ID)
			}
			if record.Data == nil {
				t.Error("Expected data to not be nil")
			}
		})
	}
}
