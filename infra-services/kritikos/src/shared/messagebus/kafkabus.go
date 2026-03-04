//go:build !local
// +build !local

package messagebus

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	kafkaGroupID             = "group.id"
	kafkaClientID            = "client.id"
	kafkaBootstrapServers    = "bootstrap.servers"
	kafkaSecurityProtocol    = "security.protocol"
	kafkaSSLCALocation       = "ssl.ca.location"
	kafkaSSLCertLocation     = "ssl.certificate.location"
	kafkaSSLKeyLocation      = "ssl.key.location"
	kafkaSSLCertVerification = "enable.ssl.certificate.verification"
)

// KafkaProducer Kafka implementation for production (default)
type KafkaProducer struct {
	producer *kafka.Producer
}

// Helper to check if error is transient (network, broker, etc)
func isTransientKafkaError(err error) bool {
	if err == nil {
		return false
	}
	if kafkaErr, ok := err.(kafka.Error); ok {
		switch kafkaErr.Code() {
		case kafka.ErrTransport, kafka.ErrAllBrokersDown, kafka.ErrTimedOut, kafka.ErrQueueFull, kafka.ErrUnknown:
			return true
		}
	}
	return false
}

// Helper to recreate producer on error
func (p *KafkaProducer) reconnectProducer(configMap map[string]any) error {
	config := &kafka.ConfigMap{}
	for k, v := range configMap {
		config.SetKey(k, v)
	}
	prod, err := kafka.NewProducer(config)
	if err != nil {
		return err
	}
	p.producer = prod
	return nil
}

// NewProducer creates a new Kafka producer with configuration from YAML file
func NewProducer(configMap map[string]any, clientId string) Producer {

	config := &kafka.ConfigMap{}
	if clientId == "" {
		clientId = GetStringValue(configMap, kafkaClientID, os.Getenv("HOSTNAME"))
	}

	// Set values from config file, with fallback defaults
	config.SetKey(kafkaBootstrapServers, GetStringValue(configMap, kafkaBootstrapServers, "localhost:9092"))
	config.SetKey(kafkaClientID, clientId)
	config.SetKey("acks", GetStringValue(configMap, "acks", "1"))
	config.SetKey("retries", GetIntValue(configMap, "retries", 3))
	config.SetKey("batch.size", GetIntValue(configMap, "batch.size", 16384))
	config.SetKey("linger.ms", GetIntValue(configMap, "linger.ms", 1))
	//config.SetKey("buffer.memory", GetIntValue(configMap, "buffer.memory", 33554432))
	//config.SetKey("compression.type", GetStringValue(configMap, "compression.type", "none"))
	//config.SetKey("max.in.flight.requests.per.connection", GetIntValue(configMap, "max.in.flight.requests.per.connection", 5))
	//config.SetKey("enable.idempotence", GetBoolValue(configMap, "enable.idempotence", false))
	config.SetKey(kafkaSecurityProtocol, GetStringValue(configMap, kafkaSecurityProtocol, "PLAINTEXT"))
	config.SetKey(kafkaSSLCALocation, GetStringValue(configMap, kafkaSSLCALocation, ""))
	config.SetKey(kafkaSSLCertLocation, GetStringValue(configMap, kafkaSSLCertLocation, ""))
	config.SetKey(kafkaSSLKeyLocation, GetStringValue(configMap, kafkaSSLKeyLocation, ""))
	config.SetKey(kafkaSSLCertVerification, GetBoolValue(configMap, kafkaSSLCertVerification, false))

	producer, err := kafka.NewProducer(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kafka producer: %v", err))
	}

	return &KafkaProducer{
		producer: producer,
	}
}

// Send sends a message to Kafka
// createKafkaMessage converts a Message to a kafka.Message with headers
func (p *KafkaProducer) createKafkaMessage(message *Message) *kafka.Message {
	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &message.Topic,
			Partition: message.Partition,
		},
		Key:       []byte(message.Key),
		Value:     message.Value,
		Timestamp: message.Timestamp,
	}
	for key, value := range message.Headers {
		kafkaMessage.Headers = append(kafkaMessage.Headers, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}
	return kafkaMessage
}

