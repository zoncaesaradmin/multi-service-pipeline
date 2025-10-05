package tasks

import (
	"context"
	"fmt"
	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"sync"
	relib "telemetry/utils/ruleenginelib"
	"time"
)

const (
	TaskCheckerInitInterval = time.Duration(5 * time.Minute)
	TaskCheckerInterval     = time.Duration(20 * time.Minute)
)

type RuleTasksHandler struct {
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper

	// copied from ruleengineconfig
	ruleTasksTopic              string
	ruleTasksConsKafkaConfigMap map[string]any
	ruleTasksProdKafkaConfigMap map[string]any

	// fields related to background task of applying rule changes to DB records
	rlogger          logging.Logger // for both rules msg and rule tasks msg handling
	ruleTaskProducer messagebus.Producer
	ruleTaskConsumer messagebus.Consumer
	taskStore        *TaskStore // OpenSearch task persistence
	isLeader         bool
	leaderMutex      sync.RWMutex
	leaderCancel     context.CancelFunc
	leaderCtx        context.Context

	// DB handler
	dbRecordHandler *DbRecordHandler

	// Metrics
	totalProcessed int64
	totalFailed    int64
	metricsMutex   sync.RWMutex
}

type taskJob struct {
	ctx         context.Context
	logger      logging.Logger
	taskBytes   []byte
	taskID      string
	submittedAt time.Time // Track when job was submitted for metrics
}

type TaskDataType struct {
	RuleEvent string
	Rule      *relib.RuleDefinition
	OldRule   *relib.RuleDefinition
}

// TaskMetrics tracks performance metrics
type TaskMetrics struct {
	TotalProcessed    int64
	TotalFailed       int64
	AverageProcessing time.Duration
	QueueLength       int
}

// GetTaskMetrics returns current task processing metrics
func (rh *RuleTasksHandler) GetTaskMetrics() TaskMetrics {
	rh.metricsMutex.RLock()
	defer rh.metricsMutex.RUnlock()

	return TaskMetrics{
		TotalProcessed: rh.totalProcessed,
		TotalFailed:    rh.totalFailed,
		// AverageProcessing would require more complex tracking
	}
}

func NewRuleTasksHandler(logger logging.Logger, topic string, pconfig map[string]any, cconfig map[string]any, inputSink chan<- *models.ChannelMessage, metricHelper *metrics.MetricsHelper) *RuleTasksHandler {

	drh := NewDbRecordHandler(logger.WithField("component", "dbRecordHandler"), inputSink, metricHelper)
	h := &RuleTasksHandler{
		ruleTasksTopic:              topic,
		isLeader:                    false,
		rlogger:                     logger,
		metricsHelper:               metricHelper,
		taskStore:                   NewTaskStore(logger), // Initialize task store
		ruleTasksProdKafkaConfigMap: pconfig,
		ruleTasksConsKafkaConfigMap: cconfig,
		dbRecordHandler:             drh,
	}

	logger.Infow("Initialized Rule Tasks Handler", "ruleTasksTopic", topic)
	return h
}

func (rh *RuleTasksHandler) Start() error {
	rh.rlogger.Infow("Starting Rule Tasks Handler", "topic", rh.ruleTasksTopic)

	// Create context for cancellation
	rh.ctx, rh.cancel = context.WithCancel(context.Background())

	// Initialize task store
	if err := rh.taskStore.Initialize(rh.ctx); err != nil {
		rh.rlogger.Errorw("Failed to initialize task store", "error", err)
		// Continue without task store - degraded mode
	}

	if err := rh.initializeRuleTaskHandling(); err != nil {
		return err
	}

	rh.dbRecordHandler.Start()

	return nil
}

func (rh *RuleTasksHandler) Stop() error {
	rh.rlogger.Info("Stopping Rule Tasks Handler...")

	rh.dbRecordHandler.Stop()

	if rh.cancel != nil {
		rh.cancel()
	}

	// Close all consumers and producer

	if rh.ruleTaskConsumer != nil {
		if err := rh.ruleTaskConsumer.Close(); err != nil {
			rh.rlogger.Errorw("Failed to close rule task consumer", "error", err)
			return err
		}
	}

	if rh.ruleTaskProducer != nil {
		if err := rh.ruleTaskProducer.Close(); err != nil {
			rh.rlogger.Errorw("Failed to close rule task producer", "error", err)
			return err
		}
	}

	rh.rlogger.Info("Stopped Rule Tasks Handler")
	return nil
}

// initializeRuleTaskHandling sets up rule task producer and consumer
func (rh *RuleTasksHandler) initializeRuleTaskHandling() error {
	// Initialize producer for distributing rule tasks
	rh.ruleTaskProducer = messagebus.NewProducer(rh.ruleTasksProdKafkaConfigMap, "ruleTaskProducer"+utils.GetEnv("HOSTNAME", ""))

	// Initialize rule task consumer with shared group for task distribution
	ruleTaskGroup := "ruleTaskConsGroup-shared"
	rh.ruleTaskConsumer = messagebus.NewConsumer(rh.ruleTasksConsKafkaConfigMap, ruleTaskGroup)

	rh.setupRuleTaskConsumerCallbacks()

	if err := rh.ruleTaskConsumer.Subscribe([]string{rh.ruleTasksTopic}); err != nil {
		rh.rlogger.Errorw("RULE TASK HANDLER - Failed to subscribe to rule tasks topic", "error", err)
		return fmt.Errorf("failed to subscribe to rule tasks topic: %w", err)
	}

	return nil
}

