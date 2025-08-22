package processing

import (
	"context"
	"os"
	"path/filepath"
	"servicegomodule/internal/models"
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
	testLogFile := "/tmp/test.log"
	procConfig := ProcConfig{
		Processor: ProcessorConfig{
			ProcessingDelay: 10 * time.Millisecond,
			BatchSize:       100,
			RuleEngine: RuleEngineConfig{
				RulesTopic:  "test-topic",
				PollTimeout: 10 * time.Millisecond,
				Logging: logging.LoggerConfig{
					Level:         logging.InfoLevel,
					FilePath:      testLogFile,
					LoggerName:    "ruleengine",
					ComponentName: "ruleengine",
					ServiceName:   "cratos",
				},
				KafkaConfigMap: map[string]any{"bootstrap.servers": "localhost:9092"},
			},
		},
		LoggerConfig: logging.LoggerConfig{
			Level:         logging.InfoLevel,
			FilePath:      testLogFile,
			LoggerName:    "pipeline",
			ComponentName: "processing",
			ServiceName:   "cratos",
		},
	}
	logger, _ := logging.NewLogger(&procConfig.LoggerConfig)
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)
	processor := NewProcessor(procConfig.Processor, logger, inputCh, outputCh)
	inputCh = make(chan *models.ChannelMessage, 10)
	outputCh = make(chan *models.ChannelMessage, 10)

	processor = NewProcessor(procConfig.Processor, logger, inputCh, outputCh)

	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}
	if processor.config.ProcessingDelay != procConfig.Processor.ProcessingDelay {
		t.Errorf("Expected processing delay %v, got %v", procConfig.Processor.ProcessingDelay, processor.config.ProcessingDelay)
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

func TestSimpleNewPipeline(t *testing.T) {
	// Create a temporary Kafka config file
	tempDir := t.TempDir()
	kafkaConfigPath := filepath.Join(tempDir, "kafka-consumer.yaml")
	err := os.WriteFile(kafkaConfigPath, []byte("bootstrap.servers: localhost:9092\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp kafka config: %v", err)
	}

	config := DefaultConfig(nil)
	// Set KafkaConfigMap fields for test
	const kafkaBootstrapServers = "bootstrap.servers"
	const kafkaLocalhost9092 = "localhost:9092"
	config.Input.KafkaConfigMap = map[string]any{kafkaBootstrapServers: kafkaLocalhost9092}
	config.Output.KafkaConfigMap = map[string]any{kafkaBootstrapServers: kafkaLocalhost9092}
	config.Processor.RuleEngine.KafkaConfigMap = map[string]any{kafkaBootstrapServers: kafkaLocalhost9092}

	logger := &mockLogger{}
	pipeline := NewPipeline(config, logger)

	if pipeline == nil {
		t.Fatal("Expected pipeline to be created with default config")
	}
}
