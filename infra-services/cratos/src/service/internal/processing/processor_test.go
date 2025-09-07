package processing

import (
	"context"
	"testing"
	"time"

	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"telemetry/utils/alert"

	"google.golang.org/protobuf/proto"
)

const (
	bootstrapServersKey = "bootstrap.servers"
)

// mockLoggerForProcessor implements the logging.Logger interface for testing
type mockLoggerForProcessor struct{}

func (m *mockLoggerForProcessor) SetLevel(level logging.Level)            {}
func (m *mockLoggerForProcessor) GetLevel() logging.Level                 { return logging.InfoLevel }
func (m *mockLoggerForProcessor) IsLevelEnabled(level logging.Level) bool { return true }

func (m *mockLoggerForProcessor) Debug(msg string)                                             {}
func (m *mockLoggerForProcessor) Info(msg string)                                              {}
func (m *mockLoggerForProcessor) Warn(msg string)                                              {}
func (m *mockLoggerForProcessor) Error(msg string)                                             {}
func (m *mockLoggerForProcessor) Fatal(msg string)                                             {}
func (m *mockLoggerForProcessor) Panic(msg string)                                             {}
func (m *mockLoggerForProcessor) Debugf(format string, args ...interface{})                    {}
func (m *mockLoggerForProcessor) Infof(format string, args ...interface{})                     {}
func (m *mockLoggerForProcessor) Warnf(format string, args ...interface{})                     {}
func (m *mockLoggerForProcessor) Errorf(format string, args ...interface{})                    {}
func (m *mockLoggerForProcessor) Fatalf(format string, args ...interface{})                    {}
func (m *mockLoggerForProcessor) Panicf(format string, args ...interface{})                    {}
func (m *mockLoggerForProcessor) Debugw(msg string, keysAndValues ...interface{})              {}
func (m *mockLoggerForProcessor) Infow(msg string, keysAndValues ...interface{})               {}
func (m *mockLoggerForProcessor) Warnw(msg string, keysAndValues ...interface{})               {}
func (m *mockLoggerForProcessor) Errorw(msg string, keysAndValues ...interface{})              {}
func (m *mockLoggerForProcessor) Fatalw(msg string, keysAndValues ...interface{})              {}
func (m *mockLoggerForProcessor) Panicw(msg string, keysAndValues ...interface{})              {}
func (m *mockLoggerForProcessor) WithFields(fields logging.Fields) logging.Logger              { return m }
func (m *mockLoggerForProcessor) WithField(key string, value interface{}) logging.Logger       { return m }
func (m *mockLoggerForProcessor) WithError(err error) logging.Logger                           { return m }
func (m *mockLoggerForProcessor) WithContext(ctx context.Context) logging.Logger               { return m }
func (m *mockLoggerForProcessor) Log(level logging.Level, msg string)                          {}
func (m *mockLoggerForProcessor) Logf(level logging.Level, format string, args ...interface{}) {}
func (m *mockLoggerForProcessor) Logw(level logging.Level, msg string, keysAndValues ...interface{}) {
}
func (m *mockLoggerForProcessor) Clone() logging.Logger { return m }
func (m *mockLoggerForProcessor) Close() error          { return nil }

func sampleProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		ProcessingDelay: 10 * time.Millisecond,
		BatchSize:       100,
		RuleEngine: RuleEngineConfig{
			RulesTopic:  "test-topic",
			PollTimeout: 10 * time.Millisecond,
			RuleEngLibLogging: logging.LoggerConfig{
				Level:         logging.InfoLevel,
				FilePath:      "/tmp/ruleenginelib.log",
				LoggerName:    "ruleenginelib",
				ComponentName: "ruleenginelib",
				ServiceName:   "cratos",
			},
			RuleHandlerLogging: logging.LoggerConfig{
				Level:         logging.InfoLevel,
				FilePath:      "/tmp/rulehandler.log",
				LoggerName:    "rulehandler",
				ComponentName: "rulehandler",
				ServiceName:   "cratos",
			},
			RulesKafkaConfigMap:         map[string]any{bootstrapServersKey: "localhost:9092"},
			RuleTasksConsKafkaConfigMap: map[string]any{bootstrapServersKey: "localhost:9092"},
			RuleTasksProdKafkaConfigMap: map[string]any{bootstrapServersKey: "localhost:9092"},
		},
	}
}
func TestProcessorConfig(t *testing.T) {
	config := ProcessorConfig{
		ProcessingDelay: 10 * time.Millisecond,
		BatchSize:       100,
	}

	if config.ProcessingDelay != 10*time.Millisecond {
		t.Errorf("Expected ProcessingDelay to be 10ms, got %v", config.ProcessingDelay)
	}

	if config.BatchSize != 100 {
		t.Errorf("Expected BatchSize to be 100, got %d", config.BatchSize)
	}
}

