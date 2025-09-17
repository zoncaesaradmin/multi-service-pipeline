package metrics

import (
	"context"
	"encoding/json"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sync"
	"testing"
	"time"
)

// mockLogger implements the logging.Logger interface for testing
type mockLogger struct{}

func (m *mockLogger) SetLevel(level logging.Level)                                       {}
func (m *mockLogger) GetLevel() logging.Level                                            { return logging.InfoLevel }
func (m *mockLogger) IsLevelEnabled(level logging.Level) bool                            { return true }
func (m *mockLogger) Debug(msg string)                                                   {}
func (m *mockLogger) Info(msg string)                                                    {}
func (m *mockLogger) Warn(msg string)                                                    {}
func (m *mockLogger) Error(msg string)                                                   {}
func (m *mockLogger) Fatal(msg string)                                                   {}
func (m *mockLogger) Panic(msg string)                                                   {}
func (m *mockLogger) Debugf(format string, args ...interface{})                          {}
func (m *mockLogger) Infof(format string, args ...interface{})                           {}
func (m *mockLogger) Warnf(format string, args ...interface{})                           {}
func (m *mockLogger) Errorf(format string, args ...interface{})                          {}
func (m *mockLogger) Fatalf(format string, args ...interface{})                          {}
func (m *mockLogger) Panicf(format string, args ...interface{})                          {}
func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{})                    {}
func (m *mockLogger) Infow(msg string, keysAndValues ...interface{})                     {}
func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{})                     {}
func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{})                    {}
func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{})                    {}
func (m *mockLogger) Panicw(msg string, keysAndValues ...interface{})                    {}
func (m *mockLogger) WithFields(fields logging.Fields) logging.Logger                    { return m }
func (m *mockLogger) WithField(key string, value interface{}) logging.Logger             { return m }
func (m *mockLogger) WithError(err error) logging.Logger                                 { return m }
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger                     { return m }
func (m *mockLogger) Log(level logging.Level, msg string)                                {}
func (m *mockLogger) Logf(level logging.Level, format string, args ...interface{})       {}
func (m *mockLogger) Logw(level logging.Level, msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Clone() logging.Logger                                              { return m }
func (m *mockLogger) Close() error                                                       { return nil }

func TestMetricType_Constants(t *testing.T) {
	tests := []struct {
		name       string
		metricType MetricType
		expected   string
	}{
		{"Counter", MetricTypeCounter, "counter"},
		{"Gauge", MetricTypeGauge, "gauge"},
		{"Histogram", MetricTypeHistogram, "histogram"},
		{"Timing", MetricTypeTiming, "timing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.metricType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.metricType))
			}
		})
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	expectedChannelBufferSize := 1000
	expectedRetentionPeriod := 10 * time.Minute
	expectedAggregationWindow := 1 * time.Minute
	expectedMaxEvents := 10000
	expectedDumpInterval := 30 * time.Second

	if config.ChannelBufferSize != expectedChannelBufferSize {
		t.Errorf("Expected ChannelBufferSize %d, got %d", expectedChannelBufferSize, config.ChannelBufferSize)
	}
	if config.RetentionPeriod != expectedRetentionPeriod {
		t.Errorf("Expected RetentionPeriod %v, got %v", expectedRetentionPeriod, config.RetentionPeriod)
	}
	if config.AggregationWindow != expectedAggregationWindow {
		t.Errorf("Expected AggregationWindow %v, got %v", expectedAggregationWindow, config.AggregationWindow)
	}
	if config.MaxEvents != expectedMaxEvents {
		t.Errorf("Expected MaxEvents %d, got %d", expectedMaxEvents, config.MaxEvents)
	}
	if config.DumpInterval != expectedDumpInterval {
		t.Errorf("Expected DumpInterval %v, got %v", expectedDumpInterval, config.DumpInterval)
	}
}

