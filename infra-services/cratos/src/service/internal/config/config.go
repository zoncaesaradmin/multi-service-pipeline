package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"sharedgomodule/logging"
	"sharedgomodule/utils"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type RawConfig struct {
	Server     RawServerConfig     `yaml:"server"`
	Logging    RawLoggingConfig    `yaml:"logging"`
	Processing RawProcessingConfig `yaml:"processing"`
}

// ServerConfig holds server-related configuration
type RawServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	ReadTimeout  int    `yaml:"readTimeout"`
	WriteTimeout int    `yaml:"writeTimeout"`
}

// LoggingConfig holds logging-related configuration
type RawLoggingConfig struct {
	Level       string `yaml:"level"`       // Log level: debug, info, warn, error, fatal, panic
	FileName    string `yaml:"fileName"`    // Path to the log file
	LoggerName  string `yaml:"loggerName"`  // Name identifier for the logger
	ServiceName string `yaml:"serviceName"` // Service name for structured logging
}

// ProcessingConfig holds processing pipeline configuration
type RawProcessingConfig struct {
	Input         RawInputConfig     `yaml:"input"`
	Processor     RawProcessorConfig `yaml:"processor"`
	Output        RawOutputConfig    `yaml:"output"`
	Channels      RawChannelConfig   `yaml:"channels"`
	PloggerConfig RawLoggingConfig   `yaml:"logging"`
}

// InputConfig holds input handler configuration
type RawInputConfig struct {
	Topics            []string      `yaml:"topics"`
	PollTimeout       time.Duration `yaml:"pollTimeout"`
	ChannelBufferSize int           `yaml:"channelBufferSize"`
}

// ProcessorConfig holds processor configuration
type RawProcessorConfig struct {
	ProcessingDelay time.Duration `yaml:"processingDelay"`
	BatchSize       int           `yaml:"batchSize"`
}

// OutputConfig holds output handler configuration
type RawOutputConfig struct {
	OutputTopic       string        `yaml:"outputTopic"`
	BatchSize         int           `yaml:"batchSize"`
	FlushTimeout      time.Duration `yaml:"flushTimeout"`
	ChannelBufferSize int           `yaml:"channelBufferSize"`
}

