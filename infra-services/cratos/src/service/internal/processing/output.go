package processing

import (
	"servicegomodule/internal/models"
	"context"
	"fmt"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"time"
)

type OutputConfig struct {
	OutputTopic       string        `json:"outputTopic"`
	BatchSize         int           `json:"batchSize"`
	FlushTimeout      time.Duration `json:"flushTimeout"`
	ChannelBufferSize int           `json:"channelBufferSize"`
}

type OutputHandler struct {
	config   OutputConfig
	producer messagebus.Producer
	logger   logging.Logger
	outputCh chan *models.ChannelMessage
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewOutputHandler(config OutputConfig, logger logging.Logger) *OutputHandler {
	ctx, cancel := context.WithCancel(context.Background())
	producer := messagebus.NewProducer("kafka-producer.yaml")

	return &OutputHandler{
		config:   config,
		producer: producer,
		logger:   logger,
		outputCh: make(chan *models.ChannelMessage, config.ChannelBufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// GetOutputChannel returns the output channel for the processor to write to
func (o *OutputHandler) GetOutputChannel() chan<- *models.ChannelMessage {
	return o.outputCh
}

func (o *OutputHandler) Start() error {
	o.logger.Infow("Starting output handler", "topic", o.config.OutputTopic, "batch_size", o.config.BatchSize)

	go o.produceLoop()
	return nil
}

func (o *OutputHandler) Stop() error {
	o.logger.Info("Stopping output handler")
	o.cancel()

	if o.producer != nil {
		if err := o.producer.Close(); err != nil {
			o.logger.Errorw("Error closing producer", "error", err)
			return err
		}
	}

	return nil
}

func (o *OutputHandler) produceLoop() {
	defer func() {
		if r := recover(); r != nil {
			o.logger.Errorw("Output handler panic recovered", "panic", r)
		}
	}()

	batch := make([]*models.ChannelMessage, 0, o.config.BatchSize)
	flushTicker := time.NewTicker(o.config.FlushTimeout)
	defer flushTicker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			if len(batch) > 0 {
				o.flushBatch(batch)
			}
			o.logger.Info("Output handler produce loop stopped")
			return

		case <-flushTicker.C:
			if len(batch) > 0 {
				o.flushBatch(batch)
				batch = batch[:0]
			}

		case message := <-o.outputCh:
			batch = append(batch, message)
			o.logger.Debugw("Added message to batch", "batch_size", len(batch), "type", message.Type)

			if len(batch) >= o.config.BatchSize {
				o.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (o *OutputHandler) flushBatch(batch []*models.ChannelMessage) {
	if len(batch) == 0 {
		return
	}

	o.logger.Debugw("Flushing batch to Kafka", "batch_size", len(batch), "topic", o.config.OutputTopic)

	for i, message := range batch {
		if err := o.sendMessage(message); err != nil {
			o.logger.Errorw("Failed to send message", "error", err, "batch_index", i)
		}
	}

	o.logger.Debugw("Batch flushed successfully", "messages_sent", len(batch))
}

func (o *OutputHandler) sendMessage(channelMsg *models.ChannelMessage) error {

	message := &messagebus.Message{
		Topic: o.config.OutputTopic,
		Value: channelMsg.Data,
	}

	_, _, err := o.producer.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", o.config.OutputTopic, err)
	}

	o.logger.Debugw("Message sent successfully", "topic", o.config.OutputTopic, "size", len(channelMsg.Data))
	return nil
}

func (o *OutputHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":        "running",
		"output_topic":  o.config.OutputTopic,
		"batch_size":    o.config.BatchSize,
		"flush_timeout": o.config.FlushTimeout.String(),
	}
}
