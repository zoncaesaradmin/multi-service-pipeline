package processing

import (
	"context"
	"fmt"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"time"
)

type OutputConfig struct {
	OutputTopic       string
	BatchSize         int
	FlushTimeout      time.Duration
	ChannelBufferSize int
	KafkaConfigMap    map[string]any
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
	producer := messagebus.NewProducer(config.KafkaConfigMap, "prealertsProducer"+utils.GetEnv("HOSTNAME", ""))

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

	// Process each message individually
	successCount := 0
	for i, message := range batch {
		// Use trace-aware logger if available
		msgLogger := o.logger
		if message.Context != nil {
			msgLogger = utils.WithTraceLogger(o.logger, message.Context)
		}

		if err := o.sendMessage(message); err != nil {
			msgLogger.Errorw("Failed to send message", "error", err, "batch_index", i, "key", message.Key)
			// Don't commit offset for failed messages
		} else {
			// Commit offset immediately after successful send
			if message.HasCommitCallback() {
				if err := message.Commit(context.Background()); err != nil {
					msgLogger.Errorw("Failed to commit message after successful send", "error", err, "key", message.Key)
				}
			}
			successCount++
		}
	}

	o.logger.Debugw("Batch flushed successfully",
		"total_messages", len(batch),
		"successful_messages", successCount,
		"failed_messages", len(batch)-successCount)
}

func (o *OutputHandler) sendMessage(channelMsg *models.ChannelMessage) error {
	// Use trace-aware logger if available
	msgLogger := o.logger
	if channelMsg.Context != nil {
		msgLogger = utils.WithTraceLogger(o.logger, channelMsg.Context)
	}

	// Check if this is an empty message (no alerts generated)
	if len(channelMsg.Data) == 0 && channelMsg.Meta["empty_result"] == "true" {
		msgLogger.Debugw("Skipping empty message send to Kafka", "key", channelMsg.Key)
		return nil // Success - we don't need to send empty messages
	}

	// Ensure trace ID is in the message headers for downstream services
	headers := make(map[string]string)
	for k, v := range channelMsg.Meta {
		headers[k] = v
	}

	// Extract and ensure trace ID is propagated
	if channelMsg.Context != nil {
		if traceID, ok := utils.GetTraceID(channelMsg.Context); ok {
			headers[utils.TraceIDHeader] = traceID
		}
	}

	message := &messagebus.Message{
		Topic:     o.config.OutputTopic,
		Value:     channelMsg.Data,
		Headers:   headers,
		Key:       channelMsg.Key,
		Partition: channelMsg.Partition,
	}

	_, _, err := o.producer.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", o.config.OutputTopic, err)
	}

	msgLogger.Debugw("Message sent successfully", "topic", o.config.OutputTopic, "size", len(channelMsg.Data), "headers", headers)
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
