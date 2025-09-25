package metrics

import (
	"context"
	"fmt"
	"servicegomodule/internal/config"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sort"
	"sync"
	"time"
)

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTiming    MetricType = "timing"
)

// MetricEvent represents a single metric event
type MetricEvent struct {
	Type      MetricType             `json:"type"`
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Message   *models.ChannelMessage `json:"-"` // Reference to channel message for context
}

// MetricSummary represents aggregated metrics for a time window
type MetricSummary struct {
	Name       string            `json:"name"`
	Type       MetricType        `json:"type"`
	Count      int64             `json:"count"`
	Sum        float64           `json:"sum"`
	Min        float64           `json:"min"`
	Max        float64           `json:"max"`
	Avg        float64           `json:"avg"`
	Labels     map[string]string `json:"labels"`
	WindowSize time.Duration     `json:"windowSize"`
	LastUpdate time.Time         `json:"lastUpdate"`
}

// PipelineMetrics holds metrics for the entire pipeline
type PipelineMetrics struct {
	TotalMessages         int64                    `json:"totalMessages"`
	ProcessedMessages     int64                    `json:"processedMessages"`
	FailedMessages        int64                    `json:"failedMessages"`
	ErrorCount            int64                    `json:"errorCount"`
	RetryCount            int64                    `json:"retryCount"`
	AverageProcessingTime time.Duration            `json:"averageProcessingTime"`
	ThroughputPerSecond   float64                  `json:"msgCountThroughputPerSec"`
	BytesProcessed        int64                    `json:"bytesProcessed"`
	StageMetrics          map[string]*StageMetrics `json:"stageMetrics"`
	LastUpdated           time.Time                `json:"lastUpdated"`
}

// StageMetrics holds metrics for individual pipeline stages
type StageMetrics struct {
	StageName           string        `json:"stageName"`
	MessagesCompleted   int64         `json:"messagesCompleted"`
	MessagesFailed      int64         `json:"messagesFailed"`
	AverageLatency      time.Duration `json:"averageLatency"`
	MinLatency          time.Duration `json:"minLatency"`
	MaxLatency          time.Duration `json:"maxLatency"`
	TotalLatency        time.Duration `json:"totalLatency"`
	ThroughputPerSecond float64       `json:"countThroughputPerSec"`
	FirstProcessed      time.Time     `json:"firstProcessed"`
	LastProcessed       time.Time     `json:"lastProcessed"`
}

// MetricsCollector manages metrics collection and aggregation
type MetricsCollector struct {
	mu                sync.RWMutex
	logger            logging.Logger
	metricsChan       chan *MetricEvent
	ctx               context.Context
	cancel            context.CancelFunc
	retentionPeriod   time.Duration
	aggregationWindow time.Duration

	// Storage
	events          []*MetricEvent
	summaries       map[string]*MetricSummary
	pipelineMetrics *PipelineMetrics

	// Configuration
	maxEvents    int
	dumpInterval time.Duration
	started      bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger logging.Logger, config MetricsConfig) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	mc := &MetricsCollector{
		logger:            logger.WithField("component", "metrics"),
		metricsChan:       make(chan *MetricEvent, config.ChannelBufferSize),
		ctx:               ctx,
		cancel:            cancel,
		retentionPeriod:   config.RetentionPeriod,
		aggregationWindow: config.AggregationWindow,
		maxEvents:         config.MaxEvents,
		dumpInterval:      config.DumpInterval,
		events:            make([]*MetricEvent, 0),
		summaries:         make(map[string]*MetricSummary),
		pipelineMetrics: &PipelineMetrics{
			StageMetrics: make(map[string]*StageMetrics),
			LastUpdated:  time.Now(),
		},
	}

	return mc
}

// MetricsConfig holds configuration for the metrics collector
type MetricsConfig struct {
	ChannelBufferSize int           `json:"channelBufferSize"`
	RetentionPeriod   time.Duration `json:"retentionPeriod"`
	AggregationWindow time.Duration `json:"aggregationWindow"`
	MaxEvents         int           `json:"maxEvents"`
	DumpInterval      time.Duration `json:"dumpInterval"`
}