// waitForDelivery waits for message delivery confirmation
func (p *KafkaProducer) waitForDelivery(ctx context.Context, deliveryChan chan kafka.Event) (int32, int64, error) {
	select {
	case event := <-deliveryChan:
		if msg, ok := event.(*kafka.Message); ok {
			if msg.TopicPartition.Error != nil {
				return 0, 0, msg.TopicPartition.Error
			}
			return msg.TopicPartition.Partition, int64(msg.TopicPartition.Offset), nil
		}
		return 0, 0, fmt.Errorf("unexpected event type")
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	}
}

func (p *KafkaProducer) Send(ctx context.Context, message *Message) (int32, int64, error) {
	message.Timestamp = time.Now()
	kafkaMessage := p.createKafkaMessage(message)

	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	var lastErr error
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.producer.Produce(kafkaMessage, deliveryChan)
		if err != nil {
			if isTransientKafkaError(err) {
				lastErr = err
				time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
				continue
			}
			return 0, 0, fmt.Errorf("failed to produce message: %w", err)
		}

		partition, offset, err := p.waitForDelivery(ctx, deliveryChan)
		if err != nil {
			if isTransientKafkaError(err) {
				lastErr = err
				time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
				continue
			}
			return 0, 0, fmt.Errorf("delivery failed: %w", err)
		}
		return partition, offset, nil
	}
	return 0, 0, fmt.Errorf("Kafka Send failed after retries: %w", lastErr)
}

// SendAsync sends a message to Kafka asynchronously
// sendAsyncWithRetries handles the async send logic with retries
func (p *KafkaProducer) sendAsyncWithRetries(ctx context.Context, message *Message, resultChan chan SendResult) {
	defer close(resultChan)
	message.Timestamp = time.Now()
	kafkaMessage := p.createKafkaMessage(message)

	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	var lastErr error
	maxRetries := 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.producer.Produce(kafkaMessage, deliveryChan)
		if err != nil {
			if isTransientKafkaError(err) {
				lastErr = err
				time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
				continue
			}
			resultChan <- SendResult{
				Partition: 0,
				Offset:    0,
				Error:     fmt.Errorf("failed to produce message: %w", err),
			}
			return
		}

		partition, offset, err := p.waitForDelivery(ctx, deliveryChan)
		if err != nil {
			if ctx.Err() != nil {
				resultChan <- SendResult{Partition: 0, Offset: 0, Error: ctx.Err()}
				return
			}
			if isTransientKafkaError(err) {
				lastErr = err
				time.Sleep(time.Duration(500*(attempt+1)) * time.Millisecond)
				continue
			}
			resultChan <- SendResult{
				Partition: 0,
				Offset:    0,
				Error:     fmt.Errorf("delivery failed: %w", err),
			}
			return
		}

		resultChan <- SendResult{
			Partition: partition,
			Offset:    offset,
			Error:     nil,
		}
		return
	}

	resultChan <- SendResult{
		Partition: 0,
		Offset:    0,
		Error:     fmt.Errorf("Kafka SendAsync failed after retries: %w", lastErr),
	}
}

func (p *KafkaProducer) SendAsync(ctx context.Context, message *Message) <-chan SendResult {
	resultChan := make(chan SendResult, 1)
	go p.sendAsyncWithRetries(ctx, message, resultChan)
	return resultChan
}

// Close closes the Kafka producer
func (p *KafkaProducer) Close() error {
	p.producer.Close()
	return nil
}

type KafkaConsumer struct {
	consumer           *kafka.Consumer
	assignedPartitions []PartitionAssignment
	onAssign           func([]PartitionAssignment)
	onRevoke           func([]PartitionAssignment)
	onMessage          func(*Message)
	onError            func(error)
}

