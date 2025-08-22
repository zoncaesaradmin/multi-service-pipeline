package messagebus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testTopic = "test-topic"
	testValue = "test-value"
)

// Test Message struct creation and validation
func TestMessageCreation(t *testing.T) {
	message := Message{
		Topic:   testTopic,
		Key:     "test-key",
		Value:   []byte(testValue),
		Headers: map[string]string{"header1": "value1"},
	}

	assert.Equal(t, testTopic, message.Topic)
	assert.Equal(t, "test-key", message.Key)
	assert.Equal(t, []byte(testValue), message.Value)
	assert.Equal(t, "value1", message.Headers["header1"])
}

// Test Message header manipulation
func TestMessageHeaderManipulation(t *testing.T) {
	message := Message{
		Topic:   testTopic,
		Value:   []byte(testValue),
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
	config := map[string]interface{}{
		"bus_type":    "local",
		"message_dir": "/tmp/cratos-messagebus-test",
	}
	producer := NewProducer(config)
	assert.NotNil(t, producer)
	assert.Implements(t, (*Producer)(nil), producer)

	consumer := NewConsumer(config, "")
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
	config := map[string]interface{}{
		"bus_type":    "local",
		"message_dir": "/tmp/cratos-messagebus-test",
	}
	producer := NewProducer(config)
	defer producer.Close()

	// These should compile if the interface is correctly implemented
	assert.NotNil(t, producer)
	assert.Implements(t, (*Producer)(nil), producer)
}

func TestConsumerInterfaceCompliance(t *testing.T) {
	// This test ensures that our Consumer interface has the expected methods
	// We test this by creating a real consumer and checking its methods
	config := map[string]interface{}{
		"bus_type":    "local",
		"message_dir": "/tmp/cratos-messagebus-test",
	}
	consumer := NewConsumer(config, "")
	defer consumer.Close()

	// These should compile if the interface is correctly implemented
	assert.NotNil(t, consumer)
	assert.Implements(t, (*Consumer)(nil), consumer)
}

func TestAsyncSendResultTimeout(t *testing.T) {
	// Test that we can handle timeout scenarios with async sends
	resultChan := make(chan SendResult, 1)

	// Simulate async send result
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