// DefaultMetricsConfig returns default configuration
func DefaultMetricsConfig(cfg *config.RawConfig) MetricsConfig {
	if cfg != nil {
		return MetricsConfig{
			ChannelBufferSize: 1000,
			RetentionPeriod:   time.Duration(cfg.Metrics.RetentionPeriod) * time.Minute,
			AggregationWindow: time.Duration(cfg.Metrics.AggregationWindow) * time.Minute,
			MaxEvents:         cfg.Metrics.MaxEvents,
			DumpInterval:      time.Duration(cfg.Metrics.DumpInterval) * time.Second,
		}
	} else {
		return MetricsConfig{
			ChannelBufferSize: 1000,
			RetentionPeriod:   10 * time.Minute,
			AggregationWindow: 1 * time.Minute,
			MaxEvents:         10000,
			DumpInterval:      30 * time.Second,
		}
	}
}

// Start begins the metrics collection process
func (mc *MetricsCollector) Start() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.started {
		return fmt.Errorf("metrics collector already started")
	}

	mc.started = true

	// Start metrics processing goroutine
	go mc.processMetrics()

	// Start periodic dump goroutine
	go mc.periodicDump()

	// Start cleanup goroutine
	go mc.periodicCleanup()

	mc.logger.Info("Metrics collector started")
	return nil
}

// Stop stops the metrics collection
func (mc *MetricsCollector) Stop() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.started {
		return nil
	}

	mc.started = false
	mc.cancel()
	close(mc.metricsChan)

	mc.logger.Info("Metrics collector stopped")
	return nil
}

// SendMetric sends a metric event to the collector
func (mc *MetricsCollector) SendMetric(event *MetricEvent) {
	if !mc.started {
		return
	}

	select {
	case mc.metricsChan <- event:
		// Event sent successfully
	default:
		// Channel is full, log warning
		mc.logger.Warn("Metrics channel is full, dropping metric event")
	}
}

// processMetrics processes incoming metric events
func (mc *MetricsCollector) processMetrics() {
	for {
		select {
		case <-mc.ctx.Done():
			mc.logger.Info("Metrics processing stopped")
			return
		case event := <-mc.metricsChan:
			if event != nil {
				mc.processMetricEvent(event)
			}
		}
	}
}

// processMetricEvent processes a single metric event
func (mc *MetricsCollector) processMetricEvent(event *MetricEvent) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Add to events list
	mc.events = append(mc.events, event)

	// Limit events size
	if len(mc.events) > mc.maxEvents {
		mc.events = mc.events[len(mc.events)-mc.maxEvents:]
	}

	// Update summaries
	mc.updateSummary(event)

	// Update pipeline metrics
	mc.updatePipelineMetrics(event)
}

// updateSummary updates metric summaries
func (mc *MetricsCollector) updateSummary(event *MetricEvent) {
	key := mc.getSummaryKey(event)

	summary, exists := mc.summaries[key]
	if !exists {
		summary = &MetricSummary{
			Name:       event.Name,
			Type:       event.Type,
			Labels:     event.Labels,
			Min:        event.Value,
			Max:        event.Value,
			WindowSize: mc.aggregationWindow,
		}
		mc.summaries[key] = summary
	}

	summary.Count++
	summary.Sum += event.Value
	summary.Avg = summary.Sum / float64(summary.Count)
	summary.LastUpdate = event.Timestamp

	if event.Value < summary.Min {
		summary.Min = event.Value
	}
	if event.Value > summary.Max {
		summary.Max = event.Value
	}
}

// updatePipelineMetrics updates overall pipeline metrics
func (mc *MetricsCollector) updatePipelineMetrics(event *MetricEvent) {
	switch event.Name {
	case "message.received":
		mc.pipelineMetrics.TotalMessages++
	case "message.processed":
		mc.pipelineMetrics.ProcessedMessages++
		if event.Message != nil {
			mc.pipelineMetrics.BytesProcessed += event.Message.Size
			duration := event.Message.GetTotalProcessingTime()
			mc.updateAverageProcessingTime(duration)
		}
	case "message.failed":
		mc.pipelineMetrics.FailedMessages++
	case "error.occurred":
		mc.pipelineMetrics.ErrorCount++
	case "retry.attempted":
		mc.pipelineMetrics.RetryCount++
	}

	// Update stage metrics
	if stage, ok := event.Labels["stage"]; ok {
		mc.updateStageMetrics(stage, event)
	}

	mc.pipelineMetrics.LastUpdated = time.Now()
	mc.calculateThroughput()
}

