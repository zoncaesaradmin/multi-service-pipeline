package messagebus

import (
	"context"
	"time"
)

// Message represents a message in the message bus
type Message struct {
	Topic     string            `json:"topic"`
	Key       string            `json:"key,omitempty"`
	Value     []byte            `json:"value"`
	Headers   map[string]string `json:"headers,omitempty"`
	Partition int32             `json:"partition,omitempty"`
	Offset    int64             `json:"offset,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// Producer interface for publishing messages
type Producer interface {
	// Send sends a message synchronously and returns the partition and offset
	Send(ctx context.Context, message *Message) (partition int32, offset int64, err error)

	// SendAsync sends a message asynchronously and returns a channel for the result
	// The channel will receive a SendResult when the operation completes
	SendAsync(ctx context.Context, message *Message) <-chan SendResult

	// Close closes the producer
	Close() error
}

// SendResult represents the result of an asynchronous send operation
type SendResult struct {
	Partition int32 // The partition the message was sent to
	Offset    int64 // The offset of the message
	Error     error // Any error that occurred during sending
}

// Consumer interface for consuming messages
type Consumer interface {
	// Subscribe subscribes to topics
	Subscribe(topics []string) error

	// OnMessage sets a callback for incoming messages
	OnMessage(fn func(*Message))

	// OnAssign sets a callback for partition assignment events
	OnAssign(fn func([]PartitionAssignment))

	// OnRevoke sets a callback for partition revocation events
	OnRevoke(fn func([]PartitionAssignment))

	// Commit manually commits the offset for a message
	Commit(ctx context.Context, message *Message) error

	// AssignedPartitions returns the currently assigned partitions
	AssignedPartitions() []PartitionAssignment

	// Close closes the consumer
	Close() error
}

// PartitionAssignment is an agnostic representation of a partition assignment
type PartitionAssignment struct {
	Topic     string
	Partition int32
}
