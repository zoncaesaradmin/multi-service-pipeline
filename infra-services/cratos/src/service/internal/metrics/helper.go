package metrics

import (
	"servicegomodule/internal/models"
	"time"
)

// MetricsHelper provides convenient methods for sending metrics
type MetricsHelper struct {
	collector *MetricsCollector
	stage     string
}

// NewMetricsHelper creates a new metrics helper for a specific stage
func NewMetricsHelper(collector *MetricsCollector, stage string) *MetricsHelper {
	return &MetricsHelper{
		collector: collector,
		stage:     stage,
	}
}

// RecordMessageReceived records when a message is received
func (mh *MetricsHelper) RecordMessageReceived(msg *models.ChannelMessage) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "message.received",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage, "origin": string(msg.Origin)},
		Timestamp: time.Now(),
		Message:   msg,
	}
	mh.collector.SendMetric(event)
}

// RecordMessageCompleted records when a message is successfully processed
func (mh *MetricsHelper) RecordMessageCompleted(msg *models.ChannelMessage) {
	duration := msg.GetTotalProcessingTime()

	// Record processing completion
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "message.processed",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage},
		Timestamp: time.Now(),
		Duration:  duration,
		Message:   msg,
	}
	mh.collector.SendMetric(event)

	// Record processing time
	timingEvent := &MetricEvent{
		Type:      MetricTypeTiming,
		Name:      "processing.duration",
		Value:     float64(duration.Milliseconds()),
		Labels:    map[string]string{"stage": mh.stage},
		Timestamp: time.Now(),
		Duration:  duration,
		Message:   msg,
	}
	mh.collector.SendMetric(timingEvent)
}

// RecordMessageFailed records when message processing fails
func (mh *MetricsHelper) RecordMessageFailed(msg *models.ChannelMessage, reason string) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "message.failed",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage, "reason": reason},
		Timestamp: time.Now(),
		Message:   msg,
	}
	mh.collector.SendMetric(event)
}

// RecordError records when an error occurs
func (mh *MetricsHelper) RecordError(msg *models.ChannelMessage, errorType string) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "error.occurred",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage, "type": errorType},
		Timestamp: time.Now(),
		Message:   msg,
	}
	mh.collector.SendMetric(event)
}

// RecordRetry records when a retry is attempted
func (mh *MetricsHelper) RecordRetry(msg *models.ChannelMessage, attempt int) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "retry.attempted",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage, "attempt": string(rune(attempt))},
		Timestamp: time.Now(),
		Message:   msg,
	}
	mh.collector.SendMetric(event)
}

// RecordStageLatency records latency for a specific stage
func (mh *MetricsHelper) RecordStageLatency(duration time.Duration, operation string) {
	event := &MetricEvent{
		Type:      MetricTypeTiming,
		Name:      "stage.latency",
		Value:     float64(duration.Milliseconds()),
		Labels:    map[string]string{"stage": mh.stage, "operation": operation},
		Timestamp: time.Now(),
		Duration:  duration,
	}
	mh.collector.SendMetric(event)
}

// RecordStageCompleted records when a stage completes processing
func (mh *MetricsHelper) RecordStageCompleted(msg *models.ChannelMessage, duration time.Duration) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "stage.completed",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage},
		Timestamp: time.Now(),
		Duration:  duration,
		Message:   msg,
	}
	mh.collector.SendMetric(event)
}

// RecordStageFailed records when a stage fails
func (mh *MetricsHelper) RecordStageFailed(msg *models.ChannelMessage, reason string) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "stage.failed",
		Value:     1,
		Labels:    map[string]string{"stage": mh.stage, "reason": reason},
		Timestamp: time.Now(),
		Message:   msg,
	}
	mh.collector.SendMetric(event)
}

// RecordGauge records a gauge metric
func (mh *MetricsHelper) RecordGauge(name string, value float64, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["stage"] = mh.stage

	event := &MetricEvent{
		Type:      MetricTypeGauge,
		Name:      name,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
	mh.collector.SendMetric(event)
}

// RecordCounter records a counter metric
func (mh *MetricsHelper) RecordCounter(name string, value float64, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["stage"] = mh.stage

	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      name,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
	mh.collector.SendMetric(event)
}

// RecordHistogram records a histogram metric
func (mh *MetricsHelper) RecordHistogram(name string, value float64, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["stage"] = mh.stage

	event := &MetricEvent{
		Type:      MetricTypeHistogram,
		Name:      name,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
	mh.collector.SendMetric(event)
}

// GetCollector returns the underlying metrics collector
func (mh *MetricsHelper) GetCollector() *MetricsCollector {
	return mh.collector
}
