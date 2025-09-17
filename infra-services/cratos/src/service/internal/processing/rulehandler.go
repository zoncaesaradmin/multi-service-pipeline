package processing

import (
	"context"
	"encoding/json"
	"fmt"
	eapi "servicegomodule/external/api"
	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"strings"
	"sync"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
	"time"
)

const (
	TaskCheckerInitInterval = time.Duration(5 * time.Minute)
	TaskCheckerInterval     = time.Duration(20 * time.Minute)
)

type RuleEngineConfig struct {
	RulesTopic                  string
	PollTimeout                 time.Duration
	RuleEngLibLogging           logging.LoggerConfig
	RulesKafkaConfigMap         map[string]any
	RuleHandlerLogging          logging.LoggerConfig
	RuleTasksTopic              string
	RuleTasksConsKafkaConfigMap map[string]any
	RuleTasksProdKafkaConfigMap map[string]any
}

type RuleEngineHandler struct {
	ruleconsumer  messagebus.Consumer
	config        RuleEngineConfig
	plogger       logging.Logger // handle to pipeline logger
	reInst        relib.RuleEngineType
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper

	// fields related to background task of applying rule changes to DB records
	rlogger          logging.Logger // for both rules msg and rule tasks msg handling
	ruleTaskProducer messagebus.Producer
	ruleTaskConsumer messagebus.Consumer
	isLeader         bool
	leaderMutex      sync.RWMutex
	leaderCancel     context.CancelFunc
	leaderCtx        context.Context
}

func NewRuleHandler(config RuleEngineConfig, logger logging.Logger, metricHelper *metrics.MetricsHelper) *RuleEngineHandler {
	// use simple filename - path resolution is handled by messagebus config loader
	consumer := messagebus.NewConsumer(config.RulesKafkaConfigMap, "ruleConsGroup"+utils.GetEnv("HOSTNAME", ""))

	reInst := relib.CreateRuleEngineInstance(
		// this creates separate logger for rule engine lib handling
		relib.LoggerInfo{
			ServiceName: config.RuleEngLibLogging.ServiceName,
			Level:       config.RuleEngLibLogging.Level.String(),
			FilePath:    config.RuleEngLibLogging.FilePath,
		},
		[]string{relib.RuleTypeMgmt})

	rlogger, err := logging.NewLogger(&config.RuleHandlerLogging)
	if err != nil {
		logger.Errorw("RULE HANDLER - Failed to create rule tasks logger, using ruleengine logger", "error", err)
		rlogger = logger
	}

	h := &RuleEngineHandler{
		ruleconsumer:  consumer,
		config:        config,
		plogger:       logger,
		isLeader:      false,
		reInst:        reInst,
		rlogger:       rlogger,
		metricsHelper: metricHelper,
	}

	logger.Info("Initialized Rule Handler module")
	rlogger.Infow("Initialized Rule Handler", "ruleTopic", config.RulesTopic, "ruleTasksTopic", h.config.RuleTasksTopic)
	return h
}

func (rh *RuleEngineHandler) Start() error {
	rh.rlogger.Infow("Starting Rule Engine Handler", "topic", rh.config.RulesTopic)

	// Create context for cancellation
	rh.ctx, rh.cancel = context.WithCancel(context.Background())

	if err := rh.initializeRuleTaskHandling(); err != nil {
		return err
	}

	if err := rh.setupRuleConsumer(); err != nil {
		return err
	}

	_ = eapi.FetchAlertMappings()

	return nil
}

// initializeRuleTaskHandling sets up rule task producer and consumer
func (rh *RuleEngineHandler) initializeRuleTaskHandling() error {
	// Initialize producer for distributing rule tasks
	rh.ruleTaskProducer = messagebus.NewProducer(rh.config.RuleTasksProdKafkaConfigMap, "ruleTaskProducer"+utils.GetEnv("HOSTNAME", ""))

	// Initialize rule task consumer with shared group for task distribution
	ruleTaskGroup := "ruleTaskConsGroup-shared"
	rh.ruleTaskConsumer = messagebus.NewConsumer(rh.config.RuleTasksConsKafkaConfigMap, ruleTaskGroup)

	rh.setupRuleTaskConsumerCallbacks()

	if err := rh.ruleTaskConsumer.Subscribe([]string{rh.config.RuleTasksTopic}); err != nil {
		rh.rlogger.Errorw("RULE TASK HANDLER - Failed to subscribe to rule tasks topic", "error", err)
		return fmt.Errorf("failed to subscribe to rule tasks topic: %w", err)
	}

	return nil
}

