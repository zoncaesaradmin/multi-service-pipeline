package rules

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"servicegomodule/internal/metrics"
	"servicegomodule/internal/models"
	"servicegomodule/internal/tasks"
	eapi "sharedgomodule/extapi"
	"sharedgomodule/logging"
	"sharedgomodule/messagebus"
	"sharedgomodule/utils"
	"strings"
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
	ruleconsumer  messagebus.Consumer
	config        RuleEngineConfig
	plogger       logging.Logger // handle to pipeline logger
	reInst        relib.RuleEngineType
	ctx           context.Context
	cancel        context.CancelFunc
	metricsHelper *metrics.MetricsHelper

	// fields related to background task of applying rule changes to DB records
	rlogger logging.Logger // for both rules msg and rule tasks msg handling

	// severity cache for mapping event titles to severity levels
	severityCache map[string]string

	// link to rule task handler
	ruleTaskHandler *tasks.RuleTasksHandler
}

func NewRuleHandler(config RuleEngineConfig, logger logging.Logger, inputSink chan<- *models.ChannelMessage, metricHelper *metrics.MetricsHelper) *RuleEngineHandler {
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

	rth := tasks.NewRuleTasksHandler(rlogger, config.RuleTasksTopic, config.RuleTasksProdKafkaConfigMap, config.RuleTasksConsKafkaConfigMap, inputSink, metricHelper)
	h := &RuleEngineHandler{
		ruleTaskHandler: rth,
		ruleconsumer:    consumer,
		config:          config,
		plogger:         logger,
		reInst:          reInst,
		rlogger:         rlogger,
		metricsHelper:   metricHelper,
		severityCache:   make(map[string]string),
	}

	logger.Info("Initialized Rule Handler module")
	rlogger.Infow("Initialized Rule Handler", "ruleTopic", config.RulesTopic)
	return h
}

func (rh *RuleEngineHandler) Start() error {
	rh.rlogger.Infow("Starting Rule Engine Handler", "topic", rh.config.RulesTopic)

	// Create context for cancellation
	rh.ctx, rh.cancel = context.WithCancel(context.Background())

	rh.ruleTaskHandler.Start()

	// read all rules already configured from config store
	if err := rh.ReadExistingRules(); err != nil {
		rh.rlogger.Errorw("Failed to load existing rules", "error", err)
	}

	if err := rh.setupRuleConsumer(); err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	for _, siteType := range strings.Split(eapi.AllSiteTypesStr, ",") {
		if err := eapi.FetchSeverityMappings(eapi.EventSvcBaseURL, siteType, rh.severityCache, client); err != nil {
			return fmt.Errorf("Failed to initialize severity cache for %s: %w", siteType, err)
		}
	}

	return nil
}

func (rh *RuleEngineHandler) Stop() error {
	rh.rlogger.Info("Stopping Rule Engine Handler...")

	rh.ruleTaskHandler.Stop()

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

	rh.plogger.Info("Stopped Rule Engine Handler")
	return nil
}

func (rh *RuleEngineHandler) ReadExistingRules() error {
	rh.rlogger.Info("Initializing MongoDB rule loading")

	return nil
}

func (rh *RuleEngineHandler) GetRuleInfo() []byte {
	return rh.reInst.GetAllRuleInfo()
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

	userMeta := relib.TransactionMetadata{TraceId: traceID}

	// For DELETE operations, capture old rule definitions BEFORE rule engine update
	// because HandleRuleEvent will remove the rule from cache
	oldRuleDefinitions, err := rh.getOldRuleDefinitions(message.Value)
	if err != nil {
		msgLogger.Errorw("Failed to get old rule definitions", "error", err.Error())
		// Continue processing even if we can't get old rules
	}

	res, err := rh.reInst.HandleRuleEvent(message.Value, userMeta)
	if err != nil {
		msgLogger.Errorw("RULE HANDLER - Failed to handle rule event", "error", err)
	} else if res != nil && len(res.Rules) > 0 {
		msgLogger.Debugw("RULE HANDLER - Successfully handled rule event", "result", res, "oldRulesCount", len(oldRuleDefinitions))
		rh.processRuleEvent(msgLogger, traceID, res, oldRuleDefinitions)
	}

	if err := rh.ruleconsumer.Commit(context.Background(), message); err != nil {
		msgLogger.Errorw("RULE HANDLER - Failed to commit message", "error", err)
	}
}

// getOldRuleDefinitions extracts rule UUIDs from incoming message and fetches old rule definitions
func (rh *RuleEngineHandler) getOldRuleDefinitions(msgBytes []byte) (map[string]*relib.RuleDefinition, error) {
	var rInput relib.AlertRuleMsg
	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule message: %w", err)
	}

	oldRules := make(map[string]*relib.RuleDefinition)
	for _, alertRule := range rInput.AlertRules {
		if rule, exists := rh.reInst.GetRule(alertRule.UUID); exists {
			oldRules[alertRule.UUID] = &rule
		}
	}
	return oldRules, nil
}

