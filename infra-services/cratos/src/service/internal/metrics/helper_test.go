package metrics

import (
	"servicegomodule/internal/models"
	"testing"
	"time"
)

const (
	expectedStartCollectorError = "Failed to start collector: %v"
	expectedAtLeastOneEvent     = "Expected at least one event"
	expectedMetricTypeCounter   = "Expected MetricTypeCounter, got %s"
	expectedValueFormat         = "Expected value %f, got %f"
	expectedNameFormat          = "Expected name '%s', got %s"
	expectedStageProcessor      = "Expected stage 'processor', got %s"
)

func TestNewMetricsHelper(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	stage := "test-stage"

	helper := NewMetricsHelper(collector, stage)

	if helper == nil {
		t.Fatal("Expected non-nil MetricsHelper")
	}
	if helper.collector != collector {
		t.Error("Expected collector to be set")
	}
	if helper.stage != stage {
		t.Errorf("Expected stage %s, got %s", stage, helper.stage)
	}
}

func TestMetricsHelper_RecordMessageReceived(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "input")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	msg.Origin = models.ChannelMessageOriginKafka

	helper.RecordMessageReceived(msg)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check that the event was recorded
	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1] // Get the last event
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != "message.received" {
		t.Errorf("Expected name 'message.received', got %s", event.Name)
	}
	if event.Value != 1 {
		t.Errorf("Expected value 1, got %f", event.Value)
	}
	if event.Labels["stage"] != "input" {
		t.Errorf("Expected stage 'input', got %s", event.Labels["stage"])
	}
	if event.Labels["origin"] != string(models.ChannelMessageOriginKafka) {
		t.Errorf("Expected origin 'kafka', got %s", event.Labels["origin"])
	}
}

func TestMetricsHelper_RecordMessageCompleted(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	msg.EntryTimestamp = time.Now().Add(-100 * time.Millisecond)

	helper.RecordMessageCompleted(msg)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) < 2 {
		t.Fatal("Expected at least two events (counter and timing)")
	}

	// Check counter event
	var counterEvent, timingEvent *MetricEvent
	for _, event := range events {
		if event.Name == "message.processed" {
			counterEvent = event
		} else if event.Name == "processing.duration" {
			timingEvent = event
		}
	}

	if counterEvent == nil {
		t.Fatal("Expected to find message.processed event")
	}
	if counterEvent.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", counterEvent.Type)
	}
	if counterEvent.Value != 1 {
		t.Errorf("Expected value 1, got %f", counterEvent.Value)
	}

	if timingEvent == nil {
		t.Fatal("Expected to find processing.duration event")
	}
	if timingEvent.Type != MetricTypeTiming {
		t.Errorf("Expected MetricTypeTiming, got %s", timingEvent.Type)
	}
}

func TestMetricsHelper_RecordMessageFailed(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	reason := "processing_error"

	helper.RecordMessageFailed(msg, reason)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != "message.failed" {
		t.Errorf("Expected name 'message.failed', got %s", event.Name)
	}
	if event.Labels["reason"] != reason {
		t.Errorf("Expected reason '%s', got %s", reason, event.Labels["reason"])
	}
}

func TestMetricsHelper_RecordError(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	errorType := "validation_error"

	helper.RecordError(msg, errorType)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != "error.occurred" {
		t.Errorf("Expected name 'error.occurred', got %s", event.Name)
	}
	if event.Labels["type"] != errorType {
		t.Errorf("Expected type '%s', got %s", errorType, event.Labels["type"])
	}
}

func TestMetricsHelper_RecordRetry(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	attempt := 2

	helper.RecordRetry(msg, attempt)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != "retry.attempted" {
		t.Errorf("Expected name 'retry.attempted', got %s", event.Name)
	}
	if event.Labels["attempt"] != string(rune(attempt)) {
		t.Errorf("Expected attempt '%s', got %s", string(rune(attempt)), event.Labels["attempt"])
	}
}

func TestMetricsHelper_RecordStageLatency(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	duration := 150 * time.Millisecond
	operation := "validate_input"

	helper.RecordStageLatency(duration, operation)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeTiming {
		t.Errorf("Expected MetricTypeTiming, got %s", event.Type)
	}
	if event.Name != "stage.latency" {
		t.Errorf("Expected name 'stage.latency', got %s", event.Name)
	}
	if event.Value != float64(duration.Milliseconds()) {
		t.Errorf("Expected value %f, got %f", float64(duration.Milliseconds()), event.Value)
	}
	if event.Labels["operation"] != operation {
		t.Errorf("Expected operation '%s', got %s", operation, event.Labels["operation"])
	}
}

func TestMetricsHelper_RecordStageCompleted(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	duration := 100 * time.Millisecond

	helper.RecordStageCompleted(msg, duration)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != "stage.completed" {
		t.Errorf("Expected name 'stage.completed', got %s", event.Name)
	}
	if event.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, event.Duration)
	}
}