// setupRuleTaskConsumerCallbacks configures callbacks for rule task consumer
func (rh *RuleEngineHandler) setupRuleTaskConsumerCallbacks() {
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
func (rh *RuleEngineHandler) handleRuleTaskMessage(message *messagebus.Message) {
	if message == nil {
		return
	}

	// Extract or generate trace ID from message headers
	traceID := utils.ExtractTraceID(message.Headers)
	// Use trace-aware logger for this message
	msgLogger := utils.WithTraceLoggerFromID(rh.rlogger, traceID)

	msgLogger.Debugw("RULE TASK HANDLER - Received task", "size", len(message.Value))

	// Process rule task with structured data unmarshaling
	sendToDBBatchProcessor(rh.ctx, msgLogger, message.Value, rh.reInst)

	if err := rh.ruleTaskConsumer.Commit(context.Background(), message); err != nil {
		msgLogger.Errorw("RULE TASK HANDLER - Failed to commit message", "error", err)
	}
}

// handlePartitionAssignment manages partition assignment and leadership
func (rh *RuleEngineHandler) handlePartitionAssignment(assignments []messagebus.PartitionAssignment) {
	rh.rlogger.Infow("RULE TASK HANDLER - Assigned partitions", "partitions", assignments)
	for _, assignment := range assignments {
		if assignment.Partition == 0 {
			rh.SetLeader(true)
			break
		}
	}
}

// handlePartitionRevocation manages partition revocation and leadership transfer
func (rh *RuleEngineHandler) handlePartitionRevocation(revoked []messagebus.PartitionAssignment) {
	rh.rlogger.Infow("RULE TASK HANDLER - Revoked partitions", "partitions", revoked)
	for _, r := range revoked {
		if r.Partition == 0 {
			rh.SetLeader(false)
			break
		}
	}
}

// setupRuleConsumer configures the main rules topic consumer
func (rh *RuleEngineHandler) setupRuleConsumer() error {
	rh.ruleconsumer.OnMessage(func(message *messagebus.Message) {
		rh.handleRuleMessage(message)
	})

	if err := rh.ruleconsumer.Subscribe([]string{rh.config.RulesTopic}); err != nil {
		rh.rlogger.Errorw("Failed to subscribe to rules topic", "error", err)
		return fmt.Errorf("failed to subscribe to rules topic: %w", err)
	}

	return nil
}

// handleRuleMessage processes incoming rule messages
func (rh *RuleEngineHandler) handleRuleMessage(message *messagebus.Message) {
	if message == nil {
		return
	}

	// Extract or generate trace ID from message headers
	traceID := utils.ExtractTraceID(message.Headers)
	// Use trace-aware logger for this message
	msgLogger := utils.WithTraceLoggerFromID(rh.rlogger, traceID)

	msgLogger.Debugw("RULE HANDLER - Received message", "size", len(message.Value))

	userMeta := relib.UserMetaData{TraceId: traceID}
	res, err := rh.reInst.HandleRuleEvent(message.Value, userMeta)
	if err != nil {
		msgLogger.Errorw("RULE HANDLER - Failed to handle rule event", "error", err)
	} else if res != nil && len(res.RuleJSON) > 0 {
		rh.processRuleEvent(msgLogger, traceID, res)
	}

	if err := rh.ruleconsumer.Commit(context.Background(), message); err != nil {
		msgLogger.Errorw("RULE HANDLER - Failed to commit message", "error", err)
	}
}

// processRuleEvent handles rule event processing results and task distribution
func (rh *RuleEngineHandler) processRuleEvent(l logging.Logger, traceID string, res *relib.RuleMsgResult) {
	l.Debugw("RULE HANDLER - Converted rule JSON", "action", res.Action, "convertedRule", string(res.RuleJSON))

	if rh.Leader() {
		if rh.distributeRuleTask(l, traceID, res) {
			l.Debugw("RULE HANDLER - Successfully distributed rule task", "action", res.Action)
		}
	} else {
		l.Debug("RULE HANDLER - Not leader, skipping rule task distribution")
	}
}

func (rh *RuleEngineHandler) Stop() error {
	rh.rlogger.Info("Stopping Rule Engine Handler...")

	if rh.cancel != nil {
		rh.cancel()
	}

	// Close all consumers and producer

	if rh.ruleconsumer != nil {
		if err := rh.ruleconsumer.Close(); err != nil {
			rh.rlogger.Errorw("Failed to close consumer", "error", err)
			return err
		}
	}

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

	rh.plogger.Info("Stopped Rule Engine Handler")
	return nil
}

func sendToDBBatchProcessor(ctx context.Context, logger logging.Logger, taskBytes []byte, reInst relib.RuleEngineType) {
	logger.Info("RULE TASK HANDLER - sending rule to DB batch processor")
}

func (rh *RuleEngineHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":         "running",
		"rulesTopic":     rh.config.RulesTopic,
		"ruleTasksTopic": rh.config.RuleTasksTopic,
		"isLeader":       rh.Leader(),
		"poll_timeout":   rh.config.PollTimeout.String(),
	}
}