// processRuleEvent handles rule event processing with old rule definitions
func (rh *RuleEngineHandler) processRuleEvent(l logging.Logger, traceID string, res *relib.RuleMsgResult, oldRules map[string]*relib.RuleDefinition) {
	l.Debugw("RULE HANDLER - Converted rule JSON with old rules",
		"ruleEvent", res.RuleEvent,
		"convertedRules", res.Rules,
		"oldRulesCount", len(oldRules))

	if rh.ruleTaskHandler.Leader() {
		operationLabels := map[string]string{
			"event_type":    res.RuleEvent,
			"rule_count":    fmt.Sprintf("%d", len(res.Rules)),
			"trace_id":      traceID,
			"has_old_rules": fmt.Sprintf("%v", len(oldRules) > 0),
		}
		if rh.metricsHelper != nil && rh.metricsHelper.GetCollector() != nil {
			defer rh.metricsHelper.GetCollector().Timer("flow1_rule_event_processing", operationLabels)()
		}

		for _, rule := range res.Rules {
			// rule event can have multiple rules, treat each rule as one task
			// For CREATE operations, proceed even if no old rule (new rule creation)
			// For UPDATE/DELETE operations, only process if we have the old rule
			if res.RuleEvent != relib.RuleEventCreate {
				if _, exists := oldRules[rule.AlertRuleUUID]; !exists {
					l.Debugw("RULE HANDLER - No old rule definition found for task, skipping",
						"ruleEvent", res.RuleEvent,
						"ruleUUID", rule.AlertRuleUUID)
					continue
				}
			}

			// Get old rule if available, or nil for CREATE operations or when not found
			var oldRule *relib.RuleDefinition
			if oldRuleRef, exists := oldRules[rule.AlertRuleUUID]; exists {
				oldRule = oldRuleRef
			}
			if rh.distributeRuleTask(l, traceID, res.RuleEvent, &rule, oldRule) {
				l.Debugw("RULE HANDLER - Successfully distributed rule task with old rules", "ruleEvent", res.RuleEvent)
			}
		}
	} else {
		l.Debug("RULE HANDLER - Not leader, skipping rule task distribution")
	}
}

// distributeRuleTask distributes rule task with old rule definitions included
func (rh *RuleEngineHandler) distributeRuleTask(l logging.Logger, traceID string, eventType string, rule *relib.RuleDefinition, oldRule *relib.RuleDefinition) bool {
	// Check if this rule change actually requires database processing
	if !rh.shouldDistributeTask(eventType, rule) {
		l.Debugw("RULE HANDLER - Skipping task distribution",
			"ruleEvent", eventType,
			"reason", "no database processing required")
		return true // Return true since skipping is successful
	}

	return rh.ruleTaskHandler.DistributeRuleTask(l, traceID, eventType, rule, oldRule)
}

func (rh *RuleEngineHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"status":       "running",
		"rulesTopic":   rh.config.RulesTopic,
		"poll_timeout": rh.config.PollTimeout.String(),
	}
}

func (rh *RuleEngineHandler) ApplyRuleToRecord(l logging.Logger, aObj *alert.Alert, origin models.ChannelMessageOrigin) (*alert.Alert, error) {
	if !eapi.NeedsRuleProcessing(aObj) {
		l.WithField("recId", eapi.RecordIdentifier(aObj)).Infof("RECORD PROC - skipped rule lookup")
		return aObj, nil
	}

	convStartTime := time.Now()
	convRecord := eapi.ConvertAlertObjectToRuleEngineInput(aObj)
	l.WithField("recId", eapi.RecordIdentifier(aObj)).Debugw("RECORD PROC - after conversion", "convertedData", convRecord, "timeTaken", fmt.Sprintf("%v", time.Since(convStartTime)))

	lookupStartTime := time.Now()
	lookupResult := rh.reInst.EvaluateRules(relib.Data(convRecord))
	if rh.metricsHelper != nil {
		rh.metricsHelper.RecordStageLatency(time.Since(lookupStartTime), "rule_lookup")
		rh.metricsHelper.RecordCounter("criteria.count", float64(lookupResult.CritCount), map[string]string{
			"hit": fmt.Sprintf("%v", lookupResult.IsRuleHit),
		})
	}
	lookupTimeTaken := time.Since(lookupStartTime)
	if rh.metricsHelper != nil {
		rh.metricsHelper.RecordStageCompleted(nil, lookupTimeTaken)
	}
	l.WithField("recId", eapi.RecordIdentifier(aObj)).Debugw("RECORD PROC - post lookup", "lookupResult", lookupResult, "timeTaken", fmt.Sprintf("%v", lookupTimeTaken), "origin", origin)

	if lookupResult.IsRuleHit {
		rh.handleRuleHit(l, aObj, lookupResult)
	} else {
		rh.handleRuleMiss(l, aObj, lookupResult, origin)
	}

	return aObj, nil
}

