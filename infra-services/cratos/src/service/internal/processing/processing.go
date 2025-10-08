package processing

import (
	"fmt"
	"log"
	"servicegomodule/internal/config"
	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"servicegomodule/internal/rules"
	"sharedgomodule/logging"
	"sharedgomodule/utils"
)

type ProcConfig struct {
	Input        InputConfig
	Processor    ProcessorConfig
	Output       OutputConfig
	Channels     ChannelConfig
	LoggerConfig logging.LoggerConfig
}

type ChannelConfig struct {
	InputBufferSize  int
	OutputBufferSize int
}

type Pipeline struct {
	config        ProcConfig
	logger        logging.Logger // application logger
	plogger       logging.Logger // logger for all processing pipeline components
	inputHandler  *InputHandler
	processor     *Processor
	outputHandler *OutputHandler
	inputCh       <-chan *models.ChannelMessage
	outputCh      chan<- *models.ChannelMessage
}

func NewPipeline(config ProcConfig, logger logging.Logger, metricsCollector *metrics.MetricsCollector) *Pipeline {
	plogger := initPipelineLogger(config.LoggerConfig)

	// Create metrics helpers for each stage
	inputMetricsHelper := metrics.NewMetricsHelper(metricsCollector, "input")
	rulelookupMetricsHelper := metrics.NewMetricsHelper(metricsCollector, "rulelookup")
	processorMetricsHelper := metrics.NewMetricsHelper(metricsCollector, "processor")
	outputMetricsHelper := metrics.NewMetricsHelper(metricsCollector, "output")

	// Create handlers with metrics helpers
	inputHandler := NewInputHandler(config.Input, plogger.WithField("component", "input"), inputMetricsHelper)
	outputHandler := NewOutputHandler(config.Output, plogger.WithField("component", "output"), outputMetricsHelper)
	processor := NewProcessor(config.Processor, plogger.WithField("component", "processor"), inputHandler.GetInputChannel(), outputHandler.GetOutputChannel(), inputHandler.GetInputSink(), processorMetricsHelper, rulelookupMetricsHelper)

	return &Pipeline{
		config:        config,
		logger:        logger,
		plogger:       plogger,
		inputHandler:  inputHandler,
		processor:     processor,
		outputHandler: outputHandler,
		inputCh:       inputHandler.GetInputChannel(),
		outputCh:      outputHandler.GetOutputChannel(),
	}
}

func (p *Pipeline) Start() error {
	p.logger.Info("Starting processing pipeline")

	if err := p.outputHandler.Start(); err != nil {
		return fmt.Errorf("failed to start output handler: %w", err)
	}

	if err := p.processor.Start(); err != nil {
		p.outputHandler.Stop()
		return fmt.Errorf("failed to start processor: %w", err)
	}

	if err := p.inputHandler.Start(); err != nil {
		p.processor.Stop()
		p.outputHandler.Stop()
		return fmt.Errorf("failed to start input handler: %w", err)
	}

	p.logger.Info("Processing pipeline started successfully")
	return nil
}

func (p *Pipeline) Stop() error {
	p.logger.Info("Stopping processing pipeline")

	var errs []error

	if err := p.inputHandler.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("error stopping input handler: %w", err))
	}

	if err := p.processor.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("error stopping processor: %w", err))
	}

	if err := p.outputHandler.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("error stopping output handler: %w", err))
	}

	if len(errs) > 0 {
		p.logger.Errorw("Errors occurred during pipeline shutdown", "error_count", len(errs))
		return fmt.Errorf("pipeline shutdown errors: %v", errs)
	}

	p.logger.Info("Processing pipeline stopped successfully")
	return nil
}

func (p *Pipeline) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"pipeline_status": "running",
		"input_stats":     p.inputHandler.GetStatus(),
		"processor_stats": p.processor.GetStatus(),
		"output_stats":    p.outputHandler.GetStatus(),
	}
}

func (p *Pipeline) GetRuleInfo() []byte {
	return p.processor.GetRuleInfo()
}