// ChannelConfig holds channel buffer configuration
type RawChannelConfig struct {
	InputBufferSize  int `yaml:"inputBufferSize"`
	OutputBufferSize int `yaml:"outputBufferSize"`
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *RawConfig {
	config := &RawConfig{
		Server: RawServerConfig{
			Host:         utils.GetEnv("SERVER_HOST", "localhost"),
			Port:         utils.GetEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  utils.GetEnvInt("SERVER_READ_TIMEOUT", 10),
			WriteTimeout: utils.GetEnvInt("SERVER_WRITE_TIMEOUT", 10),
		},
		Logging: RawLoggingConfig{
			Level:       utils.GetEnv("LOG_LEVEL", "info"),
			FileName:    utils.GetEnv("LOG_FILE_NAME", "main.log"),
			LoggerName:  utils.GetEnv("LOG_LOGGER_NAME", "main"),
			ServiceName: utils.GetEnv("LOG_SERVICE_NAME", "cratos"),
		},
		Processing: RawProcessingConfig{
			Input: RawInputConfig{
				Topics:            parseTopics(utils.GetEnv("PROCESSING_INPUT_TOPICS", "input-topic")),
				PollTimeout:       time.Duration(utils.GetEnvInt("PROCESSING_INPUT_POLL_TIMEOUT_MS", 1000)) * time.Millisecond,
				ChannelBufferSize: utils.GetEnvInt("PROCESSING_INPUT_BUFFER_SIZE", 1000),
			},
			Processor: RawProcessorConfig{
				ProcessingDelay: time.Duration(utils.GetEnvInt("PROCESSING_DELAY_MS", 10)) * time.Millisecond,
				BatchSize:       utils.GetEnvInt("PROCESSING_BATCH_SIZE", 100),
			},
			Output: RawOutputConfig{
				OutputTopic:       utils.GetEnv("PROCESSING_OUTPUT_TOPIC", "output-topic"),
				BatchSize:         utils.GetEnvInt("PROCESSING_OUTPUT_BATCH_SIZE", 50),
				FlushTimeout:      time.Duration(utils.GetEnvInt("PROCESSING_OUTPUT_FLUSH_TIMEOUT_MS", 5000)) * time.Millisecond,
				ChannelBufferSize: utils.GetEnvInt("PROCESSING_OUTPUT_BUFFER_SIZE", 1000),
			},
			Channels: RawChannelConfig{
				InputBufferSize:  utils.GetEnvInt("PROCESSING_CHANNELS_INPUT_BUFFER_SIZE", 1000),
				OutputBufferSize: utils.GetEnvInt("PROCESSING_CHANNELS_OUTPUT_BUFFER_SIZE", 1000),
			},
			PloggerConfig: RawLoggingConfig{
				Level:       utils.GetEnv("PROCESSING_PLOGGER_LEVEL", "info"),
				FileName:    utils.GetEnv("PROCESSING_PLOGGER_FILE_NAME", "/tmp/cratos-pipeline.log"),
				LoggerName:  utils.GetEnv("PROCESSING_PLOGGER_LOGGER_NAME", "pipeline"),
				ServiceName: utils.GetEnv("PROCESSING_PLOGGER_SERVICE_NAME", "cratos"),
			},
		},
	}

	return config
}

// parseTopics parses comma-separated topics from a string
func parseTopics(topicsStr string) []string {
	if topicsStr == "" {
		return []string{}
	}
	topics := strings.Split(topicsStr, ",")
	for i, topic := range topics {
		topics[i] = strings.TrimSpace(topic)
	}
	return topics
}

// LoadConfigFromFile loads configuration from a YAML file with optional environment variable overrides
func LoadConfigFromFile(configPath string) (*RawConfig, error) {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %w", configPath, err)
	}

	// Parse YAML
	config := &RawConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing YAML config file %s: %w", configPath, err)
	}

	// Override with environment variables if they exist
	overrideWithEnvVars(config)

	return config, nil
}

// LoadConfigWithDefaults loads configuration from file if it exists, falling back to environment variables and defaults
func LoadConfigWithDefaults(configPath string) *RawConfig {
	// Try to load from file first
	if config, err := LoadConfigFromFile(configPath); err == nil {
		return config
	}

	// Fallback to environment variables and defaults
	return LoadConfig()
}

