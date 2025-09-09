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
	"strconv"
	"strings"
	"sync"
	"time"
)

// File-based message storage for cross-process communication
var (
	messageBusDir = "/tmp/cratos-messagebus-test"
	offsetDir     = "/tmp/cratos-messagebus-offsets"
	globalMutex   = sync.RWMutex{}
)

// init ensures the message bus directory exists
func init() {
	os.MkdirAll(messageBusDir, 0755)
	os.MkdirAll(offsetDir, 0755)
}

// LocalProducer file-based implementation for development
type LocalProducer struct {
	// No internal state needed - all state is in files
}

// NewProducer creates a new local producer with configuration from YAML file
func NewProducer(configMap map[string]any, clientId string) Producer {
	// Update messageBusDir if specified in config
	if baseDir := GetStringValue(configMap, "local.base.dir", ""); baseDir != "" {
		messageBusDir = baseDir
		offsetDir = filepath.Join(baseDir, "offsets")
		// Ensure the directories exist
		os.MkdirAll(messageBusDir, 0755)
		os.MkdirAll(offsetDir, 0755)
	}

	return &LocalProducer{}
}

// Send sends a message to file storage
func (p *LocalProducer) Send(ctx context.Context, message *Message) (int32, int64, error) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	message.Timestamp = time.Now()
	message.Partition = 0 // Single partition for local

	topicDir := filepath.Join(messageBusDir, message.Topic)
	log.Printf("[MessageBus] Producer using topicDir: %s", topicDir)
	if err := os.MkdirAll(topicDir, 0755); err != nil {
		log.Printf("[MessageBus] Producer failed to create topicDir: %v", err)
		return 0, 0, fmt.Errorf("failed to create topic directory: %w", err)
	}

	files, err := os.ReadDir(topicDir)
	if err != nil {
		log.Printf("[MessageBus] Producer failed to read topicDir: %v", err)
		return 0, 0, fmt.Errorf("failed to read topic directory: %w", err)
	}

	offset := int64(len(files))
	message.Offset = offset

	messageData, err := json.Marshal(message)
	if err != nil {
		log.Printf("[MessageBus] Producer failed to marshal message: %v", err)
		return 0, 0, fmt.Errorf("failed to marshal message: %w", err)
	}

	filename := filepath.Join(topicDir, fmt.Sprintf("%010d.json", offset))
	log.Printf("[MessageBus] Producer writing file: %s", filename)
	if err := os.WriteFile(filename, messageData, 0644); err != nil {
		log.Printf("[MessageBus] Producer failed to write file: %v", err)
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
	topics        []string
	lastRead      map[string]int64
	mutex         sync.RWMutex
	onMessage     func(*Message)
	onAssign      func([]PartitionAssignment)
	onRevoke      func([]PartitionAssignment)
	assigned      []PartitionAssignment
	watcherCancel context.CancelFunc
	watcherDone   chan struct{}
	consumerGroup string // Store consumer group for persistent offset tracking
}

// NewConsumer creates a new local consumer with configuration from YAML file
// The cgroup parameter is used for persistent offset tracking
func NewConsumer(configMap map[string]any, cgroup string) Consumer {
	// Create local consumer with configuration
	consumer := &LocalConsumer{
		lastRead:      make(map[string]int64),
		assigned:      []PartitionAssignment{},
		watcherDone:   make(chan struct{}),
		consumerGroup: cgroup,
	}

	// Update messageBusDir if specified in config
	if baseDir := GetStringValue(configMap, "local.base.dir", ""); baseDir != "" {
		messageBusDir = baseDir
		offsetDir = filepath.Join(baseDir, "offsets")
		// Ensure the directories exist
		os.MkdirAll(messageBusDir, 0755)
		os.MkdirAll(offsetDir, 0755)
	}

	return consumer
}

// getOffsetFilePath returns the path to the offset file for a consumer group and topic
func (c *LocalConsumer) getOffsetFilePath(topic string) string {
	// Handle empty consumer group by using a default
	consumerGroup := c.consumerGroup
	if consumerGroup == "" {
		consumerGroup = "default-consumer-group"
	}

	// Use consumer group to create unique offset files per group
	groupDir := filepath.Join(offsetDir, consumerGroup)
	os.MkdirAll(groupDir, 0755)
	return filepath.Join(groupDir, fmt.Sprintf("%s.offset", topic))
}