// updateStageMetrics updates metrics for a specific pipeline stage
func (mc *MetricsCollector) updateStageMetrics(stageName string, event *MetricEvent) {
	stageMetrics, exists := mc.pipelineMetrics.StageMetrics[stageName]
	if !exists {
		stageMetrics = &StageMetrics{
			StageName:  stageName,
			MinLatency: 0, // Initialize to 0, will be set on first measurement
		}
		mc.pipelineMetrics.StageMetrics[stageName] = stageMetrics
	}

	switch event.Name {
	case "stage.completed":
		stageMetrics.MessagesCompleted++
		stageMetrics.LastProcessed = event.Timestamp

		// Set FirstProcessed timestamp on first message
		if stageMetrics.FirstProcessed.IsZero() {
			stageMetrics.FirstProcessed = event.Timestamp
		}

		if event.Duration > 0 {
			stageMetrics.TotalLatency += event.Duration
			stageMetrics.AverageLatency = time.Duration(int64(stageMetrics.TotalLatency) / stageMetrics.MessagesCompleted)

			// Initialize MinLatency on first measurement
			if stageMetrics.MinLatency == 0 || event.Duration < stageMetrics.MinLatency {
				stageMetrics.MinLatency = event.Duration
			}
			if event.Duration > stageMetrics.MaxLatency {
				stageMetrics.MaxLatency = event.Duration
			}
		}
	case "stage.failed":
		stageMetrics.MessagesFailed++
	}
}

// updateAverageProcessingTime updates the average processing time
func (mc *MetricsCollector) updateAverageProcessingTime(duration time.Duration) {
	if mc.pipelineMetrics.ProcessedMessages > 0 {
		currentTotal := mc.pipelineMetrics.AverageProcessingTime * time.Duration(mc.pipelineMetrics.ProcessedMessages-1)
		mc.pipelineMetrics.AverageProcessingTime = (currentTotal + duration) / time.Duration(mc.pipelineMetrics.ProcessedMessages)
	} else {
		mc.pipelineMetrics.AverageProcessingTime = duration
	}
}

// calculateThroughput calculates throughput per second
func (mc *MetricsCollector) calculateThroughput() {
	now := time.Now()
	elapsed := now.Sub(mc.pipelineMetrics.LastUpdated)
	if elapsed > 0 && mc.pipelineMetrics.ProcessedMessages > 0 {
		// Calculate throughput based on total runtime
		totalElapsed := now.Sub(time.Now().Add(-mc.aggregationWindow))
		if totalElapsed > 0 {
			mc.pipelineMetrics.ThroughputPerSecond = float64(mc.pipelineMetrics.ProcessedMessages) / totalElapsed.Seconds()
		}
	}

	// Calculate stage-specific throughput
	for _, stageMetrics := range mc.pipelineMetrics.StageMetrics {
		if !stageMetrics.FirstProcessed.IsZero() && stageMetrics.MessagesCompleted > 0 {
			// Calculate throughput based on total elapsed time from first message to now
			totalElapsed := now.Sub(stageMetrics.FirstProcessed)
			if totalElapsed > 0 {
				stageMetrics.ThroughputPerSecond = float64(stageMetrics.MessagesCompleted) / totalElapsed.Seconds()
			}
		}
	}
}

// getSummaryKey generates a unique key for metric summaries
func (mc *MetricsCollector) getSummaryKey(event *MetricEvent) string {
	key := fmt.Sprintf("%s:%s", event.Type, event.Name)

	// Sort labels to ensure deterministic key generation
	var keys []string
	for k := range event.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		key += fmt.Sprintf(":%s=%s", k, event.Labels[k])
	}
	return key
}

// periodicDump periodically dumps metrics to logs
func (mc *MetricsCollector) periodicDump() {
	ticker := time.NewTicker(mc.dumpInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			mc.logger.Info("Periodic dump stopped")
			return
		case <-ticker.C:
			mc.dumpMetrics()
		}
	}
}

// periodicCleanup periodically cleans up old metrics
func (mc *MetricsCollector) periodicCleanup() {
	ticker := time.NewTicker(mc.retentionPeriod / 2)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			mc.logger.Info("Periodic cleanup stopped")
			return
		case <-ticker.C:
			mc.cleanupOldMetrics()
		}
	}
}

