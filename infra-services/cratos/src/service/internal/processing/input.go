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

// InputConfig holds configuration for the input handler
type InputConfig struct {
	Topics            []string
	PollTimeout       time.Duration
	ChannelBufferSize int
	KafkaConfigMap    map[string]any
}

// InputHandler handles input processing - reads from Kafka and writes to input channel
type InputHandler struct {
	consumer      messagebus.Consumer
	config        InputConfig
	logger        logging.Logger
	inputCh       chan *models.ChannelMessage
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper
}

// NewInputHandler creates a new input handler
func NewInputHandler(config InputConfig, logger logging.Logger, metricsHelper *metrics.MetricsHelper) *InputHandler {
	// Use simple filename - path resolution is handled by messagebus config loader
	consumer := messagebus.NewConsumer(config.KafkaConfigMap, "recordConsGroup")

	return &InputHandler{
		consumer:      consumer,
		config:        config,
		logger:        logger,
		inputCh:       make(chan *models.ChannelMessage, config.ChannelBufferSize),
		metricsHelper: metricsHelper,
	}
}

// GetInputChannel returns the input channel for the processor to read from
func (i *InputHandler) GetInputChannel() <-chan *models.ChannelMessage {
	return i.inputCh
}

// GetInputSink returns a send-only handle to the input channel so other components
// can inject messages into the processing pipeline.
func (i *InputHandler) GetInputSink() chan<- *models.ChannelMessage {
	return i.inputCh
}

// Start starts the input handler
func (i *InputHandler) Start() error {
	i.logger.Infow("Starting input handler", "topics", i.config.Topics)

	// Create context for cancellation
	i.ctx, i.cancel = context.WithCancel(context.Background())

	// Register OnMessage callback
	i.consumer.OnMessage(func(message *messagebus.Message) {
		if message != nil {
			// Extract or generate trace ID from message headers
			traceID := utils.ExtractTraceID(message.Headers)
			traceCtx := utils.WithTraceID(context.Background(), traceID)

			// Use trace-aware logger for this message
			msgLogger := utils.WithTraceLogger(i.logger, traceCtx)
			msgLogger.Debugw("Received kafka data message", "size", len(message.Value))

			// Create commit callback that will be called after successful processing
			commitCallback := func(ctx context.Context) error {
				if err := i.consumer.Commit(ctx, message); err != nil {
					msgLogger.Warnw("Failed to commit message", "error", err, "key", message.Key)
					return err
				}
				msgLogger.Debugw("Message committed successfully", "key", message.Key, "offset", message.Offset)
				return nil
			}

			channelMsg := models.NewDataMessage(message.Value, message.Key, message.Partition)
			channelMsg.CommitCallback = commitCallback
			channelMsg.Origin = models.ChannelMessageOriginKafka
			channelMsg.Context = traceCtx // Attach trace context to message

			// Set timing information
			channelMsg.EntryTimestamp = time.Now()
			channelMsg.ProcessingStage = "input"
			channelMsg.Size = int64(len(message.Value))

			// Ensure trace ID is in message headers for downstream processing
			if channelMsg.Meta == nil {
				channelMsg.Meta = make(map[string]string)
			}
			for k, v := range message.Headers {
				channelMsg.Meta[k] = v
			}
			channelMsg.Meta[utils.TraceIDHeader] = traceID // Ensure trace ID is propagated

			// Record metrics if available
			if i.metricsHelper != nil {
				i.metricsHelper.RecordMessageReceived(channelMsg)
			}

			i.inputCh <- channelMsg
			msgLogger.Debugw("Input message received", "key", message.Key, "headers", message.Headers, "size", len(message.Value))

			// Record message processed at input stage with timing
			if i.metricsHelper != nil {
				processingDuration := time.Since(channelMsg.EntryTimestamp)
				i.metricsHelper.RecordStageCompleted(channelMsg, processingDuration)
			}
		}
	})

	// Subscribe to topics
	if err := i.consumer.Subscribe(i.config.Topics); err != nil {
		i.logger.Errorf("failed to subscribe to topics: %w", err)
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	return nil
}

// Stop stops the input handler
func (i *InputHandler) Stop() error {
	i.logger.Info("Stopping input handler")

	if i.cancel != nil {
		i.cancel()
	}

	if i.consumer != nil {
		if err := i.consumer.Close(); err != nil {
			i.logger.Errorw("Error closing consumer", "error", err)
			return err
		}
	}

	return nil
}

// GetStatus returns statistics about the input handler
func (i *InputHandler) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":              "running",
		"topics":              i.config.Topics,
		"poll_timeout":        i.config.PollTimeout.String(),
		"channel_buffer_size": i.config.ChannelBufferSize,
	}
}