func TestNewMetricsCollector(t *testing.T) {
	logger := &mockLogger{}
	config := MetricsConfig{
		ChannelBufferSize: 100,
		RetentionPeriod:   5 * time.Minute,
		AggregationWindow: 30 * time.Second,
		MaxEvents:         1000,
		DumpInterval:      10 * time.Second,
	}

	mc := NewMetricsCollector(logger, config)

	if mc == nil {
		t.Fatal("Expected non-nil MetricsCollector")
	}
	if mc.retentionPeriod != config.RetentionPeriod {
		t.Errorf("Expected RetentionPeriod %v, got %v", config.RetentionPeriod, mc.retentionPeriod)
	}
	if mc.aggregationWindow != config.AggregationWindow {
		t.Errorf("Expected AggregationWindow %v, got %v", config.AggregationWindow, mc.aggregationWindow)
	}
	if mc.maxEvents != config.MaxEvents {
		t.Errorf("Expected MaxEvents %d, got %d", config.MaxEvents, mc.maxEvents)
	}
	if mc.dumpInterval != config.DumpInterval {
		t.Errorf("Expected DumpInterval %v, got %v", config.DumpInterval, mc.dumpInterval)
	}
	if mc.started {
		t.Error("Expected collector to not be started initially")
	}
	if mc.pipelineMetrics == nil {
		t.Error("Expected pipelineMetrics to be initialized")
	}
	if mc.pipelineMetrics.StageMetrics == nil {
		t.Error("Expected StageMetrics map to be initialized")
	}
}

func TestMetricsCollector_StartStop(t *testing.T) {
	logger := &mockLogger{}
	config := DefaultMetricsConfig()
	mc := NewMetricsCollector(logger, config)

	// Test Start
	err := mc.Start()
	if err != nil {
		t.Fatalf("Expected no error starting collector, got: %v", err)
	}
	if !mc.started {
		t.Error("Expected collector to be started")
	}

	// Test double start
	err = mc.Start()
	if err == nil {
		t.Error("Expected error when starting already started collector")
	}

	// Test Stop
	err = mc.Stop()
	if err != nil {
		t.Fatalf("Expected no error stopping collector, got: %v", err)
	}
	if mc.started {
		t.Error("Expected collector to be stopped")
	}

	// Test double stop
	err = mc.Stop()
	if err != nil {
		t.Errorf("Expected no error when stopping already stopped collector, got: %v", err)
	}
}

func TestMetricsCollector_SendMetric(t *testing.T) {
	logger := &mockLogger{}
	config := MetricsConfig{
		ChannelBufferSize: 2, // Small buffer for testing
		RetentionPeriod:   1 * time.Minute,
		AggregationWindow: 10 * time.Second,
		MaxEvents:         100,
		DumpInterval:      1 * time.Second,
	}
	mc := NewMetricsCollector(logger, config)

	// Test sending metric before start (should not block)
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "test.metric",
		Value:     1.0,
		Labels:    map[string]string{"stage": "test"},
		Timestamp: time.Now(),
	}
	mc.SendMetric(event)

	// Start collector and send metric
	err := mc.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer mc.Stop()

	// Wait a bit for goroutines to start
	time.Sleep(10 * time.Millisecond)

	mc.SendMetric(event)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check that metric was processed
	mc.mu.RLock()
	if len(mc.events) == 0 {
		t.Error("Expected at least one event to be processed")
	}
	if len(mc.summaries) == 0 {
		t.Error("Expected at least one summary to be created")
	}
	mc.mu.RUnlock()
}

func TestMetricsCollector_UpdateSummary(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	event1 := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "test.counter",
		Value:     5.0,
		Labels:    map[string]string{"stage": "test"},
		Timestamp: time.Now(),
	}

	event2 := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "test.counter",
		Value:     10.0,
		Labels:    map[string]string{"stage": "test"},
		Timestamp: time.Now(),
	}

	mc.updateSummary(event1)
	mc.updateSummary(event2)

	key := mc.getSummaryKey(event1)
	summary, exists := mc.summaries[key]
	if !exists {
		t.Fatal("Expected summary to exist")
	}

	if summary.Count != 2 {
		t.Errorf("Expected count 2, got %d", summary.Count)
	}
	if summary.Sum != 15.0 {
		t.Errorf("Expected sum 15.0, got %f", summary.Sum)
	}
	if summary.Avg != 7.5 {
		t.Errorf("Expected avg 7.5, got %f", summary.Avg)
	}
	if summary.Min != 5.0 {
		t.Errorf("Expected min 5.0, got %f", summary.Min)
	}
	if summary.Max != 10.0 {
		t.Errorf("Expected max 10.0, got %f", summary.Max)
	}
}

