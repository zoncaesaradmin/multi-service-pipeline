package rules

import (
	"context"
	"crypto/tls"
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
	rlogger logging.Logger // for both rules msg and rule tasks msg handling

	// severity cache for mappping event titles to severity levels
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

	if rh.ruleTaskHandler.Leader() {
		if rh.distributeRuleTask(l, traceID, res) {
			l.Debugw("RULE HANDLER - Successfully distributed rule task", "action", res.Action)
		}
	} else {
		l.Debug("RULE HANDLER - Not leader, skipping rule task distribution")
	}
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
	l.WithField("recId", eapi.RecordIdentifier(aObj)).Debugw("RECORD PROC - post lookup", "lookupResult", lookupResult, "timeTaken", fmt.Sprintf("%v", lookupTimeTaken))

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

func (rh *RuleEngineHandler) distributeRuleTask(l logging.Logger, traceID string, res *relib.RuleMsgResult) bool {
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