func (rh *RuleEngineHandler) applyRuleToRecord(l logging.Logger, aObj *alert.Alert, origin models.ChannelMessageOrigin) (*alert.Alert, error) {
	if !needsRuleProcessing(aObj) {
		l.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - skipped rule lookup")
		return aObj, nil
	}

	convStartTime := time.Now()
	convRecord := ConvertAlertObjectToRuleEngineInput(aObj)
	l.WithField("recId", recordIdentifier(aObj)).Debugw("RECORD PROC - after conversion", "convertedData", convRecord, "timeTaken", fmt.Sprintf("%v", time.Since(convStartTime)))

	lookupStartTime := time.Now()
	lookupResult := rh.reInst.EvaluateRules(relib.Data(convRecord))
	rh.metricsHelper.RecordStageLatency(time.Since(lookupStartTime), "rule_lookup")
	rh.metricsHelper.RecordCounter("criteria.count", float64(lookupResult.CritCount), map[string]string{
		"hit": fmt.Sprintf("%v", lookupResult.IsRuleHit),
	})
	lookupTimeTaken := time.Since(lookupStartTime)
	rh.metricsHelper.RecordStageCompleted(nil, lookupTimeTaken)
	l.WithField("recId", recordIdentifier(aObj)).Debugw("RECORD PROC - post lookup", "lookupResult", lookupResult, "timeTaken", fmt.Sprintf("%v", lookupTimeTaken))

	if lookupResult.IsRuleHit {
		rh.handleRuleHit(l, aObj, lookupResult)
	} else {
		rh.handleRuleMiss(l, aObj, lookupResult, origin)
	}

	return aObj, nil
}

func (rh *RuleEngineHandler) handleRuleHit(l logging.Logger, aObj *alert.Alert, lookupResult relib.RuleLookupResult) {
	aObj.RuleId = lookupResult.RuleUUID
	toPrintActionMap := make(map[string]string, 0)

	for _, action := range lookupResult.Actions {
		toPrintActionMap[action.ActionType] = action.ActionValueStr
		applyRuleAction(aObj, action.ActionType, action.ActionValueStr)
	}

	l.Debugw("RECORD PROC - rule hit", "matchCriteria", lookupResult.CriteriaHit, "actionsApplied", toPrintActionMap)
}

func (rh *RuleEngineHandler) handleRuleMiss(l logging.Logger, aObj *alert.Alert, lookupResult relib.RuleLookupResult, origin models.ChannelMessageOrigin) {
	toPrintActionMap := make(map[string]string, 0)

	if origin == models.ChannelMessageOriginDb {
		rh.revertToDefaultActions(aObj, lookupResult)
	}

	l.WithField("recId", recordIdentifier(aObj)).Infow("RECORD PROC - no rule hit", "actionsApplied", toPrintActionMap)
}

