package types

import (
	"context"
	"fmt"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
)

// this is a custom suite context structure that can be passed to the steps
type CustomContext struct {
	L           logging.Logger
	Producer    messagebus.Producer
	ConsHandler *ConsumerHandler
}

type ConsumerHandler struct {
	consumer messagebus.Consumer
	logger   logging.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewInputHandler creates a new input handler
func NewConsumerHandler(logger logging.Logger) *ConsumerHandler {

	confFilename := utils.ResolveConfFilePath("kafka-consumer.yaml")
	kafkaConf := utils.LoadConfigMap(confFilename)
	consumer := messagebus.NewConsumer(kafkaConf, "prealertConsGroup")

	return &ConsumerHandler{
		consumer: consumer,
		logger:   logger,
	}
}

// Start starts the input handler
func (i *ConsumerHandler) Start() error {
	topics := []string{"cisco_nir-prealerts"}
	i.logger.Infow("Starting consumer handler", "topics", topics)

	// Create context for cancellation
	i.ctx, i.cancel = context.WithCancel(context.Background())

	// Subscribe to topics
	if err := i.consumer.Subscribe(topics); err != nil {
		i.logger.Errorf("failed to subscribe to topics: %w", err)
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	// Start consuming in a goroutine
	go i.consumeLoop()

	return nil
}

// Stop stops the input handler
func (i *ConsumerHandler) Stop() error {
	i.logger.Info("Stopping consumer handler")

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

func (i *ConsumerHandler) consumeLoop() {
	defer func() {
		if r := recover(); r != nil {
			i.logger.Errorw("Consumer handler panic recovered", "panic", r)
		}
	}()

	for {
		select {
		case <-i.ctx.Done():
			i.logger.Info("Consumer handler consume loop stopped")
			return
		default:
			// Poll for messages
			message, err := i.consumer.Poll(100)
			if err != nil {
				i.logger.Warnw("Error polling for messages", "error", err)
				continue
			}

			if message != nil {
				i.logger.Debugw("Received data message", "size", len(message.Value))

				// Commit the message
				if err := i.consumer.Commit(context.Background(), message); err != nil {
					i.logger.Warnw("Failed to commit message", "error", err)
				}
			}
		}
	}
}
