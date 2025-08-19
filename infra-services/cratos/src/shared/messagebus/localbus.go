//go:build local
// +build local

package messagebus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// File-based message storage for cross-process communication
var (
	messageBusDir = "/tmp/cratos-messagebus"
	globalMutex   = sync.RWMutex{}
)

// init ensures the message bus directory exists
func init() {
	os.MkdirAll(messageBusDir, 0755)
}

// LocalProducer file-based implementation for development
type LocalProducer struct {
	// No internal state needed - all state is in files
}

// NewProducer creates a new local producer with configuration from YAML file
func NewProducer(configPath string) Producer {
	// Load configuration from YAML file
	configMap, err := LoadProducerConfigMap(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load producer config: %v", err))
	}

	// Update messageBusDir if specified in config
	if baseDir := GetStringValue(configMap, "local.base.dir", ""); baseDir != "" {
		messageBusDir = baseDir
		// Ensure the directory exists
		os.MkdirAll(messageBusDir, 0755)
	}

	return &LocalProducer{}
}

// Send sends a message to file storage
func (p *LocalProducer) Send(ctx context.Context, message *Message) (int32, int64, error) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	message.Timestamp = time.Now()
	message.Partition = 0 // Single partition for local

	// Create topic directory if it doesn't exist
	topicDir := filepath.Join(messageBusDir, message.Topic)
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		return 0, 0, fmt.Errorf("failed to create topic directory: %w", err)
	}

	// Get current offset by counting existing files
	files, err := os.ReadDir(topicDir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read topic directory: %w", err)
	}

	offset := int64(len(files))
	message.Offset = offset

	// Write message to file
	messageData, err := json.Marshal(message)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal message: %w", err)
	}

	filename := filepath.Join(topicDir, fmt.Sprintf("%010d.json", offset))
	if err := os.WriteFile(filename, messageData, 0644); err != nil {
		return 0, 0, fmt.Errorf("failed to write message file: %w", err)
	}

	log.Printf("[MessageBus] Sent message to %s, offset %d", message.Topic, offset)
	return message.Partition, message.Offset, nil
}

// SendAsync sends a message to file storage asynchronously
func (p *LocalProducer) SendAsync(ctx context.Context, message *Message) <-chan SendResult {
	resultChan := make(chan SendResult, 1)

	go func() {
		defer close(resultChan)

		// Check for context cancellation
		select {
		case <-ctx.Done():
			resultChan <- SendResult{
				Partition: 0,
				Offset:    0,
				Error:     ctx.Err(),
			}
			return
		default:
		}

		partition, offset, err := p.Send(ctx, message)
		resultChan <- SendResult{
			Partition: partition,
			Offset:    offset,
			Error:     err,
		}
	}()

	return resultChan
}

// Close closes the local producer
func (p *LocalProducer) Close() error {
	return nil
}

// LocalConsumer file-based implementation for development
type LocalConsumer struct {
	topics   []string
	lastRead map[string]int64
	mutex    sync.RWMutex
}

// NewConsumer creates a new local consumer with configuration from YAML file
// The cgroup parameter is ignored for local implementation as it's single consumer
func NewConsumer(configPath string, cgroup string) Consumer {
	// Load configuration from YAML file
	configMap, err := LoadConsumerConfigMap(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load consumer config: %v", err))
	}

	// Create local consumer with configuration
	consumer := &LocalConsumer{
		lastRead: make(map[string]int64),
	}

	// Update messageBusDir if specified in config
	if baseDir := GetStringValue(configMap, "local.base.dir", ""); baseDir != "" {
		messageBusDir = baseDir
		// Ensure the directory exists
		os.MkdirAll(messageBusDir, 0755)
	}

	return consumer
}

// Subscribe subscribes to topics
func (c *LocalConsumer) Subscribe(topics []string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.topics = topics
	for _, topic := range topics {
		if _, exists := c.lastRead[topic]; !exists {
			c.lastRead[topic] = -1
		}
	}

	return nil
}

// Poll polls for next available message from file storage
func (c *LocalConsumer) Poll(timeout time.Duration) (*Message, error) {
	startTime := time.Now()

	for {
		c.mutex.Lock() // Use Lock instead of RLock since we might modify lastRead

		for _, topic := range c.topics {
			topicDir := filepath.Join(messageBusDir, topic)

			// Check if topic directory exists
			if _, err := os.Stat(topicDir); os.IsNotExist(err) {
				log.Printf("[MessageBus] Topic directory %s does not exist", topicDir)
				continue
			}

			// Initialize lastRead for this topic if not present
			if _, exists := c.lastRead[topic]; !exists {
				c.lastRead[topic] = -1
				log.Printf("[MessageBus] Initialized lastRead for topic %s to -1", topic)
			}

			// Get current offset for this topic
			lastOffset := c.lastRead[topic]
			nextOffset := lastOffset + 1

			log.Printf("[MessageBus] Looking for message in topic %s at offset %d (lastOffset: %d)", topic, nextOffset, lastOffset)

			// Try to read the next message file
			filename := filepath.Join(topicDir, fmt.Sprintf("%010d.json", nextOffset))
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				log.Printf("[MessageBus] Message file %s does not exist", filename)
				continue // No new messages in this topic
			}

			log.Printf("[MessageBus] Found message file %s", filename)

			// Read and parse the message
			messageData, err := os.ReadFile(filename)
			if err != nil {
				log.Printf("[MessageBus] Failed to read message file %s: %v", filename, err)
				continue // Skip this message if we can't read it
			}

			var message Message
			if err := json.Unmarshal(messageData, &message); err != nil {
				log.Printf("[MessageBus] Failed to parse message from %s: %v", filename, err)
				continue // Skip this message if we can't parse it
			}

			// Update last read offset
			c.lastRead[topic] = nextOffset
			log.Printf("[MessageBus] Consumed message from %s, offset %d", topic, nextOffset)
			c.mutex.Unlock()
			return &message, nil
		}

		c.mutex.Unlock()

		// Check if timeout reached
		if time.Since(startTime) >= timeout {
			log.Printf("[MessageBus] Timeout reached, no messages available in any subscribed topics")
			return nil, nil
		}

		// Wait a bit before trying again
		time.Sleep(100 * time.Millisecond)
	}
}

// Commit commits the offset (no-op for local)
func (c *LocalConsumer) Commit(ctx context.Context, message *Message) error {
	// Local implementation doesn't need actual commit
	return nil
}

// Close closes the local consumer
func (c *LocalConsumer) Close() error {
	return nil
}

// CleanupMessageBus removes all message files (useful for development)
func CleanupMessageBus() error {
	return os.RemoveAll(messageBusDir)
}

// GetMessageBusStats returns statistics about the message bus
func GetMessageBusStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if _, err := os.Stat(messageBusDir); os.IsNotExist(err) {
		stats["status"] = "no_messages"
		return stats
	}

	topics, err := os.ReadDir(messageBusDir)
	if err != nil {
		stats["error"] = err.Error()
		return stats
	}

	topicStats := make(map[string]int)
	for _, topic := range topics {
		if topic.IsDir() {
			topicDir := filepath.Join(messageBusDir, topic.Name())
			files, err := os.ReadDir(topicDir)
			if err == nil {
				topicStats[topic.Name()] = len(files)
			}
		}
	}

	stats["topics"] = topicStats
	stats["message_bus_dir"] = messageBusDir
	return stats
}
