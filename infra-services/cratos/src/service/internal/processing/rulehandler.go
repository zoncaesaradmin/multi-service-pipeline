package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"sync"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
	"time"
)

type RuleEngineConfig struct {
	RulesTopic     string
	PollTimeout    time.Duration
	Logging        logging.LoggerConfig
	KafkaConfigMap map[string]any
	RuleTaskTopic  string
}

type RuleEngineHandler struct {
	ruleconsumer messagebus.Consumer
	config       RuleEngineConfig
	logger       logging.Logger
	reInst       relib.RuleEngineType
	ctx          context.Context
	cancel       context.CancelFunc

	// fields related to background task of applying rule changes to DB records
	ruleTaskProducer messagebus.Producer
	ruleTaskConsumer messagebus.Consumer
	isLeader         bool
	leaderMutex      sync.RWMutex
}

func NewRuleHandler(config RuleEngineConfig, logger logging.Logger) *RuleEngineHandler {
	// use simple filename - path resolution is handled by messagebus config loader
	consumer := messagebus.NewConsumer(config.KafkaConfigMap, "ruleConsGroup"+utils.GetEnv("HOSTNAME", ""))
	filePath := config.Logging.FilePath
	if filePath == "" {
		filePath = "/tmp/test.log"
	}
	lInfo := relib.LoggerInfo{
		ServiceName: config.Logging.ServiceName,
		FilePath:    filePath,
		Level:       config.Logging.Level.String(),
	}
	reInst := relib.CreateRuleEngineInstance(lInfo, []string{relib.RuleTypeMgmt})

	h := &RuleEngineHandler{
		ruleconsumer: consumer,
		config:       config,
		logger:       logger,
		reInst:       reInst,
		isLeader:     false,
	}

	// TODO: derive topic name from config file like other topics
	if h.config.RuleTaskTopic == "" {
		h.config.RuleTaskTopic = "cisco_nir-ruletasks"
	}

	logger.Infow("Initialized Rule Engine Handler", "ruleTopic", config.RulesTopic, "ruleTaskTopic", h.config.RuleTaskTopic)
	return h
}

func (rh *RuleEngineHandler) Start() error {
	rh.logger.Infow("Starting Rule Engine Handler", "topic", rh.config.RulesTopic)

	// Create context for cancellation
	rh.ctx, rh.cancel = context.WithCancel(context.Background())

	// Initialize producer for distributing rule tasks
	// TODO: should get explicit kafka conf file instead of using consumer's conf file
	rh.ruleTaskProducer = messagebus.NewProducer(rh.config.KafkaConfigMap)

	// Initialize rule task consumer with shared group for task distribution
	ruleTaskGroup := "ruleTaskConsGroup-shared"
	rh.ruleTaskConsumer = messagebus.NewConsumer(rh.config.KafkaConfigMap, ruleTaskGroup)

	rh.ruleTaskConsumer.OnMessage(func(message *messagebus.Message) {
		if message == nil {
			return
		}

		rh.logger.Debugw("RULE TASK HANDLER - Received task", "size", len(message.Value))

		// Process rule task
		// TODO: simplify by using data unmarshal into struct and get the action below
		sendToDBBatchProcessor(rh.ctx, rh.logger, message.Value, "RULE_TASK")

		if err := rh.ruleTaskConsumer.Commit(context.Background(), message); err != nil {
			rh.logger.Errorw("RULE TASK HANDLER - Failed to commit message", "error", err)
		}
	})

	if err := rh.ruleTaskConsumer.Subscribe([]string{rh.config.RuleTaskTopic}); err != nil {
		rh.logger.Errorw("RULE TASK HANDLER - Failed to subscribe to rule task topic", "error", err)
		return fmt.Errorf("failed to subscribe to rule task topic: %w", err)
	}

	rh.ruleconsumer.OnAssign(func(assignments []messagebus.PartitionAssignment) {
		rh.logger.Infow("RULE HANDLER - Assigned partitions", "partitions", assignments)
		for _, assignment := range assignments {
			if assignment.Partition == 0 {
				// select self as leader to generate tasks
				rh.SetLeader(true)
				break
			}
		}
	})

	rh.ruleconsumer.OnRevoke(func(revoked []messagebus.PartitionAssignment) {
		rh.logger.Infow("RULE HANDLER - Revoked partitions", "partitions", revoked)
		for _, r := range revoked {
			if r.Partition == 0 {
				// relinquish leadership
				rh.SetLeader(false)
				break
			}
		}
	})

	// Register OnMessage callback for main rules topic
	rh.ruleconsumer.OnMessage(func(message *messagebus.Message) {
		if message != nil {
			rh.logger.Debugw("RULE HANDLER - Received message", "size", len(message.Value))

			res, err := rh.reInst.HandleRuleEvent(message.Value)
			if err != nil {
				rh.logger.Errorw("RULE HANDLER - Failed to handle rule event", "error", err)
			} else if res != nil && len(res.RuleJSON) > 0 {
				rh.logger.Debugw("RULE HANDLER - Converted rule JSON", "action", res.Action, "size", len(res.RuleJSON))

				// Only leader distributes rule tasks
				if rh.IsLeader() {
					if rh.distributeRuleTask(res) {
						rh.logger.Debugw("RULE HANDLER - Successfully distributed rule task", "action", res.Action)
					}
				} else {
					rh.logger.Debug("RULE HANDLER - Not leader, skipping rule task distribution")
				}
			}

			// Commit the message
			if err := rh.ruleconsumer.Commit(context.Background(), message); err != nil {
				rh.logger.Errorw("RULE HANDLER - Failed to commit message", "error", err)
			}
		}
	})

	// Subscribe to topics
	if err := rh.ruleconsumer.Subscribe([]string{rh.config.RulesTopic}); err != nil {
		rh.logger.Errorw("Failed to subscribe to rules topic", "error", err)
		return fmt.Errorf("failed to subscribe to rules topic: %w", err)
	}

	return nil
}