// setupRuleTaskConsumerCallbacks configures callbacks for rule task consumer
func (rh *RuleTasksHandler) setupRuleTaskConsumerCallbacks() {
	rh.ruleTaskConsumer.OnMessage(func(message *messagebus.Message) {
		rh.handleRuleTaskMessage(message)
	})

	rh.ruleTaskConsumer.OnAssign(func(assignments []messagebus.PartitionAssignment) {
		rh.handlePartitionAssignment(assignments)
	})

	rh.ruleTaskConsumer.OnRevoke(func(revoked []messagebus.PartitionAssignment) {
		rh.handlePartitionRevocation(revoked)
	})
}

// handleRuleTaskMessage processes incoming rule task messages
// Flow: Read from cisco_nir-ruletasks topic - Store in opensearch - Commit offset
func (rh *RuleTasksHandler) handleRuleTaskMessage(message *messagebus.Message) {
	if message == nil {
		return
	}

	// Extract or generate trace ID from message headers
	traceID := utils.ExtractTraceID(message.Headers)
	// Use trace-aware logger for this message
	msgLogger := utils.WithTraceLoggerFromID(rh.rlogger, traceID)

	msgLogger.Debugw("RULE TASK HANDLER - Received task", "size", len(message.Value))

	// Commit the Kafka offset AFTER successful storage
	if err := rh.commitMessage(msgLogger, message); err != nil {
		msgLogger.Errorw("RULE TASK HANDLER - Failed to commit message", "error", err, "taskID", message.Headers["taskID"])
		// Even if commit fails, we've stored the task, so we can continue processing
	}
}

func (rh *RuleTasksHandler) commitMessage(logger logging.Logger, message *messagebus.Message) error {
	commitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rh.ruleTaskConsumer.Commit(commitCtx, message); err != nil {
		logger.Errorw("RULE TASK HANDLER - kafka commit failed", "error", err, "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
		return err
	}

	logger.Debugw("RULE TASk HANDLER - Message committed successfully", "topic", message.Topic, "partition", message.Partition, "offset", message.Offset)
	return nil
}

// handlePartitionAssignment manages partition assignment and leadership
func (rh *RuleTasksHandler) handlePartitionAssignment(assignments []messagebus.PartitionAssignment) {
	rh.rlogger.Infow("RULE TASK HANDLER - Assigned partitions", "partitions", assignments)
	for _, assignment := range assignments {
		if assignment.Partition == 0 {
			rh.SetLeader(true)
			break
		}
	}
}

// handlePartitionRevocation manages partition revocation and leadership transfer
func (rh *RuleTasksHandler) handlePartitionRevocation(revoked []messagebus.PartitionAssignment) {
	rh.rlogger.Infow("RULE TASK HANDLER - Revoked partitions", "partitions", revoked)
	for _, r := range revoked {
		if r.Partition == 0 {
			rh.SetLeader(false)
			break
		}
	}
}

// DistributeRuleTask distributes rule task with old rule definitions included
func (rh *RuleTasksHandler) DistributeRuleTask(l logging.Logger, traceID string, eventType string, rule *relib.RuleDefinition, oldRule *relib.RuleDefinition) bool {
	return true
}

func (rh *RuleTasksHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":         "running",
		"ruleTasksTopic": rh.ruleTasksTopic,
		"isLeader":       rh.Leader(),
	}
}

// return current leadership state
func (rh *RuleTasksHandler) Leader() bool {
	rh.leaderMutex.RLock()
	defer rh.leaderMutex.RUnlock()
	return rh.isLeader
}

// set leadership state
func (rh *RuleTasksHandler) SetLeader(isLeader bool) {
	rh.leaderMutex.Lock()
	defer rh.leaderMutex.Unlock()
	rh.isLeader = isLeader
	if isLeader {
		rh.leaderCtx, rh.leaderCancel = context.WithCancel(rh.ctx)
		go rh.periodicRuleTaskChecker(rh.leaderCtx)
	} else if rh.leaderCancel != nil {
		rh.leaderCancel()
		rh.leaderCancel = nil
		rh.leaderCtx = nil
	}
}

func (rh *RuleTasksHandler) periodicRuleTaskChecker(ctx context.Context) {

	select {
	case <-time.After(TaskCheckerInitInterval):
		rh.processPendingTasks(ctx)
	case <-ctx.Done():
		rh.rlogger.Info("periodic rule task checker is cancelled before work started")
	}

	ticker := time.NewTicker(TaskCheckerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			rh.rlogger.Info("periodic rule task checker is cancelled")
			return
		case <-ticker.C:
			rh.processPendingTasks(ctx)
		}
	}
}

func (rh *RuleTasksHandler) processPendingTasks(ctx context.Context) {
}
