//go:build local
// +build local

package messagebus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testTopic = "test-topic"

// cleanupMessageBusDir removes all message files to ensure tests start clean
func cleanupMessageBusDir() {
	os.RemoveAll("/tmp/cratos-messagebus-test")
}

// Test LocalProducer.NewProducer
func TestNewLocalProducer(t *testing.T) {
	producer := NewProducer("test_producer_config.yaml")
	assert.NotNil(t, producer)
	assert.IsType(t, &LocalProducer{}, producer)
}

// Test LocalProducer.Send
func TestLocalProducerSend(t *testing.T) {
	// Clean up any existing messages from previous tests
	cleanupMessageBusDir()

	producer := NewProducer("test_producer_config.yaml")
	assert.NotNil(t, producer)

	ctx := context.Background()
	message := &Message{
		Topic:   "test-topic",
		Key:     "test-key",
		Value:   []byte("test-value"),
		Headers: map[string]string{"header1": "value1"},
	}

	partition, offset, err := producer.Send(ctx, message)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), partition)
	assert.Equal(t, int64(0), offset)
}

// Test LocalConsumer.NewConsumer
func TestNewLocalConsumer(t *testing.T) {
	consumer := NewConsumer("test_consumer_config.yaml", "")
	assert.NotNil(t, consumer)
	assert.IsType(t, &LocalConsumer{}, consumer)
}

// Test LocalConsumer.Subscribe
func TestLocalConsumer_Subscribe(t *testing.T) {
	consumer := NewConsumer("test_consumer_config.yaml", "")
	err := consumer.Subscribe([]string{"test-topic"})
	assert.NoError(t, err)
}

// Test LocalConsumer.Poll
func TestLocalConsumer_Poll(t *testing.T) {
	consumer := NewConsumer("test_consumer_config.yaml", "")
	msg, err := consumer.Poll(time.Second)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

// Test integration between producer and consumer
func TestLocalProducerConsumerIntegration(t *testing.T) {
	// Clean up any existing messages from previous tests
	cleanupMessageBusDir()

	producer := NewProducer("test_producer_config.yaml")
	consumer := NewConsumer("test_consumer_config.yaml", "")

	ctx := context.Background()

	// Subscribe consumer
	err := consumer.Subscribe([]string{"integration-topic"})
	assert.NoError(t, err)

	// Send message via producer
	message := &Message{
		Topic: "integration-topic",
		Key:   "key1",
		Value: []byte("value1"),
	}

	partition, offset, err := producer.Send(ctx, message)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), partition)
	assert.Equal(t, int64(0), offset)

	// Consume message
	receivedMessage, err := consumer.Poll(100 * time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, receivedMessage)
	assert.Equal(t, message.Key, receivedMessage.Key)
	assert.Equal(t, message.Value, receivedMessage.Value)
}

// Test LocalProducer.Close
func TestLocalProducer_Close(t *testing.T) {
	producer := NewProducer("test_producer_config.yaml")
	err := producer.Close()
	assert.NoError(t, err)
}

// Test LocalConsumer.Close
func TestLocalConsumer_Close(t *testing.T) {
	consumer := NewConsumer("test_consumer_config.yaml", "")
	err := consumer.Close()
	assert.NoError(t, err)
}

// Test LocalConsumer.Commit
func TestLocalConsumer_Commit(t *testing.T) {
	consumer := NewConsumer("test_consumer_config.yaml", "")

	ctx := context.Background()
	message := &Message{
		Topic:     "test-topic",
		Key:       "test-key",
		Value:     []byte("test-value"),
		Offset:    5,
		Partition: 0,
	}

	err := consumer.Commit(ctx, message)
	assert.NoError(t, err) // Should be no-op for local implementation
}

// MockProducer for testing edge cases
type MockProducer struct{}

func (m *MockProducer) Send(ctx context.Context, message *Message) (int32, int64, error) {
	return 0, 0, nil
}

func (m *MockProducer) SendAsync(ctx context.Context, message *Message) <-chan SendResult {
	resultChan := make(chan SendResult, 1)
	go func() {
		defer close(resultChan)
		resultChan <- SendResult{
			Partition: 0,
			Offset:    0,
			Error:     nil,
		}
	}()
	return resultChan
}

func (m *MockProducer) Close() error {
	return nil
}

// Test that NewConsumer creates independent consumers
func TestNewConsumer_Independence(t *testing.T) {
	consumer1 := NewConsumer("test_consumer_config.yaml", "")
	consumer2 := NewConsumer("test_consumer_config.yaml", "")

	assert.NotNil(t, consumer1)
	assert.NotNil(t, consumer2)
	// Test that they are different instances (pointer comparison)
	assert.True(t, consumer1 != consumer2, "Consumers should be different instances")
}

