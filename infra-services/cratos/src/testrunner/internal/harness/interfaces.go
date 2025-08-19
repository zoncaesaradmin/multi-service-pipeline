package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"testgomodule/internal/config"
	"sharedgomodule/messagebus"
)

type TestHarness interface {
	Initialize() error
	SendMessage(data map[string]interface{}) error
	ReceiveMessage(timeout time.Duration) (map[string]interface{}, error)
	Cleanup() error
}

type LocalHarness struct {
	producer messagebus.Producer
	consumer messagebus.Consumer
	config   config.MessageBusConfig
}

func NewTestHarness(cfg config.MessageBusConfig) (TestHarness, error) {
	return NewLocalHarness(cfg), nil
}

func NewLocalHarness(cfg config.MessageBusConfig) *LocalHarness {
	producer := messagebus.NewProducer("kafka-producer.yaml")
	consumer := messagebus.NewConsumer("kafka-consumer.yaml", "")

	return &LocalHarness{
		producer: producer,
		consumer: consumer,
		config:   cfg,
	}
}

func (h *LocalHarness) Initialize() error {
	// Subscribe to test output topic to receive responses
	return h.consumer.Subscribe([]string{"test_output"})
}

func (h *LocalHarness) SendMessage(data map[string]interface{}) error {
	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message data: %w", err)
	}

	// Create message for test input topic
	message := &messagebus.Message{
		Topic: "test_input",
		Key:   "test",
		Value: jsonData,
	}

	// Send message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err = h.producer.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (h *LocalHarness) ReceiveMessage(timeout time.Duration) (map[string]interface{}, error) {
	// Poll for response message
	message, err := h.consumer.Poll(timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to poll message: %w", err)
	}

	if message == nil {
		return nil, fmt.Errorf("timeout receiving message")
	}

	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(message.Value, &responseData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return responseData, nil
}

func (h *LocalHarness) Cleanup() error {
	if err := h.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	if err := h.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}

	return nil
}
