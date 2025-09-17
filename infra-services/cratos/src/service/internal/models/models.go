package models

import (
	"context"
	"time"
)

// MessageType defines the type of message being passed through the channels
type ChannelMessageType string

// MessageOrigin defines the type of message origin
type ChannelMessageOrigin string

const (
	// ChannelMessageTypeData represents data messages that need processing
	ChannelMessageTypeData ChannelMessageType = "data"
	// ChannelMessageTypeControl represents control messages (start, stop, pause, etc.)
	ChannelMessageTypeControl ChannelMessageType = "control"
)

const (
	ChannelMessageOriginKafka ChannelMessageOrigin = "kafka"
	ChannelMessageOriginDb    ChannelMessageOrigin = "database"
)

// CommitCallback represents a function that commits the offset for a message
type CommitCallback func(ctx context.Context) error

// ChannelMessage represents a common message structure for channel communication
type ChannelMessage struct {
	Origin           ChannelMessageOrigin `json:"origin"` // e.g., "kafka" (from source service), DB read
	Type             ChannelMessageType   `json:"type"`
	Timestamp        time.Time            `json:"timestamp"`
	EntryTimestamp   time.Time            `json:"entryTimestamp"`   // When message entered the service
	ProcessStartTime time.Time            `json:"processStartTime"` // When processing started
	ProcessEndTime   time.Time            `json:"processEndTime"`   // When processing ended
	OutputTimestamp  time.Time            `json:"outputTimestamp"`  // When message is out of output
	Data             []byte               `json:"data"`
	Meta             map[string]string    `json:"meta"`
	Key              string               `json:"key"`
	Partition        int32                `json:"partition"`
	Size             int64                `json:"size"`            // Size of the message in bytes
	ProcessingStage  string               `json:"processingStage"` // Current processing stage
	ErrorCount       int                  `json:"errorCount"`      // Number of errors encountered
	RetryCount       int                  `json:"retryCount"`      // Number of retries
	CommitCallback   CommitCallback       `json:"-"`               // Not serialized, used for offset management
	Context          context.Context      `json:"-"`               // Not serialized, used for trace propagation
}

// NewChannelMessage creates a new channel message with the given type and data
func NewChannelMessage(msgType ChannelMessageType, data []byte, source string, partition int32) *ChannelMessage {
	now := time.Now()
	return &ChannelMessage{
		Type:            msgType,
		Timestamp:       now,
		EntryTimestamp:  now,
		Data:            data,
		Meta:            make(map[string]string),
		Key:             source,
		Partition:       partition,
		Size:            int64(len(data)),
		ProcessingStage: "created",
		ErrorCount:      0,
		RetryCount:      0,
	}
}

// NewDataMessage creates a new data message
func NewDataMessage(data []byte, source string, partition int32) *ChannelMessage {
	return NewChannelMessage(ChannelMessageTypeData, data, source, partition)
}

// NewControlMessage creates a new control message
func NewControlMessage(data []byte, source string) *ChannelMessage {
	return NewChannelMessage(ChannelMessageTypeControl, data, source, 0)
}

// IsDataMessage checks if the message is a data message
func (m *ChannelMessage) IsDataMessage() bool {
	return m.Type == ChannelMessageTypeData
}

// IsControlMessage checks if the message is a control message
func (m *ChannelMessage) IsControlMessage() bool {
	return m.Type == ChannelMessageTypeControl
}

// Commit commits the offset for this message if a commit callback is available
func (m *ChannelMessage) Commit(ctx context.Context) error {
	if m.CommitCallback != nil {
		return m.CommitCallback(ctx)
	}
	return nil
}

// HasCommitCallback checks if this message has a commit callback
func (m *ChannelMessage) HasCommitCallback() bool {
	return m.CommitCallback != nil
}

// GetTotalProcessingTime returns the total time spent in the service
func (m *ChannelMessage) GetTotalProcessingTime() time.Duration {
	if m.OutputTimestamp.IsZero() {
		return time.Since(m.EntryTimestamp)
	}
	return m.OutputTimestamp.Sub(m.EntryTimestamp)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}