func TestMetricsCollector_UpdatePipelineMetrics(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	msg.Size = 100

	tests := []struct {
		name    string
		event   *MetricEvent
		checkFn func(t *testing.T, pm *PipelineMetrics)
	}{
		{
			name: "message.received",
			event: &MetricEvent{
				Type:      MetricTypeCounter,
				Name:      "message.received",
				Value:     1.0,
				Labels:    map[string]string{"stage": "input"},
				Timestamp: time.Now(),
				Message:   msg,
			},
			checkFn: func(t *testing.T, pm *PipelineMetrics) {
				if pm.TotalMessages != 1 {
					t.Errorf("Expected TotalMessages 1, got %d", pm.TotalMessages)
				}
			},
		},
		{
			name: "message.processed",
			event: &MetricEvent{
				Type:      MetricTypeCounter,
				Name:      "message.processed",
				Value:     1.0,
				Labels:    map[string]string{"stage": "processor"},
				Timestamp: time.Now(),
				Message:   msg,
			},
			checkFn: func(t *testing.T, pm *PipelineMetrics) {
				if pm.ProcessedMessages != 1 {
					t.Errorf("Expected ProcessedMessages 1, got %d", pm.ProcessedMessages)
				}
				if pm.BytesProcessed != 100 {
					t.Errorf("Expected BytesProcessed 100, got %d", pm.BytesProcessed)
				}
			},
		},
		{
			name: "message.failed",
			event: &MetricEvent{
				Type:      MetricTypeCounter,
				Name:      "message.failed",
				Value:     1.0,
				Labels:    map[string]string{"stage": "processor"},
				Timestamp: time.Now(),
			},
			checkFn: func(t *testing.T, pm *PipelineMetrics) {
				if pm.FailedMessages != 1 {
					t.Errorf("Expected FailedMessages 1, got %d", pm.FailedMessages)
				}
			},
		},
		{
			name: "error.occurred",
			event: &MetricEvent{
				Type:      MetricTypeCounter,
				Name:      "error.occurred",
				Value:     1.0,
				Labels:    map[string]string{"stage": "processor"},
				Timestamp: time.Now(),
			},
			checkFn: func(t *testing.T, pm *PipelineMetrics) {
				if pm.ErrorCount != 1 {
					t.Errorf("Expected ErrorCount 1, got %d", pm.ErrorCount)
				}
			},
		},
		{
			name: "retry.attempted",
			event: &MetricEvent{
				Type:      MetricTypeCounter,
				Name:      "retry.attempted",
				Value:     1.0,
				Labels:    map[string]string{"stage": "processor"},
				Timestamp: time.Now(),
			},
			checkFn: func(t *testing.T, pm *PipelineMetrics) {
				if pm.RetryCount != 1 {
					t.Errorf("Expected RetryCount 1, got %d", pm.RetryCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metrics
			mc.pipelineMetrics = &PipelineMetrics{
				StageMetrics: make(map[string]*StageMetrics),
				LastUpdated:  time.Now(),
			}

			mc.updatePipelineMetrics(tt.event)
			tt.checkFn(t, mc.pipelineMetrics)
		})
	}
}

func TestMetricsCollector_UpdateStageMetrics(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	stageName := "test-stage"
	now := time.Now()

	// Test stage.completed event
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "stage.completed",
		Value:     1.0,
		Labels:    map[string]string{"stage": stageName},
		Timestamp: now,
		Duration:  100 * time.Millisecond,
	}

	mc.updateStageMetrics(stageName, event)

	stageMetrics, exists := mc.pipelineMetrics.StageMetrics[stageName]
	if !exists {
		t.Fatal("Expected stage metrics to exist")
	}

	if stageMetrics.StageName != stageName {
		t.Errorf("Expected StageName %s, got %s", stageName, stageMetrics.StageName)
	}
	if stageMetrics.MessagesCompleted != 1 {
		t.Errorf("Expected MessagesCompleted 1, got %d", stageMetrics.MessagesCompleted)
	}
	if stageMetrics.FirstProcessed.IsZero() {
		t.Error("Expected FirstProcessed to be set")
	}
	if stageMetrics.LastProcessed != now {
		t.Errorf("Expected LastProcessed %v, got %v", now, stageMetrics.LastProcessed)
	}
	if stageMetrics.TotalLatency != 100*time.Millisecond {
		t.Errorf("Expected TotalLatency 100ms, got %v", stageMetrics.TotalLatency)
	}
	if stageMetrics.AverageLatency != 100*time.Millisecond {
		t.Errorf("Expected AverageLatency 100ms, got %v", stageMetrics.AverageLatency)
	}
	if stageMetrics.MinLatency != 100*time.Millisecond {
		t.Errorf("Expected MinLatency 100ms, got %v", stageMetrics.MinLatency)
	}
	if stageMetrics.MaxLatency != 100*time.Millisecond {
		t.Errorf("Expected MaxLatency 100ms, got %v", stageMetrics.MaxLatency)
	}

	// Test stage.failed event
	failedEvent := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "stage.failed",
		Value:     1.0,
		Labels:    map[string]string{"stage": stageName},
		Timestamp: now,
	}

	mc.updateStageMetrics(stageName, failedEvent)

	if stageMetrics.MessagesFailed != 1 {
		t.Errorf("Expected MessagesFailed 1, got %d", stageMetrics.MessagesFailed)
	}
}