func DefaultConfig(cfg *config.RawConfig) ProcConfig {
	processing := cfg.Processing
	// Convert config processing configuration to ProcConfig
	procConfig := ProcConfig{
		Input: InputConfig{
			Topics:            processing.Input.Topics,
			ChannelBufferSize: processing.Input.ChannelBufferSize,
			KafkaConfigMap:    utils.LoadConfigMap(processing.Input.KafkaConfFile),
		},
		Processor: ProcessorConfig{
			ProcessingDelay: processing.Processor.ProcessingDelay,
			BatchSize:       processing.Processor.BatchSize,
			RuleEngine: rules.RuleEngineConfig{
				RulesTopic:                  processing.Processor.RuleProcConfig.RulesTopic,
				RuleEngLibLogging:           processing.Processor.RuleProcConfig.RelibLogging.ConvertToLoggerConfig(),
				RulesKafkaConfigMap:         utils.LoadConfigMap(processing.Processor.RuleProcConfig.RulesKafkaConfFile),
				RuleTasksTopic:              processing.Processor.RuleProcConfig.RuleTasksTopic,
				RuleHandlerLogging:          processing.Processor.RuleProcConfig.RuleHandlerLogging.ConvertToLoggerConfig(),
				RuleTasksConsKafkaConfigMap: utils.LoadConfigMap(processing.Processor.RuleProcConfig.RuleTasksConsKafkaFile),
				RuleTasksProdKafkaConfigMap: utils.LoadConfigMap(processing.Processor.RuleProcConfig.RuleTasksProdKafkaFile),
			},
		},
		Output: OutputConfig{
			OutputTopic:       processing.Output.OutputTopic,
			BatchSize:         processing.Output.BatchSize,
			FlushTimeout:      processing.Output.FlushTimeout,
			ChannelBufferSize: processing.Output.ChannelBufferSize,
			KafkaConfigMap:    utils.LoadConfigMap(processing.Output.KafkaConfFile),
		},
		Channels: ChannelConfig{
			InputBufferSize:  processing.Channels.InputBufferSize,
			OutputBufferSize: processing.Channels.OutputBufferSize,
		},
	}

	// Handle PloggerConfig
	if processing.PloggerConfig.FileName != "" {
		procConfig.LoggerConfig = processing.PloggerConfig.ConvertToLoggerConfig()
	} else {
		panic("PloggerConfig must be provided in the configuration")
	}

	return procConfig
}

func ValidateConfig(config ProcConfig) error {
	if len(config.Input.Topics) == 0 {
		return fmt.Errorf("input topics cannot be empty")
	}
	if config.Input.PollTimeout <= 0 {
		return fmt.Errorf("poll timeout must be positive")
	}
	if config.Input.ChannelBufferSize <= 0 {
		return fmt.Errorf("input channel buffer size must be positive")
	}

	if config.Processor.BatchSize <= 0 {
		return fmt.Errorf("processor batch size must be positive")
	}

	if config.Output.OutputTopic == "" {
		return fmt.Errorf("output topic cannot be empty")
	}
	if config.Output.BatchSize <= 0 {
		return fmt.Errorf("output batch size must be positive")
	}
	if config.Output.FlushTimeout <= 0 {
		return fmt.Errorf("flush timeout must be positive")
	}
	if config.Output.ChannelBufferSize <= 0 {
		return fmt.Errorf("output channel buffer size must be positive")
	}

	if config.Channels.InputBufferSize <= 0 {
		return fmt.Errorf("input buffer size must be positive")
	}
	if config.Channels.OutputBufferSize <= 0 {
		return fmt.Errorf("output buffer size must be positive")
	}

	// Validate logger configuration using the logging package's validation
	if err := config.LoggerConfig.Validate(); err != nil {
		return fmt.Errorf("logger configuration validation failed: %w", err)
	}

	return nil
}

func initPipelineLogger(cfg logging.LoggerConfig) logging.Logger {
	// Use the provided configuration directly
	logger, err := logging.NewLogger(&cfg)
	if err != nil {
		log.Fatalf("Failed to create pipeline logger: %v", err)
	}
	return logger
}
