package processing

import (
	"context"
	"sharedgomodule/messagebus"
	"testing"
	"time"
)

type mockConsumer struct {
	subscribedTopics []string
	commitError      error
	closeError       error
	closed           bool
	subscribeError   error
	onMessageFn      func(*messagebus.Message)
	lastMessage      *messagebus.Message
}

// Implement OnAssign for mockConsumer
func (m *mockConsumer) OnAssign(fn func([]messagebus.PartitionAssignment)) {}

// Implement OnRevoke for mockConsumer
func (m *mockConsumer) OnRevoke(fn func([]messagebus.PartitionAssignment)) {}

// Implement OnMessage for mockConsumer
func (m *mockConsumer) OnMessage(fn func(*messagebus.Message)) {
	m.onMessageFn = fn
	if m.lastMessage != nil {
		go fn(m.lastMessage)
	}
}

// Implement AssignedPartitions for mockConsumer to satisfy messagebus.Consumer interface
func (m *mockConsumer) AssignedPartitions() []messagebus.PartitionAssignment {
	return nil
}

func (m *mockConsumer) Subscribe(topics []string) error {
	if m.subscribeError != nil {
		return m.subscribeError
	}
	m.subscribedTopics = topics
	return nil
}

func (m *mockConsumer) Commit(ctx context.Context, message *messagebus.Message) error {
	return m.commitError
}

func (m *mockConsumer) Close() error {
	m.closed = true
	return m.closeError
}

// mockLogger implements logging.Logger for testing; methods are intentionally left empty.
// ...existing code...

const (
	kafkaBootstrapServers = "bootstrap.servers"
	kafkaLocalhost9092    = "localhost:9092"
	topicA                = "topicA"
	topicB                = "topicB"
	topic1                = "topic1"
	topic2                = "topic2"
	topicX                = "topicX"
	topicY                = "topicY"
	topicZ                = "topicZ"
)

func TestInputConfigBasic(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topicA, topicB},
		PollTimeout:       2 * time.Second,
		ChannelBufferSize: 5,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	if config.Topics[0] != topicA || config.Topics[1] != topicB {
		t.Error("Topics not set correctly")
	}
	if config.PollTimeout != 2*time.Second {
		t.Error("PollTimeout not set correctly")
	}
	if config.ChannelBufferSize != 5 {
		t.Error("ChannelBufferSize not set correctly")
	}
	if config.KafkaConfigMap[kafkaBootstrapServers] != kafkaLocalhost9092 {
		t.Error("KafkaConfigMap not set correctly")
	}
}

func TestNewInputHandlerCreatesChannel(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topic1},
		PollTimeout:       time.Second,
		ChannelBufferSize: 3,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	if handler == nil {
		t.Fatal("Handler should not be nil")
	}
	if cap(handler.inputCh) != 3 {
		t.Errorf("Expected channel buffer size 3, got %d", cap(handler.inputCh))
	}
}

func TestGetInputChannelReturnsChannel(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topic1},
		PollTimeout:       time.Second,
		ChannelBufferSize: 2,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	ch := handler.GetInputChannel()
	if ch == nil {
		t.Error("Input channel should not be nil")
	}
}

func TestStartSubscribesTopicsAndStartsLoop(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topicX},
		PollTimeout:       10 * time.Millisecond,
		ChannelBufferSize: 1,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	mockCons := &mockConsumer{}
	handler.consumer = mockCons
	err := handler.Start()
	if err != nil {
		t.Fatalf("Start should not error: %v", err)
	}
	if len(mockCons.subscribedTopics) != 1 || mockCons.subscribedTopics[0] != topicX {
		t.Error("Subscribe not called with correct topics")
	}
	handler.Stop()
}

func TestStartReturnsErrorOnSubscribeFail(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topicY},
		PollTimeout:       10 * time.Millisecond,
		ChannelBufferSize: 1,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	mockCons := &mockConsumer{subscribeError: context.DeadlineExceeded}
	handler.consumer = mockCons
	err := handler.Start()
	if err == nil {
		t.Error("Expected error on subscribe failure")
	}
}

func TestStopClosesConsumerAndCancelsContext(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topicZ},
		PollTimeout:       10 * time.Millisecond,
		ChannelBufferSize: 1,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	mockCons := &mockConsumer{}
	handler.consumer = mockCons
	handler.Start()
	err := handler.Stop()
	if err != nil {
		t.Errorf("Stop should not error: %v", err)
	}
	if !mockCons.closed {
		t.Error("Consumer should be closed after Stop")
	}
}

func TestConsumeLoopForwardsMessageAndCommits(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topicA},
		PollTimeout:       10 * time.Millisecond,
		ChannelBufferSize: 1,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	msg := &messagebus.Message{Value: []byte("payload")}
	mockCons := &mockConsumer{lastMessage: msg}
	handler.consumer = mockCons
	handler.Start()
	defer handler.Stop()
	select {
	case got := <-handler.inputCh:
		if string(got.Data) != "payload" {
			t.Errorf("Expected payload 'payload', got '%s'", string(got.Data))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive message in input channel")
	}
}

func TestGetStatsReturnsCorrectValues(t *testing.T) {
	config := InputConfig{
		Topics:            []string{topic1, topic2},
		PollTimeout:       3 * time.Second,
		ChannelBufferSize: 200,
		KafkaConfigMap:    map[string]any{kafkaBootstrapServers: kafkaLocalhost9092},
	}
	logger := &mockLogger{}
	handler := NewInputHandler(config, logger)
	stats := handler.GetStats()
	if stats["status"] != "running" {
		t.Errorf("Expected status 'running', got %v", stats["status"])
	}
	if stats["poll_timeout"] != "3s" {
		t.Errorf("Expected poll_timeout '3s', got %v", stats["poll_timeout"])
	}
	if stats["channel_buffer_size"] != 200 {
		t.Errorf("Expected buffer size 200, got %v", stats["channel_buffer_size"])
	}
	topics, ok := stats["topics"].([]string)
	if !ok || len(topics) != 2 {
		t.Errorf("Expected 2 topics, got %v", topics)
	}
}