func TestMetricsCollector_UpdateAverageProcessingTime(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	// First message
	mc.pipelineMetrics.ProcessedMessages = 1
	mc.updateAverageProcessingTime(100 * time.Millisecond)

	if mc.pipelineMetrics.AverageProcessingTime != 100*time.Millisecond {
		t.Errorf("Expected AverageProcessingTime 100ms, got %v", mc.pipelineMetrics.AverageProcessingTime)
	}

	// Second message
	mc.pipelineMetrics.ProcessedMessages = 2
	mc.updateAverageProcessingTime(200 * time.Millisecond)

	expected := 150 * time.Millisecond // (100 + 200) / 2
	if mc.pipelineMetrics.AverageProcessingTime != expected {
		t.Errorf("Expected AverageProcessingTime %v, got %v", expected, mc.pipelineMetrics.AverageProcessingTime)
	}
}

func TestMetricsCollector_GetSummaryKey(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	event := &MetricEvent{
		Type:   MetricTypeCounter,
		Name:   "test.metric",
		Labels: map[string]string{"stage": "processor", "type": "processing"},
	}

	key := mc.getSummaryKey(event)
	expected := "counter:test.metric:stage=processor:type=processing"

	if key != expected {
		t.Errorf("Expected key %s, got %s", expected, key)
	}
}

func TestMetricsCollector_CleanupOldMetrics(t *testing.T) {
	logger := &mockLogger{}
	config := MetricsConfig{
		ChannelBufferSize: 1000,
		RetentionPeriod:   100 * time.Millisecond, // Very short for testing
		AggregationWindow: 10 * time.Second,
		MaxEvents:         1000,
		DumpInterval:      10 * time.Second,
	}
	mc := NewMetricsCollector(logger, config)

	// Add old events
	oldTime := time.Now().Add(-200 * time.Millisecond)
	oldEvent := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "old.metric",
		Value:     1.0,
		Timestamp: oldTime,
	}

	// Add recent events
	recentTime := time.Now()
	recentEvent := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "recent.metric",
		Value:     1.0,
		Timestamp: recentTime,
	}

	mc.events = []*MetricEvent{oldEvent, recentEvent}
	mc.summaries["old"] = &MetricSummary{LastUpdate: oldTime}
	mc.summaries["recent"] = &MetricSummary{LastUpdate: recentTime}

	mc.cleanupOldMetrics()

	if len(mc.events) != 1 {
		t.Errorf("Expected 1 event after cleanup, got %d", len(mc.events))
	}
	if mc.events[0] != recentEvent {
		t.Error("Expected recent event to remain")
	}

	if len(mc.summaries) != 1 {
		t.Errorf("Expected 1 summary after cleanup, got %d", len(mc.summaries))
	}
	if _, exists := mc.summaries["recent"]; !exists {
		t.Error("Expected recent summary to remain")
	}
}