func (rh *RuleEngineHandler) Stop() error {
	rh.logger.Info("RULE HANDLER - Stopping Rule Engine Handler")

	if rh.cancel != nil {
		rh.cancel()
	}

	// Close all consuers and producer

	if rh.ruleconsumer != nil {
		if err := rh.ruleconsumer.Close(); err != nil {
			rh.logger.Errorw("Failed to close consumer", "error", err)
			return err
		}
	}

	if rh.ruleTaskConsumer != nil {
		if err := rh.ruleTaskConsumer.Close(); err != nil {
			rh.logger.Errorw("RULE HANDLER - Failed to close rule task consumer", "error", err)
			return err
		}
	}

	if rh.ruleTaskProducer != nil {
		if err := rh.ruleTaskProducer.Close(); err != nil {
			rh.logger.Errorw("RULE HANDLER - Failed to close rule task producer", "error", err)
			return err
		}
	}

	rh.logger.Info("RULE HANDLER - Stopped Rule Engine Handler")
	return nil
}

func sendToDBBatchProcessor(ctx context.Context, logger logging.Logger, ruleBytes []byte, action string) {
	logger.Infow("DB_BATCH - sending rule to DB batch processor", "action", action, "size", len(ruleBytes))
}

func (rh *RuleEngineHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":        "running",
		"ruleTopic":     rh.config.RulesTopic,
		"ruleTaskTopic": rh.config.RuleTaskTopic,
		"isLeader":      rh.IsLeader(),
		"poll_timeout":  rh.config.PollTimeout.String(),
	}
}

func (rh *RuleEngineHandler) applyRuleToRecord(aObj *alert.Alert) (*alert.Alert, error) {
	if needsRuleProcessing(aObj) {
		convRecord := ConvertAlertObjectToRuleEngineInput(aObj)
		rh.logger.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - converted data: %v", convRecord)
		ruleHit, ruleUuid, evalResults := rh.reInst.EvaluateRules(relib.Data(convRecord))
		if ruleHit {
			rh.logger.Infof("RECORD PROC - rule hit for record %s, rule UUID: %s, eval results: %v", recordIdentifier(aObj), ruleUuid, evalResults)
			// rule matched
			aObj.RuleId = ruleUuid
			for _, action := range evalResults.Actions {
				rh.logger.Infof("RECORD PROC - action type: %s", action.Type)
				if action.Type == "severity" {
					actionPayload, err := json.Marshal(action.Payload)
					if err != nil {
						rh.logger.Errorw("RECORD PROC - Failed to marshal action payload", "error", err)
						continue
					}
					type ActionSeverity struct {
						SeverityValue string `json:"severityValue,omitempty"`
					}
					var sact ActionSeverity
					err = json.Unmarshal(actionPayload, &sact)
					if err != nil {
						rh.logger.Errorw("RECORD PROC - Failed to unmarshal action payload", "error", err)
						return aObj, err
					}
					aObj.Severity = sact.SeverityValue
				} else if action.Type == "ACKNOWLEDGE" {
					aObj.Acknowledged = true
					aObj.AckTs = time.Now().UTC().Format(time.RFC3339)
					aObj.AutoAck = true
				} else if action.Type == "CUSTOMIZE_ANOMALY" {
					aObj.IsRuleCustomReco = true
					actionPayload, err := json.Marshal(action.Payload)
					if err != nil {
						continue
					}
					type ActionCustomReco struct {
						ApplyToExisting bool     `json:"applyToExisting"`
						CustomMessage   []string `json:"customMessage"`
					}
					var cact ActionCustomReco
					err = json.Unmarshal(actionPayload, &cact)
					if err != nil {
						continue
					}
					aObj.RuleCustomRecoStr = cact.CustomMessage
				}
			}
		} else {
			// no rule matched
			rh.logger.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - no rule hit")
		}
	} else {
		rh.logger.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - skipped rule lookup")
	}
	return aObj, nil
}

// return current leadership state
func (rh *RuleEngineHandler) IsLeader() bool {
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
		Topic: rh.config.RuleTaskTopic,
		Value: res.RuleJSON,
	}

	ctx, cancel := context.WithTimeout(rh.ctx, 5*time.Second)
	defer cancel()

	if _, _, err := rh.ruleTaskProducer.Send(ctx, out); err != nil {
		rh.logger.Errorw("RULE HANDLER - Failed to send rule task message", "error", err)
		return false
	}

	return true
}
