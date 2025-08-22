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

// KafkaProducer Kafka implementation for production (default)
type KafkaProducer struct {
	producer *kafka.Producer
}

// NewProducer creates a new Kafka producer with configuration from YAML file
func NewProducer(configPath string) Producer {
	// Load configuration from YAML file
	configMap, err := LoadProducerConfigMap(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load producer config: %v", err))
	}

	// Create Kafka config map with values from YAML
	config := &kafka.ConfigMap{}

	// Set values from config file, with fallback defaults
	config.SetKey("bootstrap.servers", GetStringValue(configMap, "bootstrap.servers", "localhost:9092"))
	config.SetKey("client.id", GetStringValue(configMap, "client.id", os.Getenv("HOSTNAME")))
	config.SetKey("acks", GetStringValue(configMap, "acks", "1"))
	config.SetKey("retries", GetIntValue(configMap, "retries", 3))
	config.SetKey("batch.size", GetIntValue(configMap, "batch.size", 16384))
	config.SetKey("linger.ms", GetIntValue(configMap, "linger.ms", 1))
	//config.SetKey("buffer.memory", GetIntValue(configMap, "buffer.memory", 33554432))
	//config.SetKey("compression.type", GetStringValue(configMap, "compression.type", "none"))
	//config.SetKey("max.in.flight.requests.per.connection", GetIntValue(configMap, "max.in.flight.requests.per.connection", 5))
	//config.SetKey("enable.idempotence", GetBoolValue(configMap, "enable.idempotence", false))
	config.SetKey("security.protocol", GetStringValue(configMap, "security.protocol", "PLAINTEXT"))
	config.SetKey("ssl.ca.location", GetStringValue(configMap, "ssl.ca.location", ""))
	config.SetKey("ssl.certificate.location", GetStringValue(configMap, "ssl.certificate.location", ""))
	config.SetKey("ssl.key.location", GetStringValue(configMap, "ssl.key.location", ""))
	config.SetKey("enable.ssl.certificate.verification", GetBoolValue(configMap, "enable.ssl.certificate.verification", false))

	producer, err := kafka.NewProducer(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kafka producer: %v", err))
	}

	return &KafkaProducer{
		producer: producer,
	}
}

// Send sends a message to Kafka
func (p *KafkaProducer) Send(ctx context.Context, message *Message) (int32, int64, error) {
	message.Timestamp = time.Now()

	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &message.Topic,
			Partition: message.Partition,
		},
		Key:       []byte(message.Key),
		Value:     message.Value,
		Timestamp: message.Timestamp,
	}

	// Add headers
	for key, value := range message.Headers {
		kafkaMessage.Headers = append(kafkaMessage.Headers, kafka.Header{
			Key:   key,
			Value: []byte(value),
		})
	}

	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	err := p.producer.Produce(kafkaMessage, deliveryChan)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to produce message: %w", err)
	}

	// Wait for delivery report
	select {
	case event := <-deliveryChan:
		if msg, ok := event.(*kafka.Message); ok {
			if msg.TopicPartition.Error != nil {
				return 0, 0, fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
			}
			return msg.TopicPartition.Partition, int64(msg.TopicPartition.Offset), nil
		}
		return 0, 0, fmt.Errorf("unexpected event type")
	case <-ctx.Done():
		return 0, 0, ctx.Err()
	}
}