func TestProcessorCreation(t *testing.T) {
	config := sampleProcessorConfig()
	logger := &mockLoggerForProcessor{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}

	if processor.config.ProcessingDelay != config.ProcessingDelay {
		t.Errorf("Expected config ProcessingDelay to be %v, got %v", config.ProcessingDelay, processor.config.ProcessingDelay)
	}

	if processor.config.BatchSize != config.BatchSize {
		t.Errorf("Expected config BatchSize to be %d, got %d", config.BatchSize, processor.config.BatchSize)
	}
}

func TestProcessorStatsRetrieval(t *testing.T) {
	config := sampleProcessorConfig()
	logger := &mockLoggerForProcessor{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	stats := processor.GetStats()
	if stats == nil {
		t.Error("Expected stats to be non-nil")
	}

	// Check that stats contain expected fields
	if _, exists := stats["status"]; !exists {
		t.Error("Expected stats to contain 'status' field")
	}

	if _, exists := stats["batch_size"]; !exists {
		t.Error("Expected stats to contain 'batch_size' field")
	}
}

func TestProcessorMessageFlow(t *testing.T) {
	config := sampleProcessorConfig()
	logger := &mockLoggerForProcessor{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Create a valid AlertStream protobuf message
	alertObj := &alert.Alert{
		AlertId:  "flow-test-1",
		StartTs:  "2023-01-01T00:00:00Z",
		Reason:   "unit test",
		Severity: "info",
	}
	aStream := &alert.AlertStream{
		AlertObject: []*alert.Alert{alertObj},
	}
	data, err := proto.Marshal(aStream)
	if err != nil {
		t.Fatalf("Failed to marshal AlertStream: %v", err)
	}
	inputMessage := models.NewDataMessage(data, "test", 0)

	// Send the message
	inputCh <- inputMessage

	// Wait for processing and check output
	select {
	case outputMessage := <-outputCh:
		if !outputMessage.IsDataMessage() {
			t.Error("Expected output message to be a data message")
		}
		// Basic validation that the message was processed
		if len(outputMessage.Data) == 0 {
			t.Error("Expected processed data to be non-empty")
		}

	case <-time.After(500 * time.Millisecond):
		t.Error("Timeout waiting for processed message (increase timeout or check processor logic)")
	}
}

func TestProcessorNonDataMessageForwarding(t *testing.T) {
	config := sampleProcessorConfig()
	logger := &mockLoggerForProcessor{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}
	defer processor.Stop()

	// Create a control message (non-data message)
	controlMessage := models.NewControlMessage([]byte("test-control"), "test")

	// Send the message
	inputCh <- controlMessage

	// Wait for forwarding and check output
	select {
	case outputMessage := <-outputCh:
		if outputMessage.IsDataMessage() {
			t.Error("Expected output message to be a control message")
		}

		if outputMessage.Type != "control" {
			t.Errorf("Expected message type to be 'control', got %s", outputMessage.Type)
		}

	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for forwarded message")
	}
}

func TestProcessorLifecycle(t *testing.T) {
	config := sampleProcessorConfig()
	logger := &mockLoggerForProcessor{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh)

	// Test start
	err := processor.Start()
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	// Test stop
	err = processor.Stop()
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}
}