func TestMetricsHelper_RecordStageFailed(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	msg := models.NewDataMessage([]byte("test"), "key", 0)
	reason := "timeout"

	helper.RecordStageFailed(msg, reason)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != "stage.failed" {
		t.Errorf("Expected name 'stage.failed', got %s", event.Name)
	}
	if event.Labels["reason"] != reason {
		t.Errorf("Expected reason '%s', got %s", reason, event.Labels["reason"])
	}
}

func TestMetricsHelper_RecordGauge(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	name := "queue.size"
	value := 42.0
	labels := map[string]string{"queue": "input"}

	helper.RecordGauge(name, value, labels)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeGauge {
		t.Errorf("Expected MetricTypeGauge, got %s", event.Type)
	}
	if event.Name != name {
		t.Errorf("Expected name '%s', got %s", name, event.Name)
	}
	if event.Value != value {
		t.Errorf("Expected value %f, got %f", value, event.Value)
	}
	if event.Labels["stage"] != "processor" {
		t.Errorf("Expected stage 'processor', got %s", event.Labels["stage"])
	}
	if event.Labels["queue"] != "input" {
		t.Errorf("Expected queue 'input', got %s", event.Labels["queue"])
	}
}

func TestMetricsHelper_RecordCounter(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	name := "rule.applied"
	value := 5.0
	labels := map[string]string{"rule_type": "validation"}

	helper.RecordCounter(name, value, labels)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeCounter {
		t.Errorf("Expected MetricTypeCounter, got %s", event.Type)
	}
	if event.Name != name {
		t.Errorf("Expected name '%s', got %s", name, event.Name)
	}
	if event.Value != value {
		t.Errorf("Expected value %f, got %f", value, event.Value)
	}
	if event.Labels["stage"] != "processor" {
		t.Errorf("Expected stage 'processor', got %s", event.Labels["stage"])
	}
	if event.Labels["rule_type"] != "validation" {
		t.Errorf("Expected rule_type 'validation', got %s", event.Labels["rule_type"])
	}
}

func TestMetricsHelper_RecordHistogram(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	name := "request.size"
	value := 1024.0
	labels := map[string]string{"endpoint": "/api/data"}

	helper.RecordHistogram(name, value, labels)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) == 0 {
		t.Fatal("Expected at least one event")
	}

	event := events[len(events)-1]
	if event.Type != MetricTypeHistogram {
		t.Errorf("Expected MetricTypeHistogram, got %s", event.Type)
	}
	if event.Name != name {
		t.Errorf("Expected name '%s', got %s", name, event.Name)
	}
	if event.Value != value {
		t.Errorf("Expected value %f, got %f", value, event.Value)
	}
	if event.Labels["stage"] != "processor" {
		t.Errorf("Expected stage 'processor', got %s", event.Labels["stage"])
	}
	if event.Labels["endpoint"] != "/api/data" {
		t.Errorf("Expected endpoint '/api/data', got %s", event.Labels["endpoint"])
	}
}

func TestMetricsHelper_GetCollector(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "test")

	retrieved := helper.GetCollector()
	if retrieved != collector {
		t.Error("Expected GetCollector to return the same collector instance")
	}
}

func TestMetricsHelper_WithNilLabels(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "processor")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	// Test all methods with nil labels
	helper.RecordGauge("test.gauge", 1.0, nil)
	helper.RecordCounter("test.counter", 1.0, nil)
	helper.RecordHistogram("test.histogram", 1.0, nil)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	events := collector.GetRecentEvents(10)
	if len(events) < 3 {
		t.Fatal("Expected at least 3 events")
	}

	// Check that stage was added to labels even when labels were nil
	for _, event := range events {
		if event.Labels["stage"] != "processor" {
			t.Errorf("Expected stage 'processor' to be added, got %s", event.Labels["stage"])
		}
	}
}

func TestMetricsHelper_IntegrationWithCollector(t *testing.T) {
	logger := &mockLogger{}
	collector := NewMetricsCollector(logger, DefaultMetricsConfig(nil))
	helper := NewMetricsHelper(collector, "integration_test")

	err := collector.Start()
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	defer collector.Stop()

	// Send a variety of metrics
	msg := models.NewDataMessage([]byte("test"), "key", 0)

	helper.RecordMessageReceived(msg)
	helper.RecordMessageCompleted(msg)
	helper.RecordCounter("custom.metric", 10.0, map[string]string{"type": "test"})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Check pipeline metrics
	pipelineMetrics := collector.GetMetrics()
	if pipelineMetrics.TotalMessages != 1 {
		t.Errorf("Expected TotalMessages 1, got %d", pipelineMetrics.TotalMessages)
	}
	if pipelineMetrics.ProcessedMessages != 1 {
		t.Errorf("Expected ProcessedMessages 1, got %d", pipelineMetrics.ProcessedMessages)
	}

	// Check summaries
	summaries := collector.GetSummaries()
	if len(summaries) == 0 {
		t.Error("Expected at least one summary")
	}

	// Check that custom counter was recorded
	found := false
	for _, summary := range summaries {
		if summary.Name == "custom.metric" && summary.Type == MetricTypeCounter {
			found = true
			if summary.Sum != 10.0 {
				t.Errorf("Expected custom metric sum 10.0, got %f", summary.Sum)
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find custom.metric counter in summaries")
	}
}
