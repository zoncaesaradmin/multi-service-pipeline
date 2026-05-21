//go:build !local
// +build !local

package messagebus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
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

	defaultMaxSendRetries   = 5
	defaultReconnectBackoff = 500 * time.Millisecond
	maxReconnectBackoff     = 5 * time.Second
	defaultDeliveryChanSize = 2000
)

// KafkaProducer Kafka implementation for production (default)
type KafkaProducer struct {
	configMap map[string]any
	clientID  string

	producerMu sync.RWMutex
	producer   *kafka.Producer

	deliveryNotifChan chan WriteConfirmNotif
	closeDone         chan struct{}
	closeOnce         sync.Once
	drainWG           sync.WaitGroup
}

func cloneConfigMap(configMap map[string]any) map[string]any {
	if configMap == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(configMap))
	for k, v := range configMap {
		cloned[k] = v
	}
	return cloned
}

func isClosed(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func sleepWithContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func sleepUntilClosed(done <-chan struct{}, wait time.Duration) bool {
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-done:
		return false
	case <-timer.C:
		return true
	}
}

func retryBackoff(attempt int) time.Duration {
	backoff := time.Duration(attempt+1) * defaultReconnectBackoff
	if backoff > maxReconnectBackoff {
		return maxReconnectBackoff
	}
	return backoff
}

func kafkaErrorOf(err error) (kafka.Error, bool) {
	if err == nil {
		return kafka.Error{}, false
	}

	var kafkaErr kafka.Error
	if errors.As(err, &kafkaErr) {
		return kafkaErr, true
	}
	return kafka.Error{}, false
}

func isRetryableKafkaError(err error) bool {
	if err == nil {
		return false
	}

	kafkaErr, ok := kafkaErrorOf(err)
	if !ok {
		return false
	}

	if kafkaErr.IsRetriable() || kafkaErr.IsTimeout() {
		return true
	}

	switch kafkaErr.Code() {
	case kafka.ErrTransport, kafka.ErrAllBrokersDown, kafka.ErrTimedOut,
		kafka.ErrTimedOutQueue, kafka.ErrQueueFull, kafka.ErrUnknown:
		return true
	default:
		return false
	}
}

func shouldRecreateKafkaClient(err error) bool {
	if err == nil {
		return false
	}

	kafkaErr, ok := kafkaErrorOf(err)
	if ok {
		if kafkaErr.IsFatal() {
			return true
		}

		switch kafkaErr.Code() {
		case kafka.ErrSsl, kafka.ErrAuthentication:
			return true
		}
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "ssl") ||
		strings.Contains(lower, "certificate") ||
		strings.Contains(lower, "x509") ||
		strings.Contains(lower, "authentication")
}

func buildProducerConfig(configMap map[string]any, clientID string) *kafka.ConfigMap {
	config := &kafka.ConfigMap{}
	if clientID == "" {
		clientID = GetStringValue(configMap, kafkaClientID, os.Getenv("HOSTNAME"))
	}

	config.SetKey(kafkaBootstrapServers, GetStringValue(configMap, kafkaBootstrapServers, "localhost:9092"))
	config.SetKey(kafkaClientID, clientID)
	config.SetKey("acks", GetStringValue(configMap, "acks", "1"))
	config.SetKey("retries", GetIntValue(configMap, "retries", 3))
	config.SetKey("batch.size", GetIntValue(configMap, "batch.size", 16384))
	config.SetKey("linger.ms", GetIntValue(configMap, "linger.ms", 1))
	config.SetKey("go.events.channel.size", GetIntValue(configMap, "go.events.channel.size", 100000))
	config.SetKey("go.produce.channel.size", GetIntValue(configMap, "go.produce.channel.size", 100000))
	config.SetKey(kafkaSecurityProtocol, GetStringValue(configMap, kafkaSecurityProtocol, "PLAINTEXT"))
	config.SetKey(kafkaSSLCALocation, GetStringValue(configMap, kafkaSSLCALocation, ""))
	config.SetKey(kafkaSSLCertLocation, GetStringValue(configMap, kafkaSSLCertLocation, ""))
	config.SetKey(kafkaSSLKeyLocation, GetStringValue(configMap, kafkaSSLKeyLocation, ""))
	config.SetKey(kafkaSSLCertVerification, GetBoolValue(configMap, kafkaSSLCertVerification, false))
	return config
}