func (rh *RuleEngineHandler) handleRuleHit(l logging.Logger, aObj *alert.Alert, lookupResult relib.RuleLookupResult) {
	aObj.AlertRuleName = lookupResult.RuleName
	toPrintActionMap := make(map[string]string, 0)

	for _, action := range lookupResult.Actions {
		toPrintActionMap[action.ActionType] = action.ActionValueStr
		rh.applyRuleAction(aObj, action.ActionType, action.ActionValueStr)
	}

	l.Debugw("RECORD PROC - rule hit", "matchCriteria", lookupResult.CriteriaHit, "actionsApplied", toPrintActionMap)
}

func (rh *RuleEngineHandler) handleRuleMiss(l logging.Logger, aObj *alert.Alert, lookupResult relib.RuleLookupResult, origin models.ChannelMessageOrigin) {
	toPrintActionMap := make(map[string]string, 0)

	if origin == models.ChannelMessageOriginDb {
		rh.revertToDefaultActions(aObj, lookupResult)
	}

	l.WithField("recId", eapi.RecordIdentifier(aObj)).Infow("RECORD PROC - no rule hit", "actionsApplied", toPrintActionMap)
}

func (rh *RuleEngineHandler) revertToDefaultActions(aObj *alert.Alert, lookupResult relib.RuleLookupResult) {
	allSupportedActions := []string{
		relib.RuleActionAcknowledge,
		relib.RuleActionSeverityOverride,
		relib.RuleActionCustomizeRecommendation,
	}

	for _, sAction := range allSupportedActions {
		if contains, actionValue := containsActionType(lookupResult.Actions, sAction); contains {
			rh.applyRuleAction(aObj, sAction, actionValue)
		} else {
			rh.applyDefaultAction(aObj, sAction)
		}
	}
}

func (rh *RuleEngineHandler) applyRuleAction(aObj *alert.Alert, actionType string, actionValue string) {
	switch actionType {
	case relib.RuleActionSeverityOverride:
		aObj.Severity = actionValue
	case relib.RuleActionAcknowledge:
		aObj.Acknowledged = true
		aObj.AckTs = utils.GetCurrentUTCTimestampMilli()
		aObj.AutoAck = true
	case relib.RuleActionCustomizeRecommendation:
		aObj.IsRuleCustomReco = true
		aObj.RuleCustomRecoStr = strings.Split(actionValue, ",")
	default:
	}
}

func (rh *RuleEngineHandler) applyDefaultAction(aObj *alert.Alert, actionType string) {
	switch actionType {
	case relib.RuleActionSeverityOverride:
		// fetch default severity from mapping and apply
		key := fmt.Sprintf("%s:%s", aObj.Vendor, aObj.MnemonicTitle)
		severity, exists := rh.severityCache[key]
		if exists {
			severity = relib.NormalizeSeverity(severity)
			aObj.Severity = severity
		}
	case relib.RuleActionAcknowledge:
		aObj.Acknowledged = false
		aObj.AckTs = utils.GetCurrentUTCTimestampMilli()
		aObj.AutoAck = false
	case relib.RuleActionCustomizeRecommendation:
		aObj.IsRuleCustomReco = false
		aObj.RuleCustomRecoStr = []string{}
		aObj.RuleCustomRecoAction = ""
	default:
	}
	aObj.AlertRuleName = ""
}

// shouldDistributeTask determines if a rule change requires database processing
// Based on action type and applyToExisting flag in the rule actions
func (rh *RuleEngineHandler) shouldDistributeTask(eventType string, rule *relib.RuleDefinition) bool {
	// Always distribute DELETE operations - they might need cleanup
	if eventType == relib.RuleEventDelete {
		rh.rlogger.Debugw("RULE HANDLER - Distributing DELETE operation", "ruleEvent", eventType)
		return true
	}

	// For CREATE and UPDATE operations, check if any rule has applyToExisting=true
	if eventType == relib.RuleEventCreate || eventType == relib.RuleEventUpdate {
		if !rule.ApplyActionsToAll {
			rh.rlogger.Debugw("RULE HANDLER - No applyToExisting=true found, skipping distribution",
				"ruleEvent", eventType)
			return false
		}

		rh.rlogger.Debugw("RULE HANDLER - Found applyToExisting=true, distributing task",
			"ruleEvent", eventType)
		return true
	}

	rh.rlogger.Warnw("RULE HANDLER - unknown rule event, distributing anyway", "ruleEvent", eventType)
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
