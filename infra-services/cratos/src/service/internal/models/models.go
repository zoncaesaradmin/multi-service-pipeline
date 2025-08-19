package models

import (
	"time"
)

// MessageType defines the type of message being passed through the channels
type ChannelMessageType string

const (
	// ChannelMessageTypeData represents data messages that need processing
	ChannelMessageTypeData ChannelMessageType = "data"
	// ChannelMessageTypeControl represents control messages (start, stop, pause, etc.)
	ChannelMessageTypeControl ChannelMessageType = "control"
)

// ChannelMessage represents a common message structure for channel communication
type ChannelMessage struct {
	Type      ChannelMessageType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      []byte             `json:"data"`
}

// NewChannelMessage creates a new channel message with the given type and data
func NewChannelMessage(msgType ChannelMessageType, data []byte, source string) *ChannelMessage {
	return &ChannelMessage{
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// NewDataMessage creates a new data message
func NewDataMessage(data []byte, source string) *ChannelMessage {
	return NewChannelMessage(ChannelMessageTypeData, data, source)
}

// NewControlMessage creates a new control message
func NewControlMessage(data []byte, source string) *ChannelMessage {
	return NewChannelMessage(ChannelMessageTypeControl, data, source)
}

// IsDataMessage checks if the message is a data message
func (m *ChannelMessage) IsDataMessage() bool {
	return m.Type == ChannelMessageTypeData
}

// IsControlMessage checks if the message is a control message
func (m *ChannelMessage) IsControlMessage() bool {
	return m.Type == ChannelMessageTypeControl
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
