package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
	"time"
)

type RuleEngineConfig struct {
	RulesTopic  string
	PollTimeout time.Duration
	Logging     logging.LoggerConfig
}

type RuleEngineHandler struct {
	consumer messagebus.Consumer
	config   RuleEngineConfig
	logger   logging.Logger
	reInst   *relib.RuleEngine
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewRuleHandler(config RuleEngineConfig, logger logging.Logger) *RuleEngineHandler {
	// use simple filename - path resolution is handled by messagebus config loader
	consumer := messagebus.NewConsumer("kafka-consumer.yaml", "ruleConsGroup"+utils.GetEnv("HOSTNAME", ""))
	lInfo := relib.LoggerInfo{
		ServiceName: config.Logging.ServiceName,
		FilePath:    config.Logging.FilePath,
		Level:       config.Logging.Level.String(),
	}
	reInst := relib.CreateRuleEngineInstance(lInfo, []string{relib.RuleTypeMgmt})

	return &RuleEngineHandler{
		consumer: consumer,
		config:   config,
		logger:   logger,
		reInst:   reInst,
	}
}

func (rh *RuleEngineHandler) Start() error {
	rh.logger.Infow("Starting Rule Engine Handler", "topic", rh.config.RulesTopic)

	// Create context for cancellation
	rh.ctx, rh.cancel = context.WithCancel(context.Background())

	// Subscribe to topics
	if err := rh.consumer.Subscribe([]string{rh.config.RulesTopic}); err != nil {
		rh.logger.Errorw("Failed to subscribe to rules topic", "error", err)
		return fmt.Errorf("failed to subscribe to rules topic: %w", err)
	}

	// Start consuming in a goroutine
	go rh.consumeLoop()

	return nil
}

func (rh *RuleEngineHandler) Stop() error {
	rh.logger.Info("Stopping Rule Engine Handler")

	if rh.cancel != nil {
		rh.cancel()
	}

	if rh.consumer != nil {
		if err := rh.consumer.Close(); err != nil {
			rh.logger.Errorw("Failed to close consumer", "error", err)
			return err
		}
	}

	return nil
}

// consumeLoop continuously polls for rule messages
func (rh *RuleEngineHandler) consumeLoop() {
	defer func() {
		if r := recover(); r != nil {
			rh.logger.Errorw("RULE HANDLER - panic recovered", "panic", r)
		}
	}()

	for {
		select {
		case <-rh.ctx.Done():
			rh.logger.Info("RULE HANDLER - consume loop stopped")
			return
		default:
			// Poll for messages
			message, err := rh.consumer.Poll(rh.config.PollTimeout)
			if err != nil {
				rh.logger.Warnw("RULE HANDLER - Error polling for messages", "error", err)
				continue
			}

			if message != nil {
				rh.logger.Debugw("RULE HANDLER - Received message", "size", len(message.Value))

				err := rh.reInst.HandleRuleEvent(message.Value)
				if err != nil {
					rh.logger.Errorw("RULE HANDLER - Failed to handle rule event", "error", err)
				}

				// Commit the message
				if err := rh.consumer.Commit(context.Background(), message); err != nil {
					rh.logger.Errorw("RULE HANDLER - Failed to commit message", "error", err)
				}
			}
		}
	}
}

func (rh *RuleEngineHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":       "running",
		"topic":        rh.config.RulesTopic,
		"poll_timeout": rh.config.PollTimeout.String(),
	}
}

func (rh *RuleEngineHandler) applyRuleToRecord(aObj *alert.Alert) (*alert.Alert, error) {
	if needsRuleProcessing(aObj) {
		convRecord := ConvertAlertObjectToRuleEngineInput(aObj)
		rh.logger.WithField("recId", recordIdentifier(aObj)).Infof("RECORD PROC - converted data: %v", convRecord)
		ruleHit, ruleUuid, evalResults := rh.reInst.EvaluateRules(convRecord)
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
					aObj.IsCustomReco = true
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
					aObj.CustomRecoStr = cact.CustomMessage
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
