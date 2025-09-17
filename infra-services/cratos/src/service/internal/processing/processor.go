package processing

import (
	"context"
	"fmt"
	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/utils"
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
	config        ProcessorConfig
	logger        logging.Logger
	inputCh       <-chan *models.ChannelMessage
	outputCh      chan<- *models.ChannelMessage
	reHandler     *RuleEngineHandler
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper
}

func NewProcessor(config ProcessorConfig, logger logging.Logger, inputCh <-chan *models.ChannelMessage, outputCh chan<- *models.ChannelMessage, metricsHelper *metrics.MetricsHelper) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	reHandler := NewRuleHandler(config.RuleEngine, logger.WithField("module", "ruleengine"))
	return &Processor{
		config:        config,
		logger:        logger,
		inputCh:       inputCh,
		outputCh:      outputCh,
		reHandler:     reHandler,
		ctx:           ctx,
		cancel:        cancel,
		metricsHelper: metricsHelper,
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
	// Start timing for this message processing
	startTime := time.Now()

	// Use trace-aware logger if context is available
	msgLogger := p.logger
	if message.Context != nil {
		msgLogger = utils.WithTraceLogger(p.logger, message.Context)
	}

	msgLogger.Debugw("Processing message", "type", message.Type, "size", len(message.Data))

	// Update processing stage
	message.ProcessStartTime = startTime
	message.ProcessingStage = "processor"

	// For non-data messages (control messages), forward them as-is
	if !message.IsDataMessage() {
		outputMessage := &models.ChannelMessage{
			Type:           message.Type,
			Data:           message.Data,
			Timestamp:      message.Timestamp,
			Partition:      message.Partition,
			CommitCallback: message.CommitCallback, // Propagate commit callback
			Context:        message.Context,        // Propagate trace context
		}

		// Record non-data message processing
		if p.metricsHelper != nil {
			processingDuration := time.Since(startTime)
			p.metricsHelper.RecordStageProcessed(outputMessage, processingDuration)
		}

		p.outputCh <- outputMessage
		return nil
	}

	// For data messages, apply processing
	aStream := &alert.AlertStream{}
	err := proto.Unmarshal(message.Data, aStream)
	if err != nil {
		msgLogger.Errorf("Failed to unmarshal input record: %w", err)
		if p.metricsHelper != nil {
			p.metricsHelper.RecordError(message, "unmarshal_failed")
		}
		return fmt.Errorf("failed to unmarshal input record: %w", err)
	}

	var outAlerts []*alert.Alert
	ruleApplyErrors := 0
	recordCountInMsg := 0

	for _, aObj := range aStream.AlertObject {
		recordCountInMsg++
		processedRecord, err := p.reHandler.applyRuleToRecord(msgLogger, aObj, message.Origin)
		if err != nil {
			msgLogger.Errorw("Failed to apply rule processing", "error", err)
			ruleApplyErrors++
			if p.metricsHelper != nil {
				p.metricsHelper.RecordError(message, "rule_apply_failed")
			}
			// keep the record unchanged if processing fails
			outAlerts = append(outAlerts, aObj)
			continue
		}

		outAlerts = append(outAlerts, processedRecord)
	}

	// Record rule applying metrics
	if p.metricsHelper != nil {
		if ruleApplyErrors > 0 {
			p.metricsHelper.RecordCounter("rule.apply.erroredrecords", float64(ruleApplyErrors), map[string]string{"type": "applying_failed"})
		}
		p.metricsHelper.RecordCounter("rule.msg.records", float64(recordCountInMsg), nil)
		p.metricsHelper.RecordCounter("rule.appliedrecords", float64(recordCountInMsg-ruleApplyErrors), nil)
	}

	if len(outAlerts) > 0 {
		aStream := &alert.AlertStream{
			AlertObject: outAlerts,
		}
		processedData, err := proto.Marshal(aStream)
		if err != nil {
			if p.metricsHelper != nil {
				p.metricsHelper.RecordError(message, "marshal_failed")
			}
			return fmt.Errorf("failed to marshal processed data: %w", err)
		}

		// Create a new message with processed data
		outputMessage := models.NewDataMessage(processedData, message.Key, message.Partition)
		outputMessage.CommitCallback = message.CommitCallback // Propagate commit callback
		outputMessage.Context = message.Context               // Propagate trace context
		outputMessage.EntryTimestamp = message.EntryTimestamp
		outputMessage.ProcessStartTime = message.ProcessStartTime
		outputMessage.ProcessEndTime = time.Now()
		outputMessage.ProcessingStage = "processor_complete"
		outputMessage.Size = int64(len(processedData))

		for k, v := range message.Meta {
			outputMessage.Meta[k] = v
		}

		// Record successful processing
		if p.metricsHelper != nil {
			processingDuration := time.Since(startTime)
			p.metricsHelper.RecordStageLatency(processingDuration, "process_message")
			p.metricsHelper.RecordStageProcessed(outputMessage, processingDuration)
		}

		p.outputCh <- outputMessage
		msgLogger.Debug("Processed message sent to output channel")
	} else {
		// Even if no alerts are generated, we should commit the offset
		// Create an empty output message just to trigger the commit
		outputMessage := models.NewDataMessage([]byte{}, message.Key, message.Partition)
		outputMessage.CommitCallback = message.CommitCallback
		outputMessage.Context = message.Context // Propagate trace context
		outputMessage.EntryTimestamp = message.EntryTimestamp
		outputMessage.ProcessStartTime = message.ProcessStartTime
		outputMessage.ProcessEndTime = time.Now()
		outputMessage.ProcessingStage = "processor_empty"
		outputMessage.Meta = map[string]string{"empty_result": "true"}
		for k, v := range message.Meta {
			outputMessage.Meta[k] = v
		}

		// Record empty result processing
		if p.metricsHelper != nil {
			processingDuration := time.Since(startTime)
			p.metricsHelper.RecordStageLatency(processingDuration, "process_empty")
			p.metricsHelper.RecordStageProcessed(outputMessage, processingDuration)
			p.metricsHelper.RecordCounter("message.empty_result", 1, nil)
		}

		p.outputCh <- outputMessage
		p.logger.Debugw("No alerts generated, sending empty message to trigger commit", "key", message.Key)
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