// loadLastReadOffset loads the last read offset for a topic from persistent storage
func (c *LocalConsumer) loadLastReadOffset(topic string) int64 {
	offsetFile := c.getOffsetFilePath(topic)
	data, err := os.ReadFile(offsetFile)
	if err != nil {
		// File doesn't exist or can't be read, start from -1 (will read from offset 0)
		// Use debug level logging to avoid confusion in tests
		log.Printf("[MessageBus] DEBUG: No offset file found for topic %s, starting from beginning", topic)
		return -1
	}

	offsetStr := strings.TrimSpace(string(data))
	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		log.Printf("[MessageBus] WARNING: Invalid offset in file for topic %s: %v, starting from beginning", topic, err)
		return -1
	}

	log.Printf("[MessageBus] DEBUG: Loaded offset %d for topic %s from persistent storage", offset, topic)
	return offset
} // saveLastReadOffset saves the last read offset for a topic to persistent storage
func (c *LocalConsumer) saveLastReadOffset(topic string, offset int64) error {
	offsetFile := c.getOffsetFilePath(topic)
	data := fmt.Sprintf("%d\n", offset)
	err := os.WriteFile(offsetFile, []byte(data), 0644)
	if err != nil {
		log.Printf("[MessageBus] Failed to save offset %d for topic %s: %v", offset, topic, err)
		return err
	}
	log.Printf("[MessageBus] Saved offset %d for topic %s to persistent storage", offset, topic)
	return nil
}

// Subscribe subscribes to topics
func (c *LocalConsumer) Subscribe(topics []string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Stop previous watcher if running
	if c.watcherCancel != nil {
		c.watcherCancel()
		<-c.watcherDone
		c.watcherCancel = nil
		c.watcherDone = make(chan struct{})
	}

	// Track previous assignments for OnRevoke
	prevAssigned := c.assigned

	c.topics = topics
	newTopics := make(map[string]struct{})
	newAssigned := make([]PartitionAssignment, 0, len(topics))
	for _, topic := range topics {
		newTopics[topic] = struct{}{}
		// Load persistent offset or initialize to -1 if not exists
		if _, exists := c.lastRead[topic]; !exists {
			c.lastRead[topic] = c.loadLastReadOffset(topic)
		}
		// Only one partition for local bus
		newAssigned = append(newAssigned, PartitionAssignment{Topic: topic, Partition: 0})
	}
	for topic := range c.lastRead {
		if _, ok := newTopics[topic]; !ok {
			delete(c.lastRead, topic)
		}
	}
	c.assigned = newAssigned

	// Fire OnRevoke for previous assignments if changed
	if c.onRevoke != nil && len(prevAssigned) > 0 {
		c.onRevoke(prevAssigned)
	}
	// Fire OnAssign for new assignments
	if c.onAssign != nil && len(newAssigned) > 0 {
		c.onAssign(newAssigned)
	}

	// Start watcher goroutine for event-driven OnMessage
	ctx, cancel := context.WithCancel(context.Background())
	c.watcherCancel = cancel
	go c.runWatcher(ctx)
	return nil
}

// OnMessage sets a callback for incoming messages
func (c *LocalConsumer) OnMessage(fn func(*Message)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.onMessage = fn
}

// OnAssign sets a callback for partition assignment events
func (c *LocalConsumer) OnAssign(fn func([]PartitionAssignment)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.onAssign = fn
}

// OnRevoke sets a callback for partition revocation events
func (c *LocalConsumer) OnRevoke(fn func([]PartitionAssignment)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.onRevoke = fn
}

// AssignedPartitions returns the currently assigned partitions
func (c *LocalConsumer) AssignedPartitions() []PartitionAssignment {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return append([]PartitionAssignment{}, c.assigned...)
}

// runWatcher continuously checks for new messages and triggers OnMessage
func (c *LocalConsumer) runWatcher(ctx context.Context) {
	if c.watcherDone != nil {
		close(c.watcherDone)
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mutex.Lock()
			c.pollTopicsForMessages()
			c.mutex.Unlock()
		}
	}
}

