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
	Topics            []string `yaml:"topics"`
	ChannelBufferSize int      `yaml:"channelBufferSize"`
	KafkaConfFile     string   `yaml:"kafkaConfigFile"`
}

// ProcessorConfig holds processor configuration
type RawProcessorConfig struct {
	ProcessingDelay time.Duration          `yaml:"processingDelay"`
	BatchSize       int                    `yaml:"batchSize"`
	RuleProcConfig  RawRuleProcessorConfig `yaml:"ruleEngine"`
}

type RawRuleProcessorConfig struct {
	RulesTopic         string           `yaml:"rulesTopic"`
	RelibLogging       RawLoggingConfig `yaml:"ruleenginelibLogging"`
	RulesKafkaConfFile string           `yaml:"rulesKafkaConfigFile"`

	RuleTasksTopic         string           `yaml:"ruleTasksTopic"`
	RuleHandlerLogging     RawLoggingConfig `yaml:"ruleHandlerLogging"`
	RuleTasksConsKafkaFile string           `yaml:"ruleTasksConsKafkaConfigFile"`
	RuleTasksProdKafkaFile string           `yaml:"ruleTasksProdKafkaConfigFile"`
}

// OutputConfig holds output handler configuration
type RawOutputConfig struct {
	OutputTopic       string        `yaml:"outputTopic"`
	BatchSize         int           `yaml:"batchSize"`
	FlushTimeout      time.Duration `yaml:"flushTimeout"`
	ChannelBufferSize int           `yaml:"channelBufferSize"`
	KafkaConfFile     string        `yaml:"kafkaConfigFile"`
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
			Port:         utils.GetEnvInt("SERVER_PORT", 4477),
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
				Topics:            parseTopics(utils.GetEnv("PROCESSING_INPUT_TOPICS", "cisco_nir-anomalies")),
				ChannelBufferSize: utils.GetEnvInt("PROCESSING_INPUT_BUFFER_SIZE", 1000),
			},
			Processor: RawProcessorConfig{
				ProcessingDelay: time.Duration(utils.GetEnvInt("PROCESSING_DELAY_MS", 10)) * time.Millisecond,
				BatchSize:       utils.GetEnvInt("PROCESSING_BATCH_SIZE", 100),
			},
			Output: RawOutputConfig{
				OutputTopic:       utils.GetEnv("PROCESSING_OUTPUT_TOPIC", "cisco_nir-prealerts"),
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
	overrideServerConfig(&config.Server)
	overrideLoggingConfig(&config.Logging)
	overrideProcessingConfig(&config.Processing)
}

// overrideServerConfig overrides server configuration with environment variables
func overrideServerConfig(server *RawServerConfig) {
	if host := utils.GetEnv("SERVER_HOST", ""); host != "" {
		server.Host = host
	}
	if port := utils.GetEnvInt("SERVER_PORT", -1); port != -1 {
		server.Port = port
	}
	if readTimeout := utils.GetEnvInt("SERVER_READ_TIMEOUT", -1); readTimeout != -1 {
		server.ReadTimeout = readTimeout
	}
	if writeTimeout := utils.GetEnvInt("SERVER_WRITE_TIMEOUT", -1); writeTimeout != -1 {
		server.WriteTimeout = writeTimeout
	}
}

// overrideLoggingConfig overrides logging configuration with environment variables
func overrideLoggingConfig(logging *RawLoggingConfig) {
	if level := utils.GetEnv("LOG_LEVEL", ""); level != "" {
		logging.Level = level
	}
	if fileName := utils.GetEnv("LOG_FILE_NAME", ""); fileName != "" {
		logging.FileName = fileName
	}
	if loggerName := utils.GetEnv("LOG_LOGGER_NAME", ""); loggerName != "" {
		logging.LoggerName = loggerName
	}
	if serviceName := utils.GetEnv("LOG_SERVICE_NAME", ""); serviceName != "" {
		logging.ServiceName = serviceName
	}
}