// OnMessage sets a callback for incoming messages
func (c *KafkaConsumer) OnMessage(fn func(*Message)) {
	c.onMessage = fn
}

// OnAssign sets a callback for partition assignment events
func (c *KafkaConsumer) OnAssign(fn func([]PartitionAssignment)) {
	c.onAssign = fn
}

// OnRevoke sets a callback for partition revocation events
func (c *KafkaConsumer) OnRevoke(fn func([]PartitionAssignment)) {
	c.onRevoke = fn
}

// AssignedPartitions returns the currently assigned partitions
func (c *KafkaConsumer) AssignedPartitions() []PartitionAssignment {
	return c.assignedPartitions
}

// Set callback for error events
func (c *KafkaConsumer) OnError(fn func(error)) {
	c.onError = fn
}

// Helper to check if error is transient for consumer
func isTransientKafkaConsumerError(err error) bool {
	if err == nil {
		return false
	}
	if kafkaErr, ok := err.(kafka.Error); ok {
		switch kafkaErr.Code() {
		case kafka.ErrTransport, kafka.ErrAllBrokersDown, kafka.ErrTimedOut, kafka.ErrUnknown:
			return true
		}
	}
	return false
}

// Helper to recreate consumer on error
func (c *KafkaConsumer) reconnectConsumer(configMap map[string]any, cgroup string) error {
	config := &kafka.ConfigMap{}
	for k, v := range configMap {
		config.SetKey(k, v)
	}
	groupID := GetStringValue(configMap, kafkaGroupID, "default-group")
	if cgroup != "" {
		groupID = cgroup
	}
	config.SetKey(kafkaGroupID, groupID)
	cons, err := kafka.NewConsumer(config)
	if err != nil {
		return err
	}
	c.consumer = cons
	return nil
}

// NewConsumer creates a new Kafka consumer with configuration from YAML file
// If cgroup is not empty, it overrides the group.id from config file
func NewConsumer(configMap map[string]any, cgroup string) Consumer {

	config := &kafka.ConfigMap{}

	// Set values from config file, with fallback defaults
	config.SetKey(kafkaBootstrapServers, GetStringValue(configMap, kafkaBootstrapServers, "localhost:9092"))

	// Use provided cgroup if not empty, otherwise use config file value
	groupID := GetStringValue(configMap, kafkaGroupID, "default-group")
	if cgroup != "" {
		groupID = cgroup
	}
	config.SetKey(kafkaGroupID, groupID)
	config.SetKey("auto.offset.reset", GetStringValue(configMap, "auto.offset.reset", "earliest"))
	config.SetKey("go.application.rebalance.enable", GetBoolValue(configMap, "go.application.rebalance.enable", true))
	config.SetKey("go.events.channel.enable", GetBoolValue(configMap, "go.events.channel.enable", true))
	config.SetKey("enable.auto.commit", GetBoolValue(configMap, "enable.auto.commit", false))
	config.SetKey("go.events.channel.size", GetIntValue(configMap, "go.events.channel.size", 100000))
	config.SetKey("session.timeout.ms", GetIntValue(configMap, "session.timeout.ms", 6000))
	config.SetKey("fetch.max.bytes", GetIntValue(configMap, "fetch.max.bytes", 1048576))
	//config.SetKey("session.timeout.ms", GetIntValue(configMap, "session.timeout.ms", 30000))
	//config.SetKey("heartbeat.interval.ms", GetIntValue(configMap, "heartbeat.interval.ms", 10000))
	//config.SetKey("fetch.min.bytes", GetIntValue(configMap, "fetch.min.bytes", 1))
	//config.SetKey("fetch.max.wait.ms", GetIntValue(configMap, "fetch.max.wait.ms", 500))
	//config.SetKey("max.partition.fetch.bytes", GetIntValue(configMap, "max.partition.fetch.bytes", 1048576))
	config.SetKey(kafkaClientID, GetStringValue(configMap, kafkaClientID, cgroup+os.Getenv("HOSTNAME")))
	config.SetKey(kafkaSecurityProtocol, GetStringValue(configMap, kafkaSecurityProtocol, "PLAINTEXT"))
	config.SetKey(kafkaSSLCALocation, GetStringValue(configMap, kafkaSSLCALocation, ""))
	config.SetKey(kafkaSSLCertLocation, GetStringValue(configMap, kafkaSSLCertLocation, ""))
	config.SetKey(kafkaSSLKeyLocation, GetStringValue(configMap, kafkaSSLKeyLocation, ""))
	config.SetKey(kafkaSSLCertVerification, GetBoolValue(configMap, kafkaSSLCertVerification, false))

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kafka consumer: %v", err))
	}

	kc := &KafkaConsumer{
		consumer: consumer,
	}
	go kc.handleEvents()
	return kc
}