func buildConsumerConfig(configMap map[string]any, cgroup string) *kafka.ConfigMap {
	config := &kafka.ConfigMap{}
	groupID := GetStringValue(configMap, kafkaGroupID, "default-group")
	if cgroup != "" {
		groupID = cgroup
	}

	config.SetKey(kafkaBootstrapServers, GetStringValue(configMap, kafkaBootstrapServers, "localhost:9092"))
	config.SetKey(kafkaGroupID, groupID)
	config.SetKey("auto.offset.reset", GetStringValue(configMap, "auto.offset.reset", "earliest"))
	config.SetKey("go.application.rebalance.enable", GetBoolValue(configMap, "go.application.rebalance.enable", true))
	config.SetKey("go.events.channel.enable", GetBoolValue(configMap, "go.events.channel.enable", true))
	config.SetKey("enable.auto.commit", GetBoolValue(configMap, "enable.auto.commit", false))
	config.SetKey("go.events.channel.size", GetIntValue(configMap, "go.events.channel.size", 100000))
	config.SetKey("queued.max.messages.kbytes", GetIntValue(configMap, "queued.max.messages.kbytes", 65536))
	config.SetKey("session.timeout.ms", GetIntValue(configMap, "session.timeout.ms", 6000))
	config.SetKey("fetch.max.bytes", GetIntValue(configMap, "fetch.max.bytes", 1048576))
	config.SetKey(kafkaClientID, GetStringValue(configMap, kafkaClientID, cgroup+os.Getenv("HOSTNAME")))
	config.SetKey(kafkaSecurityProtocol, GetStringValue(configMap, kafkaSecurityProtocol, "PLAINTEXT"))
	config.SetKey(kafkaSSLCALocation, GetStringValue(configMap, kafkaSSLCALocation, ""))
	config.SetKey(kafkaSSLCertLocation, GetStringValue(configMap, kafkaSSLCertLocation, ""))
	config.SetKey(kafkaSSLKeyLocation, GetStringValue(configMap, kafkaSSLKeyLocation, ""))
	config.SetKey(kafkaSSLCertVerification, GetBoolValue(configMap, kafkaSSLCertVerification, false))
	return config
}

func createKafkaProducer(configMap map[string]any, clientID string) (*kafka.Producer, error) {
	return kafka.NewProducer(buildProducerConfig(configMap, clientID))
}

func createKafkaConsumer(configMap map[string]any, cgroup string) (*kafka.Consumer, error) {
	return kafka.NewConsumer(buildConsumerConfig(configMap, cgroup))
}

// NewProducer creates a new Kafka producer with configuration from YAML file.
// The underlying Kafka client is created lazily so transient startup issues do
// not panic the process before the first actual write.
func NewProducer(configMap map[string]any, clientID string) Producer {
	return &KafkaProducer{
		configMap:         cloneConfigMap(configMap),
		clientID:          clientID,
		deliveryNotifChan: make(chan WriteConfirmNotif, defaultDeliveryChanSize),
		closeDone:         make(chan struct{}),
	}
}

func (p *KafkaProducer) startDeliveryReportDrainer(producer *kafka.Producer) {
	p.drainWG.Add(1)
	go func() {
		defer p.drainWG.Done()
		for ev := range producer.Events() {
			switch e := ev.(type) {
			case *kafka.Message:
				inputMetaData, ok := e.Opaque.(*WriteConfirmNotif)
				if !ok {
					fmt.Printf("Error: Opaque field not WriteConfirmNotif type, or missing: %v\n", e.Opaque)
					continue
				}

				dc := *inputMetaData
				dc.WriteError = e.TopicPartition.Error
				select {
				case p.deliveryNotifChan <- dc:
				case <-p.closeDone:
					return
				}

			case kafka.Error:
				fmt.Printf("Producer kafka error: %v\n", e)
			}
		}
	}()
}

func (p *KafkaProducer) ensureProducer() (*kafka.Producer, error) {
	p.producerMu.RLock()
	if p.producer != nil {
		producer := p.producer
		p.producerMu.RUnlock()
		return producer, nil
	}
	p.producerMu.RUnlock()

	p.producerMu.Lock()
	defer p.producerMu.Unlock()

	if p.producer != nil {
		return p.producer, nil
	}
	if isClosed(p.closeDone) {
		return nil, fmt.Errorf("kafka producer is closed")
	}

	producer, err := createKafkaProducer(p.configMap, p.clientID)
	if err != nil {
		return nil, err
	}

	p.producer = producer
	p.startDeliveryReportDrainer(producer)
	return producer, nil
}

