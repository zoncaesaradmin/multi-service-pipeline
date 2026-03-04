package processing

import (
	"context"
	"fmt"
	"runtime/debug"
	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/utils"
	"time"
)

type ProcessorConfig struct {
	ProcessingDelay time.Duration
	BatchSize       int
	//RuleEngine      rules.RuleEngineConfig
}

type Processor struct {
	config   ProcessorConfig
	logger   logging.Logger
	inputCh  <-chan *models.ChannelMessage
	outputCh chan<- *models.ChannelMessage
	//reHandler     *rules.RuleEngineHandler
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper
}

func NewProcessor(config ProcessorConfig, logger logging.Logger, inputCh <-chan *models.ChannelMessage, outputCh chan<- *models.ChannelMessage, inputSink chan<- *models.ChannelMessage, metricsHelper *metrics.MetricsHelper, rhMetricsHelper *metrics.MetricsHelper) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	//reHandler := rules.NewRuleHandler(config.RuleEngine, logger.WithField("module", "ruleengine"), inputSink, rhMetricsHelper)
	return &Processor{
		config:   config,
		logger:   logger,
		inputCh:  inputCh,
		outputCh: outputCh,
		//reHandler:     reHandler,
		ctx:           ctx,
		cancel:        cancel,
		metricsHelper: metricsHelper,
	}
}

func (p *Processor) Start() error {
	p.logger.Infow("Starting processor", "batch_size", p.config.BatchSize, "processing_delay", p.config.ProcessingDelay)
	//p.reHandler.Start()
	go p.processLoop()
	return nil
}

func (p *Processor) Stop() error {
	p.logger.Info("Stopping processor")
	//p.reHandler.Stop()
	p.cancel()
	return nil
}

func (p *Processor) processLoop() {
	defer func() {
		if r := recover(); r != nil {
			panicStr := fmt.Sprintf("%v", r)
			stack := string(debug.Stack())
			p.logger.Errorw("Processor panic recovered",
				"panic", panicStr,
				"stack", stack,
			)
		}
	}()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("Processor loop stopped")
			return
		case message := <-p.inputCh:
			if err := p.processMessage(message); err != nil {
				p.logger.Errorw("Failed to process message", "error", err, "key", message.Key)
				// Record processing failure
				if p.metricsHelper != nil {
					p.metricsHelper.RecordMessageFailed(message, "processing_error")
					p.metricsHelper.RecordStageFailed(message, "processing_error")
				}
				// Note: We don't commit the offset for failed messages, so they will be reprocessed
			}
		}
	}
}

func (p *Processor) processMessage(message *models.ChannelMessage) error {

	// Use trace-aware logger if context is available
	msgLogger := p.logger
	if message.Context != nil {
		msgLogger = utils.WithTraceLogger(p.logger, message.Context)
	}

	msgLogger.Debugw("Processing message",
		"type", message.Type, "size", len(message.Data))

	return nil
}

func (p *Processor) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":           "running",
		"batch_size":       p.config.BatchSize,
		"processing_delay": p.config.ProcessingDelay.String(),
	}
}
