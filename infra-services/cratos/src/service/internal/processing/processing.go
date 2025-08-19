package processing

import (
	"fmt"
	"log"
	"servicegomodule/internal/config"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"time"
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

func NewPipeline(config ProcConfig, logger logging.Logger) *Pipeline {
	plogger := initPipelineLogger(config.LoggerConfig)
	inputHandler := NewInputHandler(config.Input, plogger.WithField("component", "input"))
	outputHandler := NewOutputHandler(config.Output, plogger.WithField("component", "output"))
	processor := NewProcessor(config.Processor, plogger.WithField("component", "processor"), inputHandler.GetInputChannel(), outputHandler.GetOutputChannel())

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

func (p *Pipeline) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"pipeline_status": "running",
		"input_stats":     p.inputHandler.GetStats(),
		"processor_stats": p.processor.GetStats(),
		"output_stats":    p.outputHandler.GetStats(),
	}
}

func DefaultConfig(cfg *config.RawConfig) ProcConfig {
	if cfg == nil {
		// Return hardcoded defaults if no config is provided
		return ProcConfig{
			Input: InputConfig{
				Topics:            []string{"input-topic"},
				PollTimeout:       1 * time.Second,
				ChannelBufferSize: 1000,
			},
			Processor: ProcessorConfig{
				ProcessingDelay: 10 * time.Millisecond,
				BatchSize:       100,
			},
			Output: OutputConfig{
				OutputTopic:       "output-topic",
				BatchSize:         50,
				FlushTimeout:      5 * time.Second,
				ChannelBufferSize: 1000,
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

	// Check if Processing configuration exists (check for empty input topics)
	processing := cfg.Processing
	if len(processing.Input.Topics) == 0 {
		// If Processing is empty/nil, use defaults but fill LoggerConfig from main config
		procConfig := ProcConfig{
			Input: InputConfig{
				Topics:            []string{"input-topic"},
				PollTimeout:       1 * time.Second,
				ChannelBufferSize: 1000,
			},
			Processor: ProcessorConfig{
				ProcessingDelay: 10 * time.Millisecond,
				BatchSize:       100,
			},
			Output: OutputConfig{
				OutputTopic:       "output-topic",
				BatchSize:         50,
				FlushTimeout:      5 * time.Second,
				ChannelBufferSize: 1000,
			},
			Channels: ChannelConfig{
				InputBufferSize:  1000,
				OutputBufferSize: 1000,
			},
		}

		// Use PloggerConfig if available, otherwise use defaults
		if processing.PloggerConfig.FileName != "" {
			procConfig.LoggerConfig = processing.PloggerConfig.ConvertToLoggerConfig()
		} else {
			procConfig.LoggerConfig = logging.LoggerConfig{
				Level:         logging.InfoLevel,
				FilePath:      "/tmp/cratos-pipeline.log",
				LoggerName:    "pipeline",
				ComponentName: "processing",
				ServiceName:   "cratos",
			}
		}
		return procConfig
	}

	// Convert config processing configuration to ProcConfig
	procConfig := ProcConfig{
		Input: InputConfig{
			Topics:            processing.Input.Topics,
			PollTimeout:       processing.Input.PollTimeout,
			ChannelBufferSize: processing.Input.ChannelBufferSize,
		},
		Processor: ProcessorConfig{
			ProcessingDelay: processing.Processor.ProcessingDelay,
			BatchSize:       processing.Processor.BatchSize,
		},
		Output: OutputConfig{
			OutputTopic:       processing.Output.OutputTopic,
			BatchSize:         processing.Output.BatchSize,
			FlushTimeout:      processing.Output.FlushTimeout,
			ChannelBufferSize: processing.Output.ChannelBufferSize,
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
		// Use default pipeline logger configuration
		procConfig.LoggerConfig = logging.LoggerConfig{
			Level:         logging.InfoLevel,
			FilePath:      "/tmp/cratos-pipeline.log",
			LoggerName:    "pipeline",
			ComponentName: "processing",
			ServiceName:   "cratos",
		}
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