func (p *KafkaProducer) reconnectProducer() error {
	if isClosed(p.closeDone) {
		return fmt.Errorf("kafka producer is closed")
	}

	newProducer, err := createKafkaProducer(p.configMap, p.clientID)
	if err != nil {
		return err
	}

	p.producerMu.Lock()
	if isClosed(p.closeDone) {
		p.producerMu.Unlock()
		newProducer.Close()
		return fmt.Errorf("kafka producer is closed")
	}
	oldProducer := p.producer
	p.producer = newProducer
	p.producerMu.Unlock()

	p.startDeliveryReportDrainer(newProducer)
	if oldProducer != nil {
		oldProducer.Close()
	}
	return nil
}

func (p *KafkaProducer) recreateProducerIfNeeded(err error) {
	if !shouldRecreateKafkaClient(err) && !isRetryableKafkaError(err) {
		return
	}
	if reconnectErr := p.reconnectProducer(); reconnectErr != nil {
		fmt.Printf("Producer reconnect failed after error %v: %v\n", err, reconnectErr)
	}
}

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

func (p *KafkaProducer) waitForDelivery(ctx context.Context, deliveryChan chan kafka.Event) (int32, int64, error) {
	select {
	case event, ok := <-deliveryChan:
		if !ok {
			return 0, 0, fmt.Errorf("delivery channel closed before confirmation")
		}
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
	for attempt := 0; attempt < defaultMaxSendRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}

		producer, err := p.ensureProducer()
		if err != nil {
			lastErr = err
		} else {
			err = producer.Produce(kafkaMessage, deliveryChan)
			if err == nil {
				partition, offset, deliveryErr := p.waitForDelivery(ctx, deliveryChan)
				if deliveryErr == nil {
					return partition, offset, nil
				}
				err = deliveryErr
			}
			lastErr = err
			p.recreateProducerIfNeeded(err)
			if !isRetryableKafkaError(err) && !shouldRecreateKafkaClient(err) {
				return 0, 0, fmt.Errorf("failed to produce message: %w", err)
			}
		}

		if attempt == defaultMaxSendRetries-1 {
			break
		}
		if err := sleepWithContext(ctx, retryBackoff(attempt)); err != nil {
			return 0, 0, err
		}
	}

	return 0, 0, fmt.Errorf("Kafka Send failed after retries: %w", lastErr)
}

func (p *KafkaProducer) ProduceAsync(ctx context.Context, message *Message, inputCorrData WriteConfirmNotif) error {
	message.Timestamp = time.Now()
	kafkaMessage := p.createKafkaMessage(message)
	kafkaMessage.Opaque = &inputCorrData

	var lastErr error
	for attempt := 0; attempt < defaultMaxSendRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		producer, err := p.ensureProducer()
		if err != nil {
			lastErr = err
		} else {
			err = producer.Produce(kafkaMessage, nil)
			if err == nil {
				return nil
			}
			lastErr = err
			p.recreateProducerIfNeeded(err)
			if !isRetryableKafkaError(err) && !shouldRecreateKafkaClient(err) {
				return fmt.Errorf("failed to produce message asynchronously: %w", err)
			}
		}

		if attempt == defaultMaxSendRetries-1 {
			break
		}
		if err := sleepWithContext(ctx, retryBackoff(attempt)); err != nil {
			return err
		}
	}

	return fmt.Errorf("Kafka ProduceAsync failed after retries: %w", lastErr)
}

func (p *KafkaProducer) GetDeliveryNotifChannel() <-chan WriteConfirmNotif {
	return p.deliveryNotifChan
}

func (p *KafkaProducer) SendAsync(ctx context.Context, message *Message) <-chan SendResult {
	resultChan := make(chan SendResult, 1)
	go func() {
		defer close(resultChan)
		partition, offset, err := p.Send(ctx, message)
		resultChan <- SendResult{Partition: partition, Offset: offset, Error: err}
	}()
	return resultChan
}

// Close closes the Kafka producer.
func (p *KafkaProducer) Close() error {
	p.closeOnce.Do(func() {
		close(p.closeDone)

		p.producerMu.Lock()
		producer := p.producer
		p.producer = nil
		p.producerMu.Unlock()

		if producer != nil {
			producer.Close()
		}

		p.drainWG.Wait()
		close(p.deliveryNotifChan)
	})
	return nil
}

type KafkaConsumer struct {
	configMap map[string]any
	cgroup    string

	consumerMu         sync.RWMutex
	consumer           *kafka.Consumer
	topics             []string
	assignedPartitions []PartitionAssignment

	onAssign  func([]PartitionAssignment)
	onRevoke  func([]PartitionAssignment)
	onMessage func(*Message)
	onError   func(error)

	closeDone    chan struct{}
	closeOnce    sync.Once
	eventsWG     sync.WaitGroup
	reconnectMu  sync.Mutex
	reconnecting bool
}

