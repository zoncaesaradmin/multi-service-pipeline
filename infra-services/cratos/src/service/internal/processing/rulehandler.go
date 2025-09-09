package processing

import (
	"context"
	"fmt"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"strings"
	"sync"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
	"time"
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
	ruleconsumer messagebus.Consumer
	config       RuleEngineConfig
	plogger      logging.Logger // handle to pipeline logger
	reInst       relib.RuleEngineType
	ctx          context.Context
	cancel       context.CancelFunc

	// fields related to background task of applying rule changes to DB records
	rlogger          logging.Logger // for both rules msg and rule tasks msg handling
	ruleTaskProducer messagebus.Producer
	ruleTaskConsumer messagebus.Consumer
	isLeader         bool
	leaderMutex      sync.RWMutex
}

func NewRuleHandler(config RuleEngineConfig, logger logging.Logger) *RuleEngineHandler {
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
		ruleconsumer: consumer,
		config:       config,
		plogger:      logger,
		isLeader:     false,
		reInst:       reInst,
		rlogger:      rlogger,
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

	rh.rlogger.Debugw("RULE TASK HANDLER - Received task", "size", len(message.Value))

	// Process rule task with structured data unmarshaling
	sendToDBBatchProcessor(rh.ctx, rh.rlogger, message.Value, "RULE_TASK")

	if err := rh.ruleTaskConsumer.Commit(context.Background(), message); err != nil {
		rh.rlogger.Errorw("RULE TASK HANDLER - Failed to commit message", "error", err)
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

	rh.rlogger.Debugw("RULE HANDLER - Received message", "size", len(message.Value))

	res, err := rh.reInst.HandleRuleEvent(message.Value)
	if err != nil {
		rh.rlogger.Errorw("RULE HANDLER - Failed to handle rule event", "error", err)
	} else if res != nil && len(res.RuleJSON) > 0 {
		rh.processRuleResult(res)
	}

	if err := rh.ruleconsumer.Commit(context.Background(), message); err != nil {
		rh.rlogger.Errorw("RULE HANDLER - Failed to commit message", "error", err)
	}
}

// processRuleResult handles rule processing results and task distribution
func (rh *RuleEngineHandler) processRuleResult(res *relib.RuleMsgResult) {
	rh.rlogger.Debugw("RULE HANDLER - Converted rule JSON", "action", res.Action, "convertedRule", string(res.RuleJSON))

	if rh.Leader() {
		if rh.distributeRuleTask(res) {
			rh.rlogger.Debugw("RULE HANDLER - Successfully distributed rule task", "action", res.Action)
		}
	} else {
		rh.rlogger.Debug("RULE HANDLER - Not leader, skipping rule task distribution")
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

func sendToDBBatchProcessor(ctx context.Context, logger logging.Logger, ruleBytes []byte, action string) {
	logger.Infow("RULE TASK HANDLER - sending rule to DB batch processor", "action", action)
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

func (rh *RuleEngineHandler) applyRuleToRecord(l logging.Logger, aObj *alert.Alert) (*alert.Alert, error) {
	if needsRuleProcessing(aObj) {
		convRecord := ConvertAlertObjectToRuleEngineInput(aObj)
		l.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - converted data: %v", convRecord)
		lookupResult := rh.reInst.EvaluateRules(relib.Data(convRecord))
		if lookupResult.IsRuleHit {
			// rule matched
			aObj.RuleId = lookupResult.RuleUUID
			toPrintActionMap := make(map[string]any, 0)
			for _, action := range lookupResult.Actions {
				switch action.ActionType {
				case relib.RuleActionSeverityOverride:
					aObj.Severity = action.ActionValueStr
					toPrintActionMap["severity"] = aObj.Severity
				case relib.RuleActionAcknowledge:
					aObj.Acknowledged = true
					aObj.AckTs = time.Now().UTC().Format(time.RFC3339)
					aObj.AutoAck = true
					toPrintActionMap["ack"] = true
				case relib.RuleActionCustomizeRecommendation:
					aObj.IsRuleCustomReco = true
					aObj.RuleCustomRecoStr = strings.Split(action.ActionValueStr, ",")
					toPrintActionMap["customreco"] = aObj.RuleCustomRecoStr
				default:
					l.Warnf("RECORD PROC - unknown action type: %s", action.ActionType)
				}
			}
			l.Debugw("RECORD PROC - rule hit", "matchCriteria", lookupResult.CriteriaHit, "actionsApplied", toPrintActionMap)
		} else {
			// no rule matched
			l.WithField("recId", recordIdentifier(aObj)).Infow("RECORD PROC - no rule hit", "record", convRecord)
		}
	} else {
		l.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - skipped rule lookup")
	}
	return aObj, nil
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
}

func (rh *RuleEngineHandler) distributeRuleTask(res *relib.RuleMsgResult) bool {
	out := &messagebus.Message{
		Topic: rh.config.RuleTasksTopic,
		Value: res.RuleJSON,
	}

	ctx, cancel := context.WithTimeout(rh.ctx, 5*time.Second)
	defer cancel()

	if _, _, err := rh.ruleTaskProducer.Send(ctx, out); err != nil {
		rh.plogger.Errorw("RULE HANDLER - Failed to send rule task message", "error", err)
		return false
	}

	return true
}