// Set callback for partition assignment
// ...existing code...

// Internal: unified event handler for rebalance and messages
// convertToPartitionAssignments converts kafka topic partitions to PartitionAssignment slice
func convertToPartitionAssignments(partitions []kafka.TopicPartition) []PartitionAssignment {
	var assignments []PartitionAssignment
	for _, tp := range partitions {
		topic := ""
		if tp.Topic != nil {
			topic = *tp.Topic
		}
		assignments = append(assignments, PartitionAssignment{
			Topic:     topic,
			Partition: tp.Partition,
		})
	}
	return assignments
}

// handlePartitionAssignment processes partition assignment events
func (c *KafkaConsumer) handlePartitionAssignment(partitions []kafka.TopicPartition) {
	assignments := convertToPartitionAssignments(partitions)
	c.assignedPartitions = assignments
	if c.onAssign != nil {
		c.onAssign(assignments)
	}
	c.consumer.Assign(partitions)
}

// handlePartitionRevocation processes partition revocation events
func (c *KafkaConsumer) handlePartitionRevocation(partitions []kafka.TopicPartition) {
	revocations := convertToPartitionAssignments(partitions)
	if c.onRevoke != nil {
		c.onRevoke(revocations)
	}
	c.consumer.Unassign()
	c.assignedPartitions = nil
}

// convertKafkaMessage converts kafka.Message to internal Message format
func (c *KafkaConsumer) convertKafkaMessage(e *kafka.Message) *Message {
	msg := &Message{
		Topic:     *e.TopicPartition.Topic,
		Key:       string(e.Key),
		Value:     e.Value,
		Headers:   make(map[string]string),
		Partition: e.TopicPartition.Partition,
		Offset:    int64(e.TopicPartition.Offset),
		Timestamp: e.Timestamp,
	}
	for _, header := range e.Headers {
		msg.Headers[header.Key] = string(header.Value)
	}
	return msg
}

func (c *KafkaConsumer) handleEvents() {
	for event := range c.consumer.Events() {
		switch e := event.(type) {
		case kafka.AssignedPartitions:
			c.handlePartitionAssignment(e.Partitions)
		case kafka.RevokedPartitions:
			c.handlePartitionRevocation(e.Partitions)
		case *kafka.Message:
			msg := c.convertKafkaMessage(e)
			if c.onMessage != nil {
				c.onMessage(msg)
			}
		case kafka.Error:
			if c.onError != nil {
				c.onError(e)
			}
		}
	}
}

// Subscribe subscribes to topics
func (c *KafkaConsumer) Subscribe(topics []string) error {
	return c.consumer.SubscribeTopics(topics, nil)
}

// Commit manually commits the offset
func (c *KafkaConsumer) Commit(ctx context.Context, message *Message) error {
	topicPartition := kafka.TopicPartition{
		Topic:     &message.Topic,
		Partition: message.Partition,
		Offset:    kafka.Offset(message.Offset + 1),
	}

	_, err := c.consumer.CommitOffsets([]kafka.TopicPartition{topicPartition})
	return err
}

// Close closes the Kafka consumer
func (c *KafkaConsumer) Close() error {
	return c.consumer.Close()
}
