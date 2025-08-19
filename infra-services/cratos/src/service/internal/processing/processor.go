package processing

import (
	"servicegomodule/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"sharedgomodule/logging"
	"time"
)

type ProcessorConfig struct {
	ProcessingDelay time.Duration
	BatchSize       int
}

type ProcessingRecord struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]string      `json:"metadata"`
}

type Processor struct {
	config   ProcessorConfig
	logger   logging.Logger
	inputCh  <-chan *models.ChannelMessage
	outputCh chan<- *models.ChannelMessage
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewProcessor(config ProcessorConfig, logger logging.Logger, inputCh <-chan *models.ChannelMessage, outputCh chan<- *models.ChannelMessage) *Processor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Processor{
		config:   config,
		logger:   logger,
		inputCh:  inputCh,
		outputCh: outputCh,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (p *Processor) Start() error {
	p.logger.Infow("Starting processor", "batch_size", p.config.BatchSize, "processing_delay", p.config.ProcessingDelay)
	go p.processLoop()
	return nil
}

func (p *Processor) Stop() error {
	p.logger.Info("Stopping processor")
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
				p.logger.Errorw("Error processing message", "error", err)
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
	var record ProcessingRecord
	if err := json.Unmarshal(message.Data, &record); err != nil {
		return fmt.Errorf("failed to unmarshal input record: %w", err)
	}

	processedRecord, err := p.applyProcessing(record)
	if err != nil {
		return fmt.Errorf("failed to apply processing: %w", err)
	}

	processedData, err := json.Marshal(processedRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal processed record: %w", err)
	}

	// Create a new message with processed data
	outputMessage := models.NewDataMessage(processedData, "processor")

	p.outputCh <- outputMessage
	p.logger.Debug("Processed message sent to output channel")

	return nil
}

func (p *Processor) applyProcessing(input ProcessingRecord) (ProcessingRecord, error) {
	p.logger.Debugw("Applying processing transformations", "record_id", input.ID)

	if p.config.ProcessingDelay > 0 {
		time.Sleep(p.config.ProcessingDelay)
	}

	processed := ProcessingRecord{
		ID:        input.ID,
		Timestamp: time.Now().UTC(),
		Data:      make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}

	for key, value := range input.Data {
		if strValue, ok := value.(string); ok {
			processed.Data[key] = fmt.Sprintf("processed_%s", strValue)
		} else {
			processed.Data[key] = value
		}
	}

	for key, value := range input.Metadata {
		processed.Metadata[key] = value
	}
	processed.Metadata["processed_at"] = processed.Timestamp.Format(time.RFC3339)
	processed.Metadata["processing_version"] = "1.0"

	if processed.Data == nil {
		processed.Data = make(map[string]interface{})
	}
	processed.Data["processing_stats"] = map[string]interface{}{
		"processed_fields":    len(input.Data),
		"processing_delay_ms": p.config.ProcessingDelay.Milliseconds(),
	}

	return processed, nil
}

func (p *Processor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":           "running",
		"batch_size":       p.config.BatchSize,
		"processing_delay": p.config.ProcessingDelay.String(),
	}
}