func TestMetricsCollector_GetMethods(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	// Add some test data
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "test.metric",
		Value:     1.0,
		Labels:    map[string]string{"stage": "test"},
		Timestamp: time.Now(),
	}

	mc.events = []*MetricEvent{event}
	mc.summaries["test"] = &MetricSummary{Name: "test", Count: 1}

	// Test GetMetrics
	metrics := mc.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}
	if metrics.StageMetrics == nil {
		t.Error("Expected StageMetrics to be initialized")
	}

	// Test GetSummaries
	summaries := mc.GetSummaries()
	if summaries == nil {
		t.Error("Expected non-nil summaries")
	}
	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary, got %d", len(summaries))
	}

	// Test GetRecentEvents
	recentEvents := mc.GetRecentEvents(10)
	if len(recentEvents) != 1 {
		t.Errorf("Expected 1 recent event, got %d", len(recentEvents))
	}

	// Test GetRecentEvents with limit
	recentEvents = mc.GetRecentEvents(0)
	if len(recentEvents) != 1 {
		t.Errorf("Expected 1 recent event with 0 limit, got %d", len(recentEvents))
	}

	// Test GetRecentEvents with limit exceeding available events
	recentEvents = mc.GetRecentEvents(100)
	if len(recentEvents) != 1 {
		t.Errorf("Expected 1 recent event with high limit, got %d", len(recentEvents))
	}
}

func TestMetricsCollector_MaxEventsLimit(t *testing.T) {
	logger := &mockLogger{}
	config := MetricsConfig{
		ChannelBufferSize: 1000,
		RetentionPeriod:   10 * time.Minute,
		AggregationWindow: 1 * time.Minute,
		MaxEvents:         2, // Small limit for testing
		DumpInterval:      10 * time.Second,
	}
	mc := NewMetricsCollector(logger, config)

	// Add more events than the limit
	for i := 0; i < 5; i++ {
		event := &MetricEvent{
			Type:      MetricTypeCounter,
			Name:      "test.metric",
			Value:     float64(i),
			Timestamp: time.Now(),
		}
		mc.processMetricEvent(event)
	}

	if len(mc.events) != 2 {
		t.Errorf("Expected events to be limited to 2, got %d", len(mc.events))
	}

	// Check that the latest events are kept
	if mc.events[0].Value != 3.0 || mc.events[1].Value != 4.0 {
		t.Error("Expected the latest events to be kept")
	}
}

