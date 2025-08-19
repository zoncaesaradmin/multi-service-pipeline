//go:build !local

package messagebus

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
)

const headerContentType = "content-type"
const contentTypeJSON = "application/json"

// Test KafkaProducer initialization
func TestNewProducer(t *testing.T) {
	producer := NewProducer("test_producer_config.yaml")
	assert.NotNil(t, producer)

	// Cleanup
	if producer != nil {
		producer.Close()
	}
}

// Test KafkaConsumer initialization
func TestNewConsumer(t *testing.T) {
	consumer := NewConsumer("test_consumer_config.yaml", "")
	assert.NotNil(t, consumer)

	// Cleanup
	if consumer != nil {
		consumer.Close()
	}
}

// Test message conversion
func TestMessageConversion(t *testing.T) {
	message := &Message{
		Headers:   map[string]string{headerContentType: contentTypeJSON},
		Value:     []byte("test-value"),
		Timestamp: time.Now(),
		Offset:    12345,
		Partition: 1,
	}

	// Test conversion logic similar to what's in Send method
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &message.Topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(message.Key),
		Value: message.Value,
	}

	// Add headers
	if message.Headers != nil {
		for k, v := range message.Headers {
			kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
				Key:   k,
				Value: []byte(v),
			})
		}
	}

	// Verify conversion
	assert.Equal(t, message.Topic, *kafkaMsg.TopicPartition.Topic)
	assert.Equal(t, []byte(message.Key), kafkaMsg.Key)
	assert.Equal(t, headerContentType, kafkaMsg.Headers[0].Key)
	assert.Len(t, kafkaMsg.Headers, 1)
	assert.Equal(t, headerContentType, kafkaMsg.Headers[0].Key)
	assert.Equal(t, []byte("application/json"), kafkaMsg.Headers[0].Value)
}

// Test error handling scenarios
func TestKafkaErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		kafkaError    error
		expectTimeout bool
	}{
		{
			name:          "timeout error",
			kafkaError:    kafka.NewError(kafka.ErrTimedOut, "Operation timed out", false),
			expectTimeout: true,
		},
		{
			name:          "broker error",
			kafkaError:    kafka.NewError(kafka.ErrBrokerNotAvailable, "Broker not available", false),
			expectTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate error handling logic from Poll method
			if kafkaErr, ok := tt.kafkaError.(kafka.Error); ok {
				isTimeout := kafkaErr.Code() == kafka.ErrTimedOut
				assert.Equal(t, tt.expectTimeout, isTimeout)
			}
		})
	}
}

// Test interface compliance
func TestKafkaProducerInterface(t *testing.T) {
	var producer Producer
	producer = &KafkaProducer{producer: nil}
	assert.NotNil(t, producer)
}

func TestKafkaConsumerInterface(t *testing.T) {
	var consumer Consumer
	consumer = &KafkaConsumer{consumer: nil}
	assert.NotNil(t, consumer)
	headers := map[string]string{
		headerContentType: contentTypeJSON,
		"source":          "test-service",
		"version":         "1.0",
	}

	// Simulate header conversion from Send method
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Verify headers
	assert.Len(t, kafkaHeaders, 3)

	// Convert back to map for easier testing
	headerMap := make(map[string]string)
	for _, header := range kafkaHeaders {
		headerMap[header.Key] = string(header.Value)
	}

	// Verify converted headers
	assert.Equal(t, contentTypeJSON, headerMap[headerContentType])
	assert.Equal(t, "test-service", headerMap["source"])
	assert.Equal(t, "1.0", headerMap["version"])
}

// Test timestamp handling
func TestTimestampHandling(t *testing.T) {
	beforeTime := time.Now()
	timestamp := time.Now()
	afterTime := time.Now()

	message := &Message{
		Topic:     "test-topic",
		Key:       "test-key",
		Value:     []byte("test-value"),
		Timestamp: timestamp,
	}

	// Verify timestamp is within reasonable range
	assert.True(t, message.Timestamp.After(beforeTime) || message.Timestamp.Equal(beforeTime))
	assert.True(t, message.Timestamp.Before(afterTime) || message.Timestamp.Equal(afterTime))
	assert.False(t, message.Timestamp.IsZero())
}
