package processing

import (
	"context"
	"fmt"
	"servicegomodule/internal/metrics"
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
	config        OutputConfig
	producer      messagebus.Producer
	logger        logging.Logger
	outputCh      chan *models.ChannelMessage
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper
}

func NewOutputHandler(config OutputConfig, logger logging.Logger, metricsHelper *metrics.MetricsHelper) *OutputHandler {
	ctx, cancel := context.WithCancel(context.Background())
	producer := messagebus.NewProducer(config.KafkaConfigMap, "prealertsProducer"+utils.GetEnv("HOSTNAME", ""))

	return &OutputHandler{
		config:        config,
		producer:      producer,
		logger:        logger,
		outputCh:      make(chan *models.ChannelMessage, config.ChannelBufferSize),
		ctx:           ctx,
		cancel:        cancel,
		metricsHelper: metricsHelper,
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
			//o.logger.Debugw("Added message to batch", "batch_size", len(batch), "type", message.Type)

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

	batchStartTime := time.Now()
	//o.logger.Debugw("Flushing batch to Kafka", "batch_size", len(batch), "topic", o.config.OutputTopic)

	// Process each message individually
	successCount := 0
	for i, message := range batch {
		sendStartTime := time.Now()

		// Use trace-aware logger if available
		msgLogger := o.logger
		if message.Context != nil {
			msgLogger = utils.WithTraceLogger(o.logger, message.Context)
		}

		if err := o.sendMessage(message); err != nil {
			msgLogger.Errorw("Failed to send message", "error", err, "batch_index", i, "key", message.Key)
			// Record send failure
			if o.metricsHelper != nil {
				o.metricsHelper.RecordError(message, "send_failed")
				o.metricsHelper.RecordStageLatency(time.Since(sendStartTime), "send_failed")
				o.metricsHelper.RecordStageFailed(message, "send_failed")
			}
			// Set output timestamp
			message.OutputTimestamp = time.Now()

			// Don't commit offset for failed messages
		} else {
			// Record successful send
			if o.metricsHelper != nil {
				o.metricsHelper.RecordStageLatency(time.Since(sendStartTime), "send_success")
			}

			// Set output timestamp after message is sent out
			message.OutputTimestamp = time.Now()

			// Commit offset immediately after successful send
			if message.HasCommitCallback() {
				commitStartTime := time.Now()
				if err := message.Commit(context.Background()); err != nil {
					msgLogger.Errorw("Failed to commit message after successful send", "error", err, "key", message.Key)
					if o.metricsHelper != nil {
						o.metricsHelper.RecordError(message, "commit_failed")
						o.metricsHelper.RecordStageLatency(time.Since(commitStartTime), "commit_failed")
					}
				} else {
					// Record successful commit and end-to-end timing
					if o.metricsHelper != nil {
						o.metricsHelper.RecordStageLatency(time.Since(commitStartTime), "commit_success")

						// Calculate and record end-to-end latency
						if !message.EntryTimestamp.IsZero() {
							endToEndLatency := time.Since(message.EntryTimestamp)
							o.metricsHelper.RecordStageLatency(endToEndLatency, "end_to_end")
						}

						// Record message as fully processed at output stage
						outputDuration := time.Since(sendStartTime)
						o.metricsHelper.RecordStageProcessed(message, outputDuration)
						o.metricsHelper.RecordMessageProcessed(message)
					}
				}
			}
			successCount++
		}

	}

	// Record batch processing metrics
	if o.metricsHelper != nil {
		batchDuration := time.Since(batchStartTime)
		o.metricsHelper.RecordStageLatency(batchDuration, "batch_flush")
		o.metricsHelper.RecordCounter("batch.processed", 1, map[string]string{
			"total_messages":      fmt.Sprintf("%d", len(batch)),
			"successful_messages": fmt.Sprintf("%d", successCount),
			"failed_messages":     fmt.Sprintf("%d", len(batch)-successCount),
		})
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
		if o.metricsHelper != nil {
			o.metricsHelper.RecordCounter("message.empty_skipped", 1, nil)
		}
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
		if o.metricsHelper != nil {
			o.metricsHelper.RecordError(channelMsg, "kafka_send_failed")
		}
		return fmt.Errorf("failed to send message to topic %s: %w", o.config.OutputTopic, err)
	}

	// Record successful send
	if o.metricsHelper != nil {
		o.metricsHelper.RecordCounter("message.sent", 1, map[string]string{
			"topic": o.config.OutputTopic,
			"size":  fmt.Sprintf("%d", len(channelMsg.Data)),
		})
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
