package processing

import (
	"context"
	"fmt"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
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
	consumer messagebus.Consumer
	config   InputConfig
	logger   logging.Logger
	inputCh  chan *models.ChannelMessage
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewInputHandler creates a new input handler
func NewInputHandler(config InputConfig, logger logging.Logger) *InputHandler {
	// Use simple filename - path resolution is handled by messagebus config loader
	consumer := messagebus.NewConsumer(config.KafkaConfigMap, "recordConsGroup")

	return &InputHandler{
		consumer: consumer,
		config:   config,
		logger:   logger,
		inputCh:  make(chan *models.ChannelMessage, config.ChannelBufferSize),
	}
}

// GetInputChannel returns the input channel for the processor to read from
func (i *InputHandler) GetInputChannel() <-chan *models.ChannelMessage {
	return i.inputCh
}

// Start starts the input handler
func (i *InputHandler) Start() error {
	i.logger.Infow("Starting input handler", "topics", i.config.Topics)

	// Create context for cancellation
	i.ctx, i.cancel = context.WithCancel(context.Background())

	// Subscribe to topics
	if err := i.consumer.Subscribe(i.config.Topics); err != nil {
		i.logger.Errorf("failed to subscribe to topics: %w", err)
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	// Start consuming in a goroutine
	go i.consumeLoop()

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

// consumeLoop continuously polls for messages and forwards to input channel
func (i *InputHandler) consumeLoop() {
	defer func() {
		if r := recover(); r != nil {
			i.logger.Errorw("Input handler panic recovered", "panic", r)
		}
	}()

	for {
		select {
		case <-i.ctx.Done():
			i.logger.Info("Input handler consume loop stopped")
			return
		default:
			// Poll for messages
			message, err := i.consumer.Poll(i.config.PollTimeout)
			if err != nil {
				i.logger.Warnw("Error polling for messages", "error", err)
				continue
			}

			if message != nil {
				i.logger.Debugw("Received kafka data message", "size", len(message.Value))

				// Create a ChannelMessage from the Kafka message
				channelMsg := models.NewDataMessage(message.Value, message.Key)
				for k, v := range message.Headers {
					channelMsg.Meta[k] = v
				}

				i.inputCh <- channelMsg
				i.logger.Debugw("Input message received", "key", message.Key, "headers", message.Headers)

				// Commit the message
				if err := i.consumer.Commit(context.Background(), message); err != nil {
					i.logger.Warnw("Failed to commit message", "error", err)
				}
			}
		}
	}
}

// GetStats returns statistics about the input handler
func (i *InputHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":              "running",
		"topics":              i.config.Topics,
		"poll_timeout":        i.config.PollTimeout.String(),
		"channel_buffer_size": i.config.ChannelBufferSize,
	}
}
