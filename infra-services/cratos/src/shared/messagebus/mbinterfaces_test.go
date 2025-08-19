package messagebus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Message struct creation and validation
func TestMessage_Creation(t *testing.T) {
	message := Message{
		Topic:   "test-topic",
		Key:     "test-key",
		Value:   []byte("test-value"),
		Headers: map[string]string{"header1": "value1"},
	}

	assert.Equal(t, "test-topic", message.Topic)
	assert.Equal(t, "test-key", message.Key)
	assert.Equal(t, []byte("test-value"), message.Value)
	assert.Equal(t, "value1", message.Headers["header1"])
}

// Test Message header manipulation
func TestMessage_HeaderManipulation(t *testing.T) {
	message := Message{
		Topic:   "test-topic",
		Value:   []byte("test-value"),
		Headers: make(map[string]string),
	}

	// Test adding headers
	message.Headers["header1"] = "value1"
	message.Headers["header2"] = "value2"

	assert.Equal(t, "value1", message.Headers["header1"])
	assert.Equal(t, "value2", message.Headers["header2"])
	assert.Len(t, message.Headers, 2)
}

// Interface compliance test
func TestInterfaces(t *testing.T) {
	// Test that we can create instances that implement the interfaces
	producer := NewProducer("test_producer_config.yaml")
	assert.NotNil(t, producer)
	assert.Implements(t, (*Producer)(nil), producer)

	consumer := NewConsumer("test_consumer_config.yaml", "")
	assert.NotNil(t, consumer)
	assert.Implements(t, (*Consumer)(nil), consumer)
}

func TestSendResultCreation(t *testing.T) {
	// Test successful result
	successResult := SendResult{
		Partition: 1,
		Offset:    123,
		Error:     nil,
	}

	assert.Equal(t, int32(1), successResult.Partition)
	assert.Equal(t, int64(123), successResult.Offset)
	assert.Nil(t, successResult.Error)

	// Test error result
	err := assert.AnError
	errorResult := SendResult{
		Partition: -1,
		Offset:    -1,
		Error:     err,
	}

	assert.Equal(t, int32(-1), errorResult.Partition)
	assert.Equal(t, int64(-1), errorResult.Offset)
	assert.Equal(t, err, errorResult.Error)
}

func TestProducerInterfaceCompliance(t *testing.T) {
	// This test ensures that our Producer interface has the expected methods
	// We test this by creating a real producer and checking its methods
	producer := NewProducer("test_producer_config.yaml")
	defer producer.Close()

	// These should compile if the interface is correctly implemented
	assert.NotNil(t, producer)
	assert.Implements(t, (*Producer)(nil), producer)
}

func TestConsumerInterfaceCompliance(t *testing.T) {
	// This test ensures that our Consumer interface has the expected methods
	// We test this by creating a real consumer and checking its methods
	consumer := NewConsumer("test_consumer_config.yaml", "")
	defer consumer.Close()

	// These should compile if the interface is correctly implemented
	assert.NotNil(t, consumer)
	assert.Implements(t, (*Consumer)(nil), consumer)
}

func TestAsyncSendResultTimeout(t *testing.T) {
	// Test that we can handle timeout scenarios with async sends
	resultChan := make(chan SendResult, 1)

	// Simulate a timeout by not sending anything to the channel
	select {
	case result := <-resultChan:
		t.Errorf("Expected timeout, but got result: %+v", result)
	case <-time.After(100 * time.Millisecond):
		// Expected timeout - test passes
	}
}

func TestAsyncSendResultSuccess(t *testing.T) {
	// Test successful async result delivery
	resultChan := make(chan SendResult, 1)

	// Simulate a successful send
	go func() {
		resultChan <- SendResult{
			Partition: 2,
			Offset:    456,
			Error:     nil,
		}
	}()

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		assert.Equal(t, int32(2), result.Partition)
		assert.Equal(t, int64(456), result.Offset)
		assert.Nil(t, result.Error)
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for async result")
	}
}