// OnMessage sets a callback for incoming messages.
func (c *KafkaConsumer) OnMessage(fn func(*Message)) {
	c.consumerMu.Lock()
	defer c.consumerMu.Unlock()
	c.onMessage = fn
}

// OnAssign sets a callback for partition assignment events.
func (c *KafkaConsumer) OnAssign(fn func([]PartitionAssignment)) {
	c.consumerMu.Lock()
	defer c.consumerMu.Unlock()
	c.onAssign = fn
}

// OnRevoke sets a callback for partition revocation events.
func (c *KafkaConsumer) OnRevoke(fn func([]PartitionAssignment)) {
	c.consumerMu.Lock()
	defer c.consumerMu.Unlock()
	c.onRevoke = fn
}

// AssignedPartitions returns the currently assigned partitions.
func (c *KafkaConsumer) AssignedPartitions() []PartitionAssignment {
	c.consumerMu.RLock()
	defer c.consumerMu.RUnlock()

	partitions := make([]PartitionAssignment, len(c.assignedPartitions))
	copy(partitions, c.assignedPartitions)
	return partitions
}

// Set callback for error events.
func (c *KafkaConsumer) OnError(fn func(error)) {
	c.consumerMu.Lock()
	defer c.consumerMu.Unlock()
	c.onError = fn
}

func (c *KafkaConsumer) emitError(err error) {
	c.consumerMu.RLock()
	onError := c.onError
	c.consumerMu.RUnlock()

	if onError != nil {
		onError(err)
	}
}

// NewConsumer creates a new Kafka consumer with configuration from YAML file.
// The actual Kafka client is created lazily on Subscribe so temporary startup
// issues do not panic the process.
func NewConsumer(configMap map[string]any, cgroup string) Consumer {
	return &KafkaConsumer{
		configMap: cloneConfigMap(configMap),
		cgroup:    cgroup,
		closeDone: make(chan struct{}),
	}
}

func (c *KafkaConsumer) ensureConsumer() (*kafka.Consumer, error) {
	c.consumerMu.RLock()
	if c.consumer != nil {
		consumer := c.consumer
		c.consumerMu.RUnlock()
		return consumer, nil
	}
	c.consumerMu.RUnlock()

	consumer, err := createKafkaConsumer(c.configMap, c.cgroup)
	if err != nil {
		return nil, err
	}

	c.consumerMu.Lock()
	if isClosed(c.closeDone) {
		c.consumerMu.Unlock()
		consumer.Close()
		return nil, fmt.Errorf("kafka consumer is closed")
	}
	if c.consumer != nil {
		existing := c.consumer
		c.consumerMu.Unlock()
		consumer.Close()
		return existing, nil
	}

	c.consumer = consumer
	c.consumerMu.Unlock()
	c.startEventLoop(consumer)
	return consumer, nil
}

func (c *KafkaConsumer) subscribeTopics(consumer *kafka.Consumer, topics []string) error {
	if len(topics) == 0 {
		return consumer.SubscribeTopics(nil, nil)
	}
	return consumer.SubscribeTopics(topics, nil)
}

func (c *KafkaConsumer) reconnectConsumer() error {
	if isClosed(c.closeDone) {
		return fmt.Errorf("kafka consumer is closed")
	}

	newConsumer, err := createKafkaConsumer(c.configMap, c.cgroup)
	if err != nil {
		return err
	}

	c.consumerMu.RLock()
	topics := append([]string(nil), c.topics...)
	c.consumerMu.RUnlock()

	if err := c.subscribeTopics(newConsumer, topics); err != nil {
		newConsumer.Close()
		return err
	}

	c.consumerMu.Lock()
	if isClosed(c.closeDone) {
		c.consumerMu.Unlock()
		newConsumer.Close()
		return fmt.Errorf("kafka consumer is closed")
	}
	oldConsumer := c.consumer
	c.consumer = newConsumer
	c.assignedPartitions = nil
	c.consumerMu.Unlock()

	c.startEventLoop(newConsumer)
	if oldConsumer != nil {
		oldConsumer.Close()
	}
	return nil
}