// dumpMetrics dumps current metrics to logs
func (mc *MetricsCollector) dumpMetrics() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	mc.logger.WithFields(map[string]interface{}{
		"totalMessages":            mc.pipelineMetrics.TotalMessages,
		"processedMessages":        mc.pipelineMetrics.ProcessedMessages,
		"failedMessages":           mc.pipelineMetrics.FailedMessages,
		"errorCount":               mc.pipelineMetrics.ErrorCount,
		"retryCount":               mc.pipelineMetrics.RetryCount,
		"averageProcessingTime":    mc.pipelineMetrics.AverageProcessingTime.String(),
		"msgCountThroughputPerSec": mc.pipelineMetrics.ThroughputPerSecond,
		"bytesProcessed":           mc.pipelineMetrics.BytesProcessed,
		//"summaryCount":             len(mc.summaries),
		//"eventCount":               len(mc.events),
	}).Info("Pipeline metrics summary")

	// Log stage metrics
	for stageName, stageMetrics := range mc.pipelineMetrics.StageMetrics {
		// Format latencies for display
		minLatencyStr := "N/A"
		if stageMetrics.MinLatency > 0 {
			minLatencyStr = stageMetrics.MinLatency.String()
		}

		maxLatencyStr := "N/A"
		if stageMetrics.MaxLatency > 0 {
			maxLatencyStr = stageMetrics.MaxLatency.String()
		}

		avgLatencyStr := "N/A"
		if stageMetrics.AverageLatency > 0 {
			avgLatencyStr = stageMetrics.AverageLatency.String()
		}

		mc.logger.WithFields(map[string]interface{}{
			"stage":                 stageName,
			"messagesCompleted":     stageMetrics.MessagesCompleted,
			"messagesFailed":        stageMetrics.MessagesFailed,
			"averageLatency":        avgLatencyStr,
			"minLatency":            minLatencyStr,
			"maxLatency":            maxLatencyStr,
			"countThroughputPerSec": stageMetrics.ThroughputPerSecond,
		}).Info("Stage metrics")
	}

	// Collect counter metrics for consolidated logging
	counterFields := map[string]interface{}{}
	for _, summary := range mc.summaries {
		if summary.Type == MetricTypeCounter {
			// Create a key that includes stage info if available
			var counterKey string
			if stage, hasStage := summary.Labels["stage"]; hasStage {
				counterKey = fmt.Sprintf("%s.%s", stage, summary.Name)
			} else {
				counterKey = summary.Name
			}
			counterFields[counterKey] = summary.Sum
		}
	}

	// Log all counters in a single log entry if any exist
	if len(counterFields) > 0 {
		mc.logger.WithFields(counterFields).Info("Counter metrics")
	}
}

// cleanupOldMetrics removes old metrics beyond retention period
func (mc *MetricsCollector) cleanupOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.retentionPeriod)

	// Clean up events
	filteredEvents := make([]*MetricEvent, 0)
	for _, event := range mc.events {
		if event.Timestamp.After(cutoff) {
			filteredEvents = append(filteredEvents, event)
		}
	}
	removed := len(mc.events) - len(filteredEvents)
	mc.events = filteredEvents

	// Clean up summaries
	for key, summary := range mc.summaries {
		if summary.LastUpdate.Before(cutoff) {
			delete(mc.summaries, key)
		}
	}

	if removed > 0 {
		mc.logger.Infof("Cleaned up %d old metric events", removed)
	}
}

// GetMetrics returns current metrics data
func (mc *MetricsCollector) GetMetrics() *PipelineMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *mc.pipelineMetrics
	metrics.StageMetrics = make(map[string]*StageMetrics)
	for k, v := range mc.pipelineMetrics.StageMetrics {
		stageCopy := *v
		metrics.StageMetrics[k] = &stageCopy
	}

	return &metrics
}

// GetSummaries returns current metric summaries
func (mc *MetricsCollector) GetSummaries() map[string]*MetricSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	summaries := make(map[string]*MetricSummary)
	for k, v := range mc.summaries {
		summaryCopy := *v
		summaries[k] = &summaryCopy
	}

	return summaries
}

// GetRecentEvents returns recent metric events
func (mc *MetricsCollector) GetRecentEvents(limit int) []*MetricEvent {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit <= 0 || limit > len(mc.events) {
		limit = len(mc.events)
	}

	start := len(mc.events) - limit
	events := make([]*MetricEvent, limit)
	copy(events, mc.events[start:])

	return events
}