// SendAsync sends a message to Kafka asynchronously
func (p *KafkaProducer) SendAsync(ctx context.Context, message *Message) <-chan SendResult {
	resultChan := make(chan SendResult, 1)

	go func() {
		defer close(resultChan)

		message.Timestamp = time.Now()

		kafkaMessage := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &message.Topic,
				Partition: message.Partition,
			},
			Key:       []byte(message.Key),
			Value:     message.Value,
			Timestamp: message.Timestamp,
		}

		// Add headers
		for key, value := range message.Headers {
			kafkaMessage.Headers = append(kafkaMessage.Headers, kafka.Header{
				Key:   key,
				Value: []byte(value),
			})
		}

		deliveryChan := make(chan kafka.Event, 1)
		defer close(deliveryChan)

		err := p.producer.Produce(kafkaMessage, deliveryChan)
		if err != nil {
			resultChan <- SendResult{
				Partition: 0,
				Offset:    0,
				Error:     fmt.Errorf("failed to produce message: %w", err),
			}
			return
		}

		// Wait for delivery report
		select {
		case event := <-deliveryChan:
			if msg, ok := event.(*kafka.Message); ok {
				if msg.TopicPartition.Error != nil {
					resultChan <- SendResult{
						Partition: 0,
						Offset:    0,
						Error:     fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error),
					}
					return
				}
				resultChan <- SendResult{
					Partition: msg.TopicPartition.Partition,
					Offset:    int64(msg.TopicPartition.Offset),
					Error:     nil,
				}
				return
			}
			resultChan <- SendResult{
				Partition: 0,
				Offset:    0,
				Error:     fmt.Errorf("unexpected event type"),
			}
		case <-ctx.Done():
			resultChan <- SendResult{
				Partition: 0,
				Offset:    0,
				Error:     ctx.Err(),
			}
		}
	}()

	return resultChan
}

// Close closes the Kafka producer
func (p *KafkaProducer) Close() error {
	p.producer.Close()
	return nil
}

// KafkaConsumer Kafka implementation for production
type KafkaConsumer struct {
	consumer *kafka.Consumer
}

// NewConsumer creates a new Kafka consumer with configuration from YAML file
// If cgroup is not empty, it overrides the group.id from config file
func NewConsumer(configPath string, cgroup string) Consumer {
	// Load configuration from YAML file
	configMap, err := LoadConsumerConfigMap(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load consumer config: %v", err))
	}

	// Create Kafka config map with values from YAML
	config := &kafka.ConfigMap{}

	// Set values from config file, with fallback defaults
	config.SetKey("bootstrap.servers", GetStringValue(configMap, "bootstrap.servers", "localhost:9092"))

	// Use provided cgroup if not empty, otherwise use config file value
	groupID := GetStringValue(configMap, "group.id", "default-group")
	if cgroup != "" {
		groupID = cgroup
	}
	config.SetKey("group.id", groupID)
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
	config.SetKey("client.id", GetStringValue(configMap, "client.id", cgroup+os.Getenv("HOSTNAME")))
	config.SetKey("security.protocol", GetStringValue(configMap, "security.protocol", "PLAINTEXT"))
	config.SetKey("ssl.ca.location", GetStringValue(configMap, "ssl.ca.location", ""))
	config.SetKey("ssl.certificate.location", GetStringValue(configMap, "ssl.certificate.location", ""))
	config.SetKey("ssl.key.location", GetStringValue(configMap, "ssl.key.location", ""))
	config.SetKey("enable.ssl.certificate.verification", GetBoolValue(configMap, "enable.ssl.certificate.verification", false))

	consumer, err := kafka.NewConsumer(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kafka consumer: %v", err))
	}

	return &KafkaConsumer{
		consumer: consumer,
	}
}

// Subscribe subscribes to topics
func (c *KafkaConsumer) Subscribe(topics []string) error {
	return c.consumer.SubscribeTopics(topics, nil)
}

// Poll polls for messages
func (c *KafkaConsumer) Poll(timeout time.Duration) (*Message, error) {
	kafkaMessage, err := c.consumer.ReadMessage(timeout)
	if err != nil {
		if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
			return nil, nil // Timeout is not an error
		}
		return nil, err
	}

	message := &Message{
		Topic:     *kafkaMessage.TopicPartition.Topic,
		Key:       string(kafkaMessage.Key),
		Value:     kafkaMessage.Value,
		Headers:   make(map[string]string),
		Partition: kafkaMessage.TopicPartition.Partition,
		Offset:    int64(kafkaMessage.TopicPartition.Offset),
		Timestamp: kafkaMessage.Timestamp,
	}

	// Convert headers
	for _, header := range kafkaMessage.Headers {
		message.Headers[header.Key] = string(header.Value)
	}

	return message, nil
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