func (c *KafkaConsumer) scheduleReconnect(triggerErr error) {
	c.reconnectMu.Lock()
	if c.reconnecting || isClosed(c.closeDone) {
		c.reconnectMu.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectMu.Unlock()

	go func() {
		defer func() {
			c.reconnectMu.Lock()
			c.reconnecting = false
			c.reconnectMu.Unlock()
		}()

		for attempt := 0; ; attempt++ {
			if err := c.reconnectConsumer(); err == nil {
				return
			} else {
				c.emitError(fmt.Errorf("kafka consumer reconnect after %v failed: %w", triggerErr, err))
			}

			if !sleepUntilClosed(c.closeDone, retryBackoff(attempt)) {
				return
			}
		}
	}()
}

func convertToPartitionAssignments(partitions []kafka.TopicPartition) []PartitionAssignment {
	assignments := make([]PartitionAssignment, 0, len(partitions))
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

func (c *KafkaConsumer) handlePartitionAssignment(consumer *kafka.Consumer, partitions []kafka.TopicPartition) {
	assignments := convertToPartitionAssignments(partitions)

	c.consumerMu.Lock()
	if c.consumer != consumer {
		c.consumerMu.Unlock()
		return
	}
	c.assignedPartitions = assignments
	onAssign := c.onAssign
	c.consumerMu.Unlock()

	if onAssign != nil {
		onAssign(assignments)
	}
	if err := consumer.Assign(partitions); err != nil {
		c.emitError(err)
	}
}

func (c *KafkaConsumer) handlePartitionRevocation(consumer *kafka.Consumer, partitions []kafka.TopicPartition) {
	revocations := convertToPartitionAssignments(partitions)

	c.consumerMu.Lock()
	if c.consumer != consumer {
		c.consumerMu.Unlock()
		return
	}
	c.assignedPartitions = nil
	onRevoke := c.onRevoke
	c.consumerMu.Unlock()

	if onRevoke != nil {
		onRevoke(revocations)
	}
	if err := consumer.Unassign(); err != nil {
		c.emitError(err)
	}
}

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

func (c *KafkaConsumer) startEventLoop(consumer *kafka.Consumer) {
	c.eventsWG.Add(1)
	go func() {
		defer c.eventsWG.Done()

		for event := range consumer.Events() {
			switch e := event.(type) {
			case kafka.AssignedPartitions:
				c.handlePartitionAssignment(consumer, e.Partitions)
			case kafka.RevokedPartitions:
				c.handlePartitionRevocation(consumer, e.Partitions)
			case kafka.OffsetsCommitted:
				// Drain commit events to avoid buildup.
			case kafka.Stats:
				// Drain stats events to avoid buildup.
			case *kafka.Message:
				msg := c.convertKafkaMessage(e)
				c.consumerMu.RLock()
				onMessage := c.onMessage
				c.consumerMu.RUnlock()
				if onMessage != nil {
					onMessage(msg)
				}
			case kafka.Error:
				c.emitError(e)
				if shouldRecreateKafkaClient(e) {
					c.scheduleReconnect(e)
				}
			}
		}
	}()
}

// Subscribe subscribes to topics.
func (c *KafkaConsumer) Subscribe(topics []string) error {
	c.consumerMu.Lock()
	c.topics = append([]string(nil), topics...)
	c.consumerMu.Unlock()

	consumer, err := c.ensureConsumer()
	if err != nil {
		return fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	if err := c.subscribeTopics(consumer, topics); err != nil {
		if shouldRecreateKafkaClient(err) || isRetryableKafkaError(err) {
			if reconnectErr := c.reconnectConsumer(); reconnectErr == nil {
				return nil
			}
		}
		return err
	}
	return nil
}

// Commit manually commits the offset.
func (c *KafkaConsumer) Commit(ctx context.Context, message *Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	consumer, err := c.ensureConsumer()
	if err != nil {
		return fmt.Errorf("failed to get kafka consumer for commit: %w", err)
	}

	topicPartition := kafka.TopicPartition{
		Topic:     &message.Topic,
		Partition: message.Partition,
		Offset:    kafka.Offset(message.Offset + 1),
	}

	_, err = consumer.CommitOffsets([]kafka.TopicPartition{topicPartition})
	if err != nil && shouldRecreateKafkaClient(err) {
		c.scheduleReconnect(err)
	}
	return err
}

// Close closes the Kafka consumer.
func (c *KafkaConsumer) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		close(c.closeDone)

		c.consumerMu.Lock()
		consumer := c.consumer
		c.consumer = nil
		c.assignedPartitions = nil
		c.consumerMu.Unlock()

		if consumer != nil {
			closeErr = consumer.Close()
		}
		c.eventsWG.Wait()
	})
	return closeErr
}