func TestMetricsCollector_CalculateThroughput(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	// Set up test data
	now := time.Now()
	mc.pipelineMetrics.ProcessedMessages = 10
	mc.pipelineMetrics.LastUpdated = now.Add(-1 * time.Second)

	// Add stage metrics
	stageName := "test-stage"
	mc.pipelineMetrics.StageMetrics[stageName] = &StageMetrics{
		StageName:         stageName,
		MessagesCompleted: 5,
		FirstProcessed:    now.Add(-2 * time.Second),
		LastProcessed:     now,
	}

	mc.calculateThroughput()

	// Check that throughput was calculated (exact value depends on timing)
	if mc.pipelineMetrics.ThroughputPerSecond < 0 {
		t.Error("Expected non-negative throughput")
	}

	stageMetrics := mc.pipelineMetrics.StageMetrics[stageName]
	if stageMetrics.ThroughputPerSecond < 0 {
		t.Error("Expected non-negative stage throughput")
	}
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	logger := &mockLogger{}
	mc := NewMetricsCollector(logger, DefaultMetricsConfig())

	err := mc.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer mc.Stop()

	// Wait for goroutines to start
	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	numGoroutines := 10
	numEventsPerGoroutine := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numEventsPerGoroutine; j++ {
				event := &MetricEvent{
					Type:      MetricTypeCounter,
					Name:      "concurrent.test",
					Value:     1.0,
					Labels:    map[string]string{"goroutine": string(rune(id))},
					Timestamp: time.Now(),
				}
				mc.SendMetric(event)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				mc.GetMetrics()
				mc.GetSummaries()
				mc.GetRecentEvents(10)
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Allow some time for processing
	time.Sleep(100 * time.Millisecond)

	// Verify that some events were processed
	metrics := mc.GetMetrics()
	if metrics == nil {
		t.Error("Expected non-nil metrics")
	}

	summaries := mc.GetSummaries()
	if summaries == nil {
		t.Error("Expected non-nil summaries")
	}
}

func TestMetricEvent_JSONSerialization(t *testing.T) {
	event := &MetricEvent{
		Type:      MetricTypeCounter,
		Name:      "test.metric",
		Value:     123.45,
		Labels:    map[string]string{"stage": "test", "type": "counter"},
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
		Message:   models.NewDataMessage([]byte("test"), "key", 0),
	}

	// Test that JSON marshaling works (Message field should be omitted due to json:"-")
	_, err := json.Marshal(event)
	if err != nil {
		t.Errorf("Failed to marshal MetricEvent: %v", err)
	}
}

func TestMetricSummary_JSONSerialization(t *testing.T) {
	summary := &MetricSummary{
		Name:       "test.summary",
		Type:       MetricTypeGauge,
		Count:      10,
		Sum:        100.0,
		Min:        5.0,
		Max:        20.0,
		Avg:        10.0,
		Labels:     map[string]string{"test": "value"},
		WindowSize: 1 * time.Minute,
		LastUpdate: time.Now(),
	}

	_, err := json.Marshal(summary)
	if err != nil {
		t.Errorf("Failed to marshal MetricSummary: %v", err)
	}
}

func TestPipelineMetrics_JSONSerialization(t *testing.T) {
	metrics := &PipelineMetrics{
		TotalMessages:         100,
		ProcessedMessages:     95,
		FailedMessages:        5,
		ErrorCount:            2,
		RetryCount:            3,
		AverageProcessingTime: 150 * time.Millisecond,
		ThroughputPerSecond:   10.5,
		BytesProcessed:        1024,
		StageMetrics: map[string]*StageMetrics{
			"test": {
				StageName:           "test",
				MessagesCompleted:   50,
				MessagesFailed:      2,
				AverageLatency:      100 * time.Millisecond,
				MinLatency:          50 * time.Millisecond,
				MaxLatency:          200 * time.Millisecond,
				TotalLatency:        5000 * time.Millisecond,
				ThroughputPerSecond: 5.0,
				FirstProcessed:      time.Now().Add(-1 * time.Hour),
				LastProcessed:       time.Now(),
			},
		},
		LastUpdated: time.Now(),
	}

	_, err := json.Marshal(metrics)
	if err != nil {
		t.Errorf("Failed to marshal PipelineMetrics: %v", err)
	}
}

func TestStageMetrics_JSONSerialization(t *testing.T) {
	stage := &StageMetrics{
		StageName:           "test-stage",
		MessagesCompleted:   100,
		MessagesFailed:      5,
		AverageLatency:      150 * time.Millisecond,
		MinLatency:          50 * time.Millisecond,
		MaxLatency:          300 * time.Millisecond,
		TotalLatency:        15000 * time.Millisecond,
		ThroughputPerSecond: 10.0,
		FirstProcessed:      time.Now().Add(-1 * time.Hour),
		LastProcessed:       time.Now(),
	}

	_, err := json.Marshal(stage)
	if err != nil {
		t.Errorf("Failed to marshal StageMetrics: %v", err)
	}
}
