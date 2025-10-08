package config

import (
	"fmt"
	"os"
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
	Metrics    RawMetricsConfig    `yaml:"metrics"`
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

type RawMetricsConfig struct {
	RetentionPeriod   int `yaml:"retentionPeriod"`   // metrics retention period in minutes
	AggregationWindow int `yaml:"aggregationWindow"` // metrics aggregation window in minutes
	MaxEvents         int `yaml:"maxEvents"`         // maximum number of events to keep in memory
	DumpInterval      int `yaml:"dumpInterval"`      // metrics dump interval in seconds
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

	return nil
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
	if inputTopic := utils.GetEnv("PROCESSING_INPUT_TOPIC", ""); inputTopic != "" {
		input.Topics = []string{inputTopic}
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
	if rulesTopic := utils.GetEnv("PROCESSING_RULES_TOPIC", ""); rulesTopic != "" {
		processor.RuleProcConfig.RulesTopic = rulesTopic
	}
	if ruleTasksTopic := utils.GetEnv("PROCESSING_RULE_TASKS_TOPIC", ""); ruleTasksTopic != "" {
		processor.RuleProcConfig.RuleTasksTopic = ruleTasksTopic
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

// ConvertLoggingConfig converts LoggingConfig to logging.LoggerConfig
func (cfg RawLoggingConfig) ConvertToLoggerConfig() logging.LoggerConfig {
	return logging.LoggerConfig{
		Level:       logging.ConvertToLogLevel(cfg.Level),
		FilePath:    cfg.FileName,
		LoggerName:  cfg.LoggerName,
		ServiceName: cfg.ServiceName,
	}
}