func (rh *RuleEngineHandler) revertToDefaultActions(aObj *alert.Alert, lookupResult relib.RuleLookupResult) {
	allSupportedActions := []string{
		relib.RuleActionAcknowledge,
		relib.RuleActionSeverityOverride,
		relib.RuleActionCustomizeRecommendation,
	}

	for _, sAction := range allSupportedActions {
		if contains, actionValue := containsActionType(lookupResult.Actions, sAction); contains {
			applyRuleAction(aObj, sAction, actionValue)
		} else {
			applyDefaultAction(aObj, sAction)
		}
	}
}

func applyRuleAction(aObj *alert.Alert, actionType string, actionValue string) {
	switch actionType {
	case relib.RuleActionSeverityOverride:
		aObj.Severity = actionValue
	case relib.RuleActionAcknowledge:
		aObj.Acknowledged = true
		aObj.AckTs = time.Now().UTC().Format(time.RFC3339)
		aObj.AutoAck = true
	case relib.RuleActionCustomizeRecommendation:
		aObj.IsRuleCustomReco = true
		aObj.RuleCustomRecoStr = strings.Split(actionValue, ",")
	default:
	}
}

func applyDefaultAction(aObj *alert.Alert, actionType string) {
	switch actionType {
	case relib.RuleActionSeverityOverride:
		// fetch default severity from mapping and apply
		//aObj.Severity = "minor"
	case relib.RuleActionAcknowledge:
		aObj.Acknowledged = false
		aObj.AckTs = ""
		aObj.AutoAck = false
	case relib.RuleActionCustomizeRecommendation:
		aObj.IsRuleCustomReco = false
		aObj.RuleCustomRecoStr = []string{}
	default:
	}
}

// return current leadership state
func (rh *RuleEngineHandler) Leader() bool {
	rh.leaderMutex.RLock()
	defer rh.leaderMutex.RUnlock()
	return rh.isLeader
}

// set leadership state
func (rh *RuleEngineHandler) SetLeader(isLeader bool) {
	rh.leaderMutex.Lock()
	defer rh.leaderMutex.Unlock()
	rh.isLeader = isLeader
	if isLeader {
		rh.leaderCtx, rh.leaderCancel = context.WithCancel(rh.ctx)
		go periodicRuleTaskChecker(rh.rlogger, rh.leaderCtx)
	} else if rh.leaderCancel != nil {
		rh.leaderCancel()
		rh.leaderCancel = nil
		rh.leaderCtx = nil
	}
}

func (rh *RuleEngineHandler) distributeRuleTask(l logging.Logger, traceID string, res *relib.RuleMsgResult) bool {
	// Marshal the RuleMsgResult directly since it already has Action and RuleJSON
	taskBytes, err := json.Marshal(res)
	if err != nil {
		l.Errorw("RULE HANDLER - Failed to marshal task data", "error", err)
		return false
	}
	out := &messagebus.Message{
		Topic: rh.config.RuleTasksTopic,
		Value: taskBytes,
		Headers: map[string]string{
			utils.TraceIDHeader: traceID,
		},
	}

	ctx, cancel := context.WithTimeout(rh.ctx, 5*time.Second)
	defer cancel()

	if _, _, err := rh.ruleTaskProducer.Send(ctx, out); err != nil {
		l.Errorw("RULE HANDLER - Failed to send rule task message", "error", err)
		return false
	}

	return true
}

func containsActionType(actions []relib.RuleHitAction, actionType string) (bool, string) {
	for _, action := range actions {
		if action.ActionType == actionType {
			return true, action.ActionValueStr
		}
	}
	return false, ""
}

func periodicRuleTaskChecker(l logging.Logger, ctx context.Context) {

	select {
	case <-time.After(TaskCheckerInitInterval):
		processPendingTasks()
	case <-ctx.Done():
		l.Info("periodic rule task checker is cancelled before work started")
	}

	ticker := time.NewTicker(TaskCheckerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			l.Info("periodic rule task checker is cancelled")
			return
		case <-ticker.C:
			processPendingTasks()
		}
	}
}

func processPendingTasks() {
	// add handling to fetch rule tasks from DB and generate tasks into rule tasks topic
}
