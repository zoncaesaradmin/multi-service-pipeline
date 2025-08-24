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
	L               logging.Logger
	ConsHandler     *ConsumerHandler
	ProducerHandler *ProducerHandler
	SentDataSize    int
	SentDataCount   int
}

type ConsumerHandler struct {
	consumer      messagebus.Consumer
	logger        logging.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	receivedAll   bool
	receivedCount int
	expectedCount int
	expectedMap   map[string]string
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

func (i *ConsumerHandler) GetReceivedAll() bool {
	return i.receivedAll
}

func (i *ConsumerHandler) SetReceivedAll(receivedAll bool) {
	i.receivedAll = receivedAll
}

func (i *ConsumerHandler) SetExpectedCount(count int) {
	i.expectedCount = count
}

func (i *ConsumerHandler) SetExpectedMap(expectedMap map[string]string) {
	i.expectedMap = expectedMap
}

func (i *ConsumerHandler) Reset() {
	i.receivedCount = 0
	i.expectedCount = 0
	i.receivedAll = false
	i.expectedMap = make(map[string]string)
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
			message, err := i.consumer.Poll(1000)
			if err != nil {
				i.logger.Warnw("Error polling for messages", "error", err)
				continue
			}

			if message != nil {

				// ensure data is valid test data only
				// ensure data received matches expectations
				if EnsureMapMatches(i.expectedMap, message.Headers) {
					i.logger.Debugw("Received valid test data message", "size", len(message.Value))
					i.receivedCount++
					if i.receivedCount == i.expectedCount {
						// simple indication for now that data is received
						// to be enhanced to include checks based on sent test data
						i.receivedAll = true
					}
				} else {
					i.logger.Debugw("Received some other data message", "size", len(message.Value))
				}

				// Commit the message
				if err := i.consumer.Commit(context.Background(), message); err != nil {
					i.logger.Warnw("Failed to commit message", "error", err)
				}
			}
		}
	}
}

type ProducerHandler struct {
	producer messagebus.Producer
	logger   logging.Logger
}

func NewProducerHandler(logger logging.Logger) *ProducerHandler {
	confFilename := utils.ResolveConfFilePath("kafka-producer.yaml")
	kafkaConf := utils.LoadConfigMap(confFilename)

	producer := messagebus.NewProducer(kafkaConf)

	return &ProducerHandler{
		producer: producer,
		logger:   logger,
	}
}
func (o *ProducerHandler) Start() error {
	o.logger.Debug("Starting producer handler")
	return nil
}

func (o *ProducerHandler) Stop() error {
	o.logger.Debug("Stopping producer handler")

	if o.producer != nil {
		if err := o.producer.Close(); err != nil {
			o.logger.Errorw("Error closing producer", "error", err)
			return err
		}
	}
	return nil
}

func (o *ProducerHandler) Send(topic string, data []byte, headers map[string]string) error {
	message := &messagebus.Message{
		Topic:   topic,
		Value:   data,
		Headers: headers,
	}
	_, _, err := o.producer.Send(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", topic, err)
	}

	o.logger.Debugw("Message sent successfully", "topic", topic, "size", len(data))

	return nil
}

func EnsureMapMatches(expected, data map[string]string) bool {
	for k, v := range expected {
		dv, ok := data[k]
		if !ok || dv != v {
			return false
		}
	}
	return true
}
