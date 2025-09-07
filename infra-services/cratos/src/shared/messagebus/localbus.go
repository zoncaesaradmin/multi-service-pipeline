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
	messageBusDir = "/tmp/cratos-messagebus-test"
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
func NewProducer(configMap map[string]any, clientId string) Producer {
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
}

// NewConsumer creates a new local consumer with configuration from YAML file
// The cgroup parameter is ignored for local implementation as it's single consumer
func NewConsumer(configMap map[string]any, cgroup string) Consumer {
	// Create local consumer with configuration
	consumer := &LocalConsumer{
		lastRead:    make(map[string]int64),
		assigned:    []PartitionAssignment{},
		watcherDone: make(chan struct{}),
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
		if _, exists := c.lastRead[topic]; !exists {
			c.lastRead[topic] = -1
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
			c.lastRead[topic] = -1
		}
		lastOffset := c.lastRead[topic]
		nextOffset := lastOffset + 1
		filename := filepath.Join(topicDir, fmt.Sprintf("%010d.json", nextOffset))
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			continue
		}
		messageData, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		var message Message
		if err := json.Unmarshal(messageData, &message); err != nil {
			continue
		}
		c.lastRead[topic] = nextOffset
		if c.onMessage != nil {
			go c.onMessage(&message)
		}
	}
}

// Commit commits the offset (no-op for local)
func (c *LocalConsumer) Commit(ctx context.Context, message *Message) error {
	// Local implementation doesn't need actual commit
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