// Test multiple messages in same topic
func TestLocalProducerConsumer_MultipleMessages(t *testing.T) {
	// Clean up any existing messages from previous tests
	cleanupMessageBusDir()

	producer := NewProducer("test_producer_config.yaml")
	consumer := NewConsumer("test_consumer_config.yaml", "")

	ctx := context.Background()

	// Subscribe to topic
	err := consumer.Subscribe([]string{"multi-topic"})
	assert.NoError(t, err)

	// Send multiple messages
	messages := []*Message{
		{Topic: "multi-topic", Key: "key1", Value: []byte("value1")},
		{Topic: "multi-topic", Key: "key2", Value: []byte("value2")},
		{Topic: "multi-topic", Key: "key3", Value: []byte("value3")},
	}

	for i, msg := range messages {
		partition, offset, err := producer.Send(ctx, msg)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), partition)
		assert.Equal(t, int64(i), offset) // Offset should increment
	}

	// Consume all messages
	for i := 0; i < len(messages); i++ {
		receivedMessage, err := consumer.Poll(100 * time.Millisecond)
		assert.NoError(t, err)
		assert.NotNil(t, receivedMessage)
		assert.Equal(t, messages[i].Key, receivedMessage.Key)
		assert.Equal(t, messages[i].Value, receivedMessage.Value)
		assert.Equal(t, int64(i), receivedMessage.Offset)
	}

	// Next poll should return nil (no more messages)
	noMessage, err := consumer.Poll(100 * time.Millisecond)
	assert.NoError(t, err)
	assert.Nil(t, noMessage)
}

// Test multiple topics
func TestLocalProducerConsumer_MultipleTopics(t *testing.T) {
	producer := NewProducer("test_producer_config.yaml")
	consumer := NewConsumer("test_consumer_config.yaml", "")

	ctx := context.Background()

	// Subscribe to multiple topics
	topics := []string{"topic1", "topic2", "topic3"}
	err := consumer.Subscribe(topics)
	assert.NoError(t, err)

	// Send messages to different topics
	messages := []*Message{
		{Topic: "topic1", Key: "key1", Value: []byte("value1")},
		{Topic: "topic2", Key: "key2", Value: []byte("value2")},
		{Topic: "topic3", Key: "key3", Value: []byte("value3")},
	}

	for _, msg := range messages {
		_, _, err := producer.Send(ctx, msg)
		assert.NoError(t, err)
	}

	// Consume messages (order may vary)
	consumedMessages := make([]*Message, 0, len(messages))
	for i := 0; i < len(messages); i++ {
		receivedMessage, err := consumer.Poll(100 * time.Millisecond)
		assert.NoError(t, err)
		assert.NotNil(t, receivedMessage)
		consumedMessages = append(consumedMessages, receivedMessage)
	}

	// Verify we got all messages
	assert.Len(t, consumedMessages, len(messages))
}

// Test Subscribe multiple times (updates subscription)
func TestLocalConsumer_Subscribe_Multiple(t *testing.T) {
	// Clean up any existing messages from previous tests
	cleanupMessageBusDir()

	consumer := NewConsumer("test_consumer_config.yaml", "")
	producer := NewProducer("test_producer_config.yaml")

	// First subscription
	err := consumer.Subscribe([]string{"topic1", "topic2"})
	assert.NoError(t, err)

	// Second subscription (should replace first)
	err = consumer.Subscribe([]string{"topic3", "topic4"})
	assert.NoError(t, err)

	// Send message to old topic - should not be consumed
	ctx := context.Background()
	oldTopicMessage := &Message{
		Topic: "topic1",
		Key:   "old-key",
		Value: []byte("old-value"),
	}

	_, _, err = producer.Send(ctx, oldTopicMessage)
	assert.NoError(t, err)

	// Send message to new topic - should be consumed
	newTopicMessage := &Message{
		Topic: "topic3",
		Key:   "new-key",
		Value: []byte("new-value"),
	}

	_, _, err = producer.Send(ctx, newTopicMessage)
	assert.NoError(t, err)

	// Poll should get the new topic message
	receivedMessage, err := consumer.Poll(100 * time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, receivedMessage)
	assert.Equal(t, "topic3", receivedMessage.Topic)
	assert.Equal(t, "new-key", receivedMessage.Key)
}

// Test message timestamp assignment
func TestLocalProducer_TimestampAssignment(t *testing.T) {
	producer := NewProducer("test_producer_config.yaml")
	ctx := context.Background()

	beforeTime := time.Now()

	message := &Message{
		Topic: "timestamp-topic",
		Key:   "timestamp-key",
		Value: []byte("timestamp-value"),
	}

	_, _, err := producer.Send(ctx, message)
	assert.NoError(t, err)

	afterTime := time.Now()

	// Verify timestamp was assigned during Send
	assert.False(t, message.Timestamp.IsZero())
	assert.True(t, message.Timestamp.After(beforeTime) || message.Timestamp.Equal(beforeTime))
	assert.True(t, message.Timestamp.Before(afterTime) || message.Timestamp.Equal(afterTime))
}

// Test message headers preservation
func TestLocalProducerConsumer_HeaderPreservation(t *testing.T) {
	producer := NewProducer("test_producer_config.yaml")
	consumer := NewConsumer("test_consumer_config.yaml", "")

	ctx := context.Background()

	err := consumer.Subscribe([]string{"header-topic"})
	assert.NoError(t, err)

	originalHeaders := map[string]string{
		"content-type":   "application/json",
		"source-service": "test-service",
		"correlation-id": "12345",
		"custom-header":  "custom-value",
	}

	message := &Message{
		Topic:   "header-topic",
		Key:     "header-key",
		Value:   []byte("header-value"),
		Headers: originalHeaders,
	}

	_, _, err = producer.Send(ctx, message)
	assert.NoError(t, err)

	receivedMessage, err := consumer.Poll(100 * time.Millisecond)
	assert.NoError(t, err)
	assert.NotNil(t, receivedMessage)

	// Verify all headers are preserved
	assert.Equal(t, len(originalHeaders), len(receivedMessage.Headers))
	for key, value := range originalHeaders {
		assert.Equal(t, value, receivedMessage.Headers[key])
	}
}
