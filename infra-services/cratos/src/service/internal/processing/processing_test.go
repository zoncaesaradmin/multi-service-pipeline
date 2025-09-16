package processing

import (
	"context"
	"os"
	"path/filepath"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"testing"
	"time"
)

// Test constants to avoid duplication
const (
	testTopic              = "test-topic"
	processingBootstrapKey = "bootstrap.servers"
	testKafkaHost          = "localhost:9092"
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

func sampleRawConfig() ProcConfig {
	return ProcConfig{
		Input: InputConfig{
			Topics:            []string{"cisco_nir-anomalies"},
			PollTimeout:       1 * time.Second,
			ChannelBufferSize: 1000,
			KafkaConfigMap:    map[string]any{"client.id": "test-consumer"},
		},
		Processor: ProcessorConfig{
			ProcessingDelay: 10 * time.Millisecond,
			BatchSize:       100,
			RuleEngine: RuleEngineConfig{
				RulesTopic:  testTopic,
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
				RulesKafkaConfigMap:         map[string]any{processingBootstrapKey: testKafkaHost},
				RuleTasksConsKafkaConfigMap: map[string]any{processingBootstrapKey: testKafkaHost},
				RuleTasksProdKafkaConfigMap: map[string]any{processingBootstrapKey: testKafkaHost},
			},
		},
		Output: OutputConfig{
			OutputTopic:       "cisco_nir-prealerts",
			BatchSize:         50,
			FlushTimeout:      5 * time.Second,
			ChannelBufferSize: 1000,
			KafkaConfigMap:    map[string]any{"client.id": "test-producer"},
		},
		Channels: ChannelConfig{
			InputBufferSize:  1000,
			OutputBufferSize: 1000,
		},
		LoggerConfig: logging.LoggerConfig{
			Level:         logging.InfoLevel,
			FilePath:      "/tmp/cratos-pipeline.log",
			LoggerName:    "pipeline",
			ComponentName: "processing",
			ServiceName:   "cratos",
		},
	}
}

func TestConfigValidation(t *testing.T) {
	config := sampleRawConfig()
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
				RulesTopic:  testTopic,
				PollTimeout: 10 * time.Millisecond,
				RuleEngLibLogging: logging.LoggerConfig{
					Level:         logging.InfoLevel,
					FilePath:      testLogFile,
					LoggerName:    "ruleenginelib",
					ComponentName: "ruleenginelib",
					ServiceName:   "cratos",
				},
				RuleHandlerLogging: logging.LoggerConfig{
					Level:         logging.InfoLevel,
					FilePath:      testLogFile,
					LoggerName:    "rulehandler",
					ComponentName: "rulehandler",
					ServiceName:   "cratos",
				},
				RulesKafkaConfigMap:         map[string]any{processingBootstrapKey: testKafkaHost},
				RuleTasksConsKafkaConfigMap: map[string]any{processingBootstrapKey: testKafkaHost},
				RuleTasksProdKafkaConfigMap: map[string]any{processingBootstrapKey: testKafkaHost},
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
	processor := NewProcessor(procConfig.Processor, logger, inputCh, outputCh, nil)

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

	handler := NewOutputHandler(config, logger, nil)

	if handler == nil {
		t.Fatal("Expected output handler to be created, got nil")
	}
	if handler.config.OutputTopic != config.OutputTopic {
		t.Errorf("Expected output topic %s, got %s", config.OutputTopic, handler.config.OutputTopic)
	}
}

func TestProcessorGetStats(t *testing.T) {
	testLogFile := "/tmp/test.log"
	config := ProcessorConfig{
		ProcessingDelay: 10 * time.Millisecond,
		BatchSize:       100,
		RuleEngine: RuleEngineConfig{
			RulesTopic:  "test-topic",
			PollTimeout: 10 * time.Millisecond,
			RuleEngLibLogging: logging.LoggerConfig{
				Level:         logging.InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    "ruleenginelib",
				ComponentName: "ruleenginelib",
				ServiceName:   "cratos",
			},
			RuleHandlerLogging: logging.LoggerConfig{
				Level:         logging.InfoLevel,
				FilePath:      testLogFile,
				LoggerName:    "rulehandler",
				ComponentName: "rulehandler",
				ServiceName:   "cratos",
			},
			RulesKafkaConfigMap:         map[string]any{processingBootstrapKey: testKafkaHost},
			RuleTasksConsKafkaConfigMap: map[string]any{processingBootstrapKey: testKafkaHost},
			RuleTasksProdKafkaConfigMap: map[string]any{processingBootstrapKey: testKafkaHost},
		}}

	logger := &mockLogger{}
	inputCh := make(chan *models.ChannelMessage, 10)
	outputCh := make(chan *models.ChannelMessage, 10)

	processor := NewProcessor(config, logger, inputCh, outputCh, nil)

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

	handler := NewOutputHandler(config, logger, nil)

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
	err := os.WriteFile(kafkaConfigPath, []byte("bootstrap.servers: "+testKafkaHost+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp kafka config: %v", err)
	}

	config := sampleRawConfig()
	// Set KafkaConfigMap fields for test
	config.Input.KafkaConfigMap = map[string]any{processingBootstrapKey: testKafkaHost}
	config.Output.KafkaConfigMap = map[string]any{processingBootstrapKey: testKafkaHost}
	config.Processor.RuleEngine.RulesKafkaConfigMap = map[string]any{processingBootstrapKey: testKafkaHost}

	logger := &mockLogger{}
	pipeline := NewPipeline(config, logger, nil)

	if pipeline == nil {
		t.Fatal("Expected pipeline to be created with default config")
	}
}