// overrideWithEnvVars overrides config values with environment variables if they are set
func overrideWithEnvVars(config *RawConfig) {
	// Server configuration overrides
	if host := utils.GetEnv("SERVER_HOST", ""); host != "" {
		config.Server.Host = host
	}
	if port := utils.GetEnvInt("SERVER_PORT", -1); port != -1 {
		config.Server.Port = port
	}
	if readTimeout := utils.GetEnvInt("SERVER_READ_TIMEOUT", -1); readTimeout != -1 {
		config.Server.ReadTimeout = readTimeout
	}
	if writeTimeout := utils.GetEnvInt("SERVER_WRITE_TIMEOUT", -1); writeTimeout != -1 {
		config.Server.WriteTimeout = writeTimeout
	}

	// Logging configuration overrides
	if level := utils.GetEnv("LOG_LEVEL", ""); level != "" {
		config.Logging.Level = level
	}
	if fileName := utils.GetEnv("LOG_FILE_NAME", ""); fileName != "" {
		config.Logging.FileName = fileName
	}
	if loggerName := utils.GetEnv("LOG_LOGGER_NAME", ""); loggerName != "" {
		config.Logging.LoggerName = loggerName
	}
	if serviceName := utils.GetEnv("LOG_SERVICE_NAME", ""); serviceName != "" {
		config.Logging.ServiceName = serviceName
	}

	// Processing configuration overrides
	if topics := utils.GetEnv("PROCESSING_INPUT_TOPICS", ""); topics != "" {
		config.Processing.Input.Topics = parseTopics(topics)
	}
	if pollTimeout := utils.GetEnvInt("PROCESSING_INPUT_POLL_TIMEOUT_MS", -1); pollTimeout != -1 {
		config.Processing.Input.PollTimeout = time.Duration(pollTimeout) * time.Millisecond
	}
	if bufferSize := utils.GetEnvInt("PROCESSING_INPUT_BUFFER_SIZE", -1); bufferSize != -1 {
		config.Processing.Input.ChannelBufferSize = bufferSize
	}
	if delay := utils.GetEnvInt("PROCESSING_DELAY_MS", -1); delay != -1 {
		config.Processing.Processor.ProcessingDelay = time.Duration(delay) * time.Millisecond
	}
	if batchSize := utils.GetEnvInt("PROCESSING_BATCH_SIZE", -1); batchSize != -1 {
		config.Processing.Processor.BatchSize = batchSize
	}
	if outputTopic := utils.GetEnv("PROCESSING_OUTPUT_TOPIC", ""); outputTopic != "" {
		config.Processing.Output.OutputTopic = outputTopic
	}
	if outputBatchSize := utils.GetEnvInt("PROCESSING_OUTPUT_BATCH_SIZE", -1); outputBatchSize != -1 {
		config.Processing.Output.BatchSize = outputBatchSize
	}
	if flushTimeout := utils.GetEnvInt("PROCESSING_OUTPUT_FLUSH_TIMEOUT_MS", -1); flushTimeout != -1 {
		config.Processing.Output.FlushTimeout = time.Duration(flushTimeout) * time.Millisecond
	}
	if outputBufferSize := utils.GetEnvInt("PROCESSING_OUTPUT_BUFFER_SIZE", -1); outputBufferSize != -1 {
		config.Processing.Output.ChannelBufferSize = outputBufferSize
	}
	if inputBufferSize := utils.GetEnvInt("PROCESSING_CHANNELS_INPUT_BUFFER_SIZE", -1); inputBufferSize != -1 {
		config.Processing.Channels.InputBufferSize = inputBufferSize
	}
	if outputChannelBufferSize := utils.GetEnvInt("PROCESSING_CHANNELS_OUTPUT_BUFFER_SIZE", -1); outputChannelBufferSize != -1 {
		config.Processing.Channels.OutputBufferSize = outputChannelBufferSize
	}

	// Pipeline logger configuration overrides
	if ploggerLevel := utils.GetEnv("PROCESSING_PLOGGER_LEVEL", ""); ploggerLevel != "" {
		config.Processing.PloggerConfig.Level = ploggerLevel
	}
	if ploggerFileName := utils.GetEnv("PROCESSING_PLOGGER_FILE_NAME", ""); ploggerFileName != "" {
		config.Processing.PloggerConfig.FileName = ploggerFileName
	}
	if ploggerLoggerName := utils.GetEnv("PROCESSING_PLOGGER_LOGGER_NAME", ""); ploggerLoggerName != "" {
		config.Processing.PloggerConfig.LoggerName = ploggerLoggerName
	}
	if ploggerServiceName := utils.GetEnv("PROCESSING_PLOGGER_SERVICE_NAME", ""); ploggerServiceName != "" {
		config.Processing.PloggerConfig.ServiceName = ploggerServiceName
	}
}

// convertLogLevel converts a string log level to logging.Level
func convertLogLevel(levelStr string) logging.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return logging.DebugLevel
	case "info":
		return logging.InfoLevel
	case "warn":
		return logging.WarnLevel
	case "error":
		return logging.ErrorLevel
	case "fatal":
		return logging.FatalLevel
	case "panic":
		return logging.PanicLevel
	default:
		return logging.InfoLevel // Default to info if invalid level
	}
}

// ConvertLoggingConfig converts LoggingConfig to logging.LoggerConfig
func (cfg RawLoggingConfig) ConvertToLoggerConfig() logging.LoggerConfig {
	return logging.LoggerConfig{
		Level:       convertLogLevel(cfg.Level),
		FilePath:    cfg.FileName,
		LoggerName:  cfg.LoggerName,
		ServiceName: cfg.ServiceName,
	}
}
