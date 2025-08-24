package processing

import (
	"context"
	"fmt"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"time"

	"telemetry/utils/alert"

	"google.golang.org/protobuf/proto"
)

type ProcessorConfig struct {
	ProcessingDelay time.Duration
	BatchSize       int
	RuleEngine      RuleEngineConfig
}

type Processor struct {
	config    ProcessorConfig
	logger    logging.Logger
	inputCh   <-chan *models.ChannelMessage
	outputCh  chan<- *models.ChannelMessage
	reHandler *RuleEngineHandler
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewProcessor(config ProcessorConfig, logger logging.Logger, inputCh <-chan *models.ChannelMessage, outputCh chan<- *models.ChannelMessage) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	reHandler := NewRuleHandler(config.RuleEngine, logger.WithField("module", "ruleengine"))
	return &Processor{
		config:    config,
		logger:    logger,
		inputCh:   inputCh,
		outputCh:  outputCh,
		reHandler: reHandler,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *Processor) Start() error {
	p.logger.Infow("Starting processor", "batch_size", p.config.BatchSize, "processing_delay", p.config.ProcessingDelay)
	p.reHandler.Start()
	go p.processLoop()
	return nil
}

func (p *Processor) Stop() error {
	p.logger.Info("Stopping processor")
	p.reHandler.Stop()
	p.cancel()
	return nil
}

func (p *Processor) processLoop() {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Errorw("Processor panic recovered", "panic", r)
		}
	}()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("Processor loop stopped")
			return
		case message := <-p.inputCh:
			if err := p.processMessage(message); err != nil {
				p.logger.Errorw("Failed to process message", "error", err)
			}
		}
	}
}

func (p *Processor) processMessage(message *models.ChannelMessage) error {
	p.logger.Debugw("Processing message", "type", message.Type, "size", len(message.Data))

	// For non-data messages (control messages), forward them as-is
	if !message.IsDataMessage() {
		outputMessage := &models.ChannelMessage{
			Type:      message.Type,
			Data:      message.Data,
			Timestamp: message.Timestamp,
		}

		p.outputCh <- outputMessage
		return nil
	}

	// For data messages, apply processing
	aStream := &alert.AlertStream{}
	err := proto.Unmarshal(message.Data, aStream)
	if err != nil {
		p.logger.Errorf("Failed to unmarshal input record: %w", err)
		return fmt.Errorf("failed to unmarshal input record: %w", err)
	}

	var outAlerts []*alert.Alert
	for _, aObj := range aStream.AlertObject {

		processedRecord, err := p.reHandler.applyRuleToRecord(aObj)
		if err != nil {
			p.logger.Errorw("Failed to apply rule processing", "error", err)
			// keep the record unchanged if processing fails
			outAlerts = append(outAlerts, aObj)
			continue
		}

		outAlerts = append(outAlerts, processedRecord)
	}

	if len(outAlerts) > 0 {
		aStream := &alert.AlertStream{
			AlertObject: outAlerts,
		}
		processedData, err := proto.Marshal(aStream)
		if err != nil {
			return fmt.Errorf("failed to marshal processed data: %w", err)
		}

		// Create a new message with processed data
		outputMessage := models.NewDataMessage(processedData, "processor")
		for k, v := range message.Meta {
			outputMessage.Meta[k] = v
		}

		p.outputCh <- outputMessage
		p.logger.Debug("Processed message sent to output channel")
	}

	return nil
}

func (p *Processor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":           "running",
		"batch_size":       p.config.BatchSize,
		"processing_delay": p.config.ProcessingDelay.String(),
	}
}