// pollTopicsForMessages checks all topics for new messages and triggers OnMessage
func (c *LocalConsumer) pollTopicsForMessages() {
	for _, topic := range c.topics {
		topicDir := filepath.Join(messageBusDir, topic)
		if _, err := os.Stat(topicDir); os.IsNotExist(err) {
			continue
		}
		if _, exists := c.lastRead[topic]; !exists {
			c.lastRead[topic] = c.loadLastReadOffset(topic)
		}
		lastOffset := c.lastRead[topic]
		nextOffset := lastOffset + 1
		filename := filepath.Join(topicDir, fmt.Sprintf("%010d.json", nextOffset))
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			continue
		}
		messageData, err := os.ReadFile(filename)
		if err != nil {
			log.Printf("[MessageBus] Failed to read message file %s: %v", filename, err)
			continue
		}
		var message Message
		if err := json.Unmarshal(messageData, &message); err != nil {
			log.Printf("[MessageBus] Failed to unmarshal message from %s: %v", filename, err)
			continue
		}

		// Update in-memory offset
		c.lastRead[topic] = nextOffset

		// Save offset to persistent storage
		if err := c.saveLastReadOffset(topic, nextOffset); err != nil {
			log.Printf("[MessageBus] Failed to save offset for topic %s: %v", topic, err)
		}

		log.Printf("[MessageBus] Consumer read message from topic %s at offset %d", topic, nextOffset)

		if c.onMessage != nil {
			go c.onMessage(&message)
		}
	}
}

// Commit commits the offset (saves to persistent storage for local)
func (c *LocalConsumer) Commit(ctx context.Context, message *Message) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Update the last read offset for this topic
	if currentOffset, exists := c.lastRead[message.Topic]; exists && message.Offset > currentOffset {
		c.lastRead[message.Topic] = message.Offset
		// Save to persistent storage
		if err := c.saveLastReadOffset(message.Topic, message.Offset); err != nil {
			return fmt.Errorf("failed to commit offset for topic %s: %w", message.Topic, err)
		}
		log.Printf("[MessageBus] Committed offset %d for topic %s", message.Offset, message.Topic)
	}

	return nil
}

// Close closes the local consumer
func (c *LocalConsumer) Close() error {
	c.mutex.Lock()
	if c.watcherCancel != nil {
		c.watcherCancel()
		<-c.watcherDone
		c.watcherCancel = nil
		c.watcherDone = make(chan struct{})
	}
	c.mutex.Unlock()
	return nil
}

// CleanupMessageBus removes all message files (useful for development)
func CleanupMessageBus() error {
	return os.RemoveAll(messageBusDir)
}

// CleanupOffsets removes all offset files (useful for development)
func CleanupOffsets() error {
	return os.RemoveAll(offsetDir)
}

// CleanupAll removes all message and offset files (useful for development)
func CleanupAll() error {
	if err := CleanupMessageBus(); err != nil {
		return fmt.Errorf("failed to cleanup message bus: %w", err)
	}
	if err := CleanupOffsets(); err != nil {
		return fmt.Errorf("failed to cleanup offsets: %w", err)
	}
	return nil
}

// GetMessageBusStats returns statistics about the message bus
func GetMessageBusStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Get message stats
	if _, err := os.Stat(messageBusDir); os.IsNotExist(err) {
		stats["status"] = "no_messages"
	} else {
		topics, err := os.ReadDir(messageBusDir)
		if err != nil {
			stats["error"] = err.Error()
		} else {
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
		}
	}

	// Get offset stats
	offsetStats := make(map[string]map[string]int64)
	if _, err := os.Stat(offsetDir); !os.IsNotExist(err) {
		consumerGroups, err := os.ReadDir(offsetDir)
		if err == nil {
			for _, group := range consumerGroups {
				if group.IsDir() {
					groupOffsets := make(map[string]int64)
					groupDir := filepath.Join(offsetDir, group.Name())
					offsetFiles, err := os.ReadDir(groupDir)
					if err == nil {
						for _, offsetFile := range offsetFiles {
							if !offsetFile.IsDir() && strings.HasSuffix(offsetFile.Name(), ".offset") {
								topic := strings.TrimSuffix(offsetFile.Name(), ".offset")
								offsetPath := filepath.Join(groupDir, offsetFile.Name())
								if data, err := os.ReadFile(offsetPath); err == nil {
									if offset, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
										groupOffsets[topic] = offset
									}
								}
							}
						}
					}
					if len(groupOffsets) > 0 {
						offsetStats[group.Name()] = groupOffsets
					}
				}
			}
		}
	}

	stats["message_bus_dir"] = messageBusDir
	stats["offset_dir"] = offsetDir
	if len(offsetStats) > 0 {
		stats["consumer_offsets"] = offsetStats
	}

	return stats
}