// overrideProcessingConfig overrides processing configuration with environment variables
func overrideProcessingConfig(processing *RawProcessingConfig) {
	overrideInputConfig(&processing.Input)
	overrideProcessorConfig(&processing.Processor)
	overrideOutputConfig(&processing.Output)
	overrideChannelConfig(&processing.Channels)
	overridePipelineLoggerConfig(&processing.PloggerConfig)
}

// overrideInputConfig overrides input configuration with environment variables
func overrideInputConfig(input *RawInputConfig) {
	if topics := utils.GetEnv("PROCESSING_INPUT_TOPICS", ""); topics != "" {
		input.Topics = parseTopics(topics)
	}
	if bufferSize := utils.GetEnvInt("PROCESSING_INPUT_BUFFER_SIZE", -1); bufferSize != -1 {
		input.ChannelBufferSize = bufferSize
	}
}

// overrideProcessorConfig overrides processor configuration with environment variables
func overrideProcessorConfig(processor *RawProcessorConfig) {
	if delay := utils.GetEnvInt("PROCESSING_DELAY_MS", -1); delay != -1 {
		processor.ProcessingDelay = time.Duration(delay) * time.Millisecond
	}
	if batchSize := utils.GetEnvInt("PROCESSING_BATCH_SIZE", -1); batchSize != -1 {
		processor.BatchSize = batchSize
	}
}

// overrideOutputConfig overrides output configuration with environment variables
func overrideOutputConfig(output *RawOutputConfig) {
	if outputTopic := utils.GetEnv("PROCESSING_OUTPUT_TOPIC", ""); outputTopic != "" {
		output.OutputTopic = outputTopic
	}
	if outputBatchSize := utils.GetEnvInt("PROCESSING_OUTPUT_BATCH_SIZE", -1); outputBatchSize != -1 {
		output.BatchSize = outputBatchSize
	}
	if flushTimeout := utils.GetEnvInt("PROCESSING_OUTPUT_FLUSH_TIMEOUT_MS", -1); flushTimeout != -1 {
		output.FlushTimeout = time.Duration(flushTimeout) * time.Millisecond
	}
	if outputBufferSize := utils.GetEnvInt("PROCESSING_OUTPUT_BUFFER_SIZE", -1); outputBufferSize != -1 {
		output.ChannelBufferSize = outputBufferSize
	}
}

// overrideChannelConfig overrides channel configuration with environment variables
func overrideChannelConfig(channels *RawChannelConfig) {
	if inputBufferSize := utils.GetEnvInt("PROCESSING_CHANNELS_INPUT_BUFFER_SIZE", -1); inputBufferSize != -1 {
		channels.InputBufferSize = inputBufferSize
	}
	if outputChannelBufferSize := utils.GetEnvInt("PROCESSING_CHANNELS_OUTPUT_BUFFER_SIZE", -1); outputChannelBufferSize != -1 {
		channels.OutputBufferSize = outputChannelBufferSize
	}
}

// overridePipelineLoggerConfig overrides pipeline logger configuration with environment variables
func overridePipelineLoggerConfig(plogger *RawLoggingConfig) {
	if ploggerLevel := utils.GetEnv("PROCESSING_PLOGGER_LEVEL", ""); ploggerLevel != "" {
		plogger.Level = ploggerLevel
	}
	if ploggerFileName := utils.GetEnv("PROCESSING_PLOGGER_FILE_NAME", ""); ploggerFileName != "" {
		plogger.FileName = ploggerFileName
	}
	if ploggerLoggerName := utils.GetEnv("PROCESSING_PLOGGER_LOGGER_NAME", ""); ploggerLoggerName != "" {
		plogger.LoggerName = ploggerLoggerName
	}
	if ploggerServiceName := utils.GetEnv("PROCESSING_PLOGGER_SERVICE_NAME", ""); ploggerServiceName != "" {
		plogger.ServiceName = ploggerServiceName
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
