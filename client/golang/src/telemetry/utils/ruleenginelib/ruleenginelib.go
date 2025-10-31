package ruleenginelib

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Constants for rule engine match keys
const (
	MatchKeyFabricName = "fabricName"
	MatchKeyCategory   = "category"
	MatchKeyTitle      = "title"
	MatchKeySeverity   = "severity"
	MatchKeySwitch     = "switch"
	MatchKeyInterface  = "interface"
	MatchKeyIp         = "ip"
	MatchKeyVrf        = "vrf"
	MatchKeyVni        = "vni"
	MatchKeySubnet     = "subnet"
	MatchKeyTenant     = "tenant"
	MatchKeyVpc        = "vpc"
	MatchKeyRoute      = "route"
)

// Constants for rule types
const (
	RuleTypeMgmt      = "ALERT_RULES"
	RuleTypeSource    = "GLOBAL_CATEGORY_RULES"
	ScopeSystem       = "system"
	PrimaryKeyDefault = "PRIMARY_KEY_DEFAULT"
	PrimaryKeySystem  = "PRIMARY_KEY_SYSTEM"
)

// External event types - these are the actual message types received from config service
const (
	RuleEventCreate  = "CREATE_ALERT_RULE"
	RuleEventUpdate  = "UPDATE_ALERT_RULE"
	RuleEventDelete  = "DELETE_ALERT_RULE"
	RuleEventEnable  = "ENABLE_ALERT_RULE"
	RuleEventDisable = "DISABLE_ALERT_RULE"
)

// Internal event types - these are the 3 core operations the rule engine supports
const (
	InternalEventCreate = "CREATE_RULE"
	InternalEventUpdate = "UPDATE_RULE"
	InternalEventDelete = "DELETE_RULE"
)

// action types supported and used in RuleAction.ActionType
const (
	RuleActionAcknowledge             = "ACKNOWLEDGE"
	RuleActionSeverityOverride        = "OVERRIDE_SEVERITY"
	RuleActionCustomizeRecommendation = "CUSTOMIZE_RECOMMENDATION"
)

// severity types supported
const (
	SeverityCritical = "critical"
	SeverityMajor    = "major"
	SeverityMinor    = "minor"
	SeverityWarning  = "warning"
	SeverityDefault  = "none"
)

// external type for rule lookup result
type RuleLookupResult struct {
	IsRuleHit bool `json:"isRuleHit"`
	// below are relevant only if IsRuleHit is true
	RuleUUID           string          `json:"ruleUUID"`
	RuleName           string          `json:"ruleName"`
	CriteriaHit        RuleHitCriteria `json:"criteria"`
	Actions            []RuleHitAction `json:"actions"`
	CritCount          int             `json:"criteriaCount"`
	ApplyToAllExisting bool            `json:"applyToAllExisting"`
}

// external type for rule lookup result actions
type RuleHitAction struct {
	ActionType     string `json:"actionType"`
	ActionValueStr string `json:"actionValueStr"`
}

// external type for rule lookup result match condition
type RuleHitCriteria struct {
	Any []AstConditional `json:"any"`
	All []AstConditional `json:"all"`
}

// RuleMsgResult encapsulates the outcome of processing a rule message.
// Action always reflects the incoming message action (even on errors after
// successful unmarshal). RuleJSON will contain the normalized rule bytes
// ready for the rule engine when applicable (create / update with non-empty
// payload). It is nil for delete operations (rule UUID is still carried in
// the payload) or when the action is invalid / irrelevant
type RuleMsgResult struct {
	RuleEvent  string            `json:"ruleEvent"`  // Original message event type
	Rules      []RuleDefinition  `json:"rules"`      // Processed rules
	RuleEvents map[string]string `json:"ruleEvents"` // Per-rule event types: ruleUUID -> eventType
}

type TransactionMetadata struct {
	TraceId string
}

type LoggerInfo struct {
	ServiceName string
	FilePath    string
	Level       string
}

// CreateRuleEngineInstance creates a new RuleEngine instance with rule type filtering
func CreateRuleEngineInstance(lInfo LoggerInfo, ruleTypes []string) *RuleEngine {
	logger := CreateLoggerInstance(lInfo.ServiceName, lInfo.FilePath, zerologLevel(lInfo.Level))
	nre := NewRuleEngineInstance(nil, logger)
	nre.RuleTypes = ruleTypes

	go nre.initAlertRules()

	return nre
}

func isValidEventType(msgType string) bool {
	switch msgType {
	case RuleEventCreate, RuleEventUpdate, RuleEventDelete, RuleEventEnable, RuleEventDisable:
		return true
	default:
		return false
	}
}

func isRuleTypeOfInterest(rType string, validTypes []string) bool {
	for _, vType := range validTypes {
		if vType == rType {
			return true
		}
	}
	return false
}

func isRelevantRule(meta AlertRuleMetadata, validTypes []string) bool {
	if !isRuleTypeOfInterest(meta.RuleType, validTypes) {
		return false
	}
	if !isValidEventType(meta.RuleEventType) {
		return false
	}
	return true
}

func (re *RuleEngine) HandleAlertRuleEvent(msgBytes []byte, userMeta TransactionMetadata) (*RuleMsgResult, map[string]*RuleDefinition, error) {
	var rInput AlertRuleMsg
	trLogger := re.Logger.WithField("traceId", userMeta.TraceId)
	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		trLogger.Infof("Failed to unmarshal rule message: %v", err.Error())
		// log and ignore invalid messages
		return nil, nil, err
	}

	trLogger.Debugf("RELIB - received msg unmarshalled: %+v", rInput)

	originalMsgType := rInput.Metadata.RuleEventType
	if !isRelevantRule(rInput.Metadata, re.RuleTypes) {
		trLogger.Debug("RELIB - ignore invalid/non-relevant rule")
		return &RuleMsgResult{RuleEvent: originalMsgType}, nil, nil
	}

	// For DELETE operations, capture old rule definitions BEFORE rule engine update
	// because HandleAlertRuleEvent will remove the rule from cache
	oldRules := make(map[string]*RuleDefinition)
	for _, alertRule := range rInput.AlertRules {
		if rule, exists := re.GetRule(alertRule.UUID); exists {
			oldRules[alertRule.UUID] = &rule
		}
	}

	trLogger.Infof("RELIB - processing valid rule for operation: %s", originalMsgType)

	// Process each payload item individually based on state
	var allProcessedRules []RuleDefinition
	ruleEvents := make(map[string]string)
	processedCount := 0

	for i, alertRule := range rInput.AlertRules {
		// Determine effective event type based on original event and state
		effectiveEventType, shouldProcess := re.determineEffectiveEventType(trLogger, originalMsgType, alertRule, i)

		if !shouldProcess {
			continue // Skip this payload item (e.g., CREATE with state=false)
		}

		// Create individual message for this payload item
		singleRuleInput := AlertRuleMsg{
			Metadata:   rInput.Metadata,
			AlertRules: []AlertRuleConfig{alertRule},
		}

		// Update metadata to reflect effective event type
		singleRuleInput.Metadata.RuleEventType = effectiveEventType

		// Process this individual rule
		rule, err := re.processSingleRule(trLogger, singleRuleInput, effectiveEventType)
		if err != nil {
			trLogger.Infof("RELIB - failed to process rule %d: %v", i, err)
			continue
		}

		allProcessedRules = append(allProcessedRules, rule)

		// Track event type per rule UUID
		ruleEvents[rule.AlertRuleUUID] = effectiveEventType

		processedCount++
	}

	if processedCount == 0 {
		trLogger.Info("RELIB - no rules processed (all dropped or failed)")
		return nil, oldRules, fmt.Errorf("no rules processed")
	}

	result := &RuleMsgResult{
		RuleEvent:  originalMsgType, // Original message event type
		Rules:      allProcessedRules,
		RuleEvents: ruleEvents, // Per-rule event types
	}

	trLogger.Infof("RELIB - successfully processed %d rules with original event type: %s", processedCount, originalMsgType)
	return result, oldRules, nil
}

// determineEffectiveEventType determines what event type to use based on original event and state
func (re *RuleEngine) determineEffectiveEventType(logger *Logger, originalEventType string, alertRule AlertRuleConfig, index int) (string, bool) {
	// Extract state from the alert rule - need to handle the state field properly
	// Since your JSON shows boolean, but struct has string, we need to parse appropriately
	ruleState := re.extractRuleState(alertRule)

	logger.Debugf("RELIB - rule %d: originalEvent=%s, state=%v", index, originalEventType, ruleState)

	switch originalEventType {
	case RuleEventCreate:
		if !ruleState {
			logger.Infof("RELIB - dropping CREATE rule %d (state=false)", index)
			return "", false // Drop CREATE with state=false
		}
		logger.Debugf("RELIB - processing CREATE rule %d (state=true)", index)
		return InternalEventCreate, true

	case RuleEventUpdate:
		if !ruleState {
			logger.Infof("RELIB - converting UPDATE rule %d to DELETE (state=false)", index)
			return InternalEventDelete, true // Convert UPDATE with state=false to DELETE
		}
		// For re-enablement scenario: Check if rule exists in engine
		if _, exists := re.GetRule(alertRule.UUID); exists {
			logger.Debugf("RELIB - processing UPDATE rule %d (state=true, rule exists)", index)
			return InternalEventUpdate, true // Rule exists, normal update
		} else {
			logger.Infof("RELIB - converting UPDATE rule %d to CREATE (state=true, rule missing - re-enablement)", index)
			return InternalEventCreate, true // Rule missing, treat as create for re-enablement
		}

	case RuleEventEnable:
		// ENABLE always converts to CREATE (simple rule: enable = add to engine)
		logger.Debugf("RELIB - converting ENABLE rule %d to CREATE", index)
		return InternalEventCreate, true

	case RuleEventDisable:
		// DISABLE always converts to DELETE (simple rule: disable = remove from engine)
		logger.Debugf("RELIB - converting DISABLE rule %d to DELETE", index)
		return InternalEventDelete, true

	case RuleEventDelete:
		logger.Debugf("RELIB - processing DELETE rule %d", index)
		return InternalEventDelete, true // DELETE events are always processed

	default:
		logger.Infof("RELIB - unknonwn event type %s for rule %d", originalEventType, index)
		return "", false
	}
}

// extractRuleState extracts the boolean state from AlertRuleConfig
func (re *RuleEngine) extractRuleState(alertRule AlertRuleConfig) bool {
	return strings.ToLower(alertRule.State) == "true"
}

// processSingleRule processes a single rule with the determined event type
func (re *RuleEngine) processSingleRule(logger *Logger, ruleInput AlertRuleMsg, eventType string) (RuleDefinition, error) {
	ruleJsonBytes, err := ConvertToRuleEngineFormat(ruleInput.AlertRules)
	if err != nil {
		return RuleDefinition{}, fmt.Errorf("failed to convert rule format: %w", err)
	}

	if len(ruleJsonBytes) == 0 {
		return RuleDefinition{}, fmt.Errorf("empty rule after conversion")
	}

	logger.Debugf("RELIB - processing single rule with event type %s: %s", eventType, string(ruleJsonBytes))
	handledRules, err := re.handleRuleMsgEvents(logger, ruleJsonBytes, eventType)
	if len(handledRules) >= 0 {
		return handledRules[0], nil
	}
	return RuleDefinition{}, err
}

func (re *RuleEngine) handleRuleMsgEvents(l *Logger, rmsg []byte, msgType string) ([]RuleDefinition, error) {
	l.Debugf("RELIB - handleRuleMsgEvents called with msgType: %s, ruleData: %s", msgType, string(rmsg))
	switch msgType {
	case InternalEventCreate:
		// Handle create rule event
		l.Debugf("RELIB - handling internal create rule msg event")
		return re.AddRule(string(rmsg))
	case InternalEventUpdate:
		// Handle update rule event
		l.Debugf("RELIB - handling internal update rule msg event")
		return re.AddRule(string(rmsg))
	case InternalEventDelete:
		// Handle delete rule event
		l.Debugf("RELIB - handling internal delete rule msg event")
		return re.DeleteRule(string(rmsg))
	}
	l.Infof("RELIB - unknown internal rule event type: %s", msgType)
	return []RuleDefinition{}, errors.New("unknown internal rule event type")
}

// initAlertRules initializes the rules from already existing rules configured
func (re *RuleEngine) initAlertRules() {
	rules := GetAllConfiguredAlertRules()
	for _, rule := range rules {
		re.AddRule(string(rule))
	}
}

// ParseRuleDefinitions parses the rule JSON into structured RuleDefinition types
func ParseRuleDefinitions(ruleBytes []byte) ([]*RuleDefinition, error) {
	var rules []*RuleDefinition
	if err := json.Unmarshal(ruleBytes, &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules into structured format: %w", err)
	}
	return rules, nil
}

// ExtractConditions extracts conditions from matchCriteriaEntries
func ExtractConditions(rule *RuleDefinition) []map[string]interface{} {
	var conditions []map[string]interface{}
	if rule == nil || rule.MatchCriteriaEntries == nil {
		return conditions
	}

	for _, criteriaList := range rule.MatchCriteriaEntries {
		for _, criteria := range criteriaList {
			if criteria.Condition.All != nil || criteria.Condition.Any != nil {
				// Convert AstCondition to map[string]interface{}
				conditionMap := make(map[string]interface{})
				if len(criteria.Condition.All) > 0 {
					conditionMap["all"] = criteria.Condition.All
				}
				if len(criteria.Condition.Any) > 0 {
					conditionMap["any"] = criteria.Condition.Any
				}
				conditions = append(conditions, conditionMap)
			}
		}
	}
	return conditions
}

// RuleChangeRequest represents a request to process rule changes
type RuleChangeRequest struct {
	RuleEvent       string
	NewRule         *RuleDefinition
	OldRule         *RuleDefinition
	ApplyToExisting bool
	AlertRuleUUID   string
}

// RuleChangeResponse represents the response from processing rule changes
type RuleChangeResponse struct {
	ShouldProcess    bool
	Conditions       []map[string]interface{}
	NeedsCleanup     bool
	ProcessingReason string
}

// ProcessRuleChangeRequest processes a rule change request and returns what conditions to query
func ProcessRuleChangeRequest(req RuleChangeRequest) RuleChangeResponse {
	response := RuleChangeResponse{
		ShouldProcess: false,
		Conditions:    []map[string]interface{}{},
		NeedsCleanup:  false,
	}

	switch req.RuleEvent {
	case InternalEventDelete:
		return processDeleteRequest(req)
	case InternalEventUpdate:
		return processUpdateRequest(req)
	case InternalEventCreate:
		return processCreateRequest(req)
	default:
		response.ProcessingReason = fmt.Sprintf("Unknown rule event: %s", req.RuleEvent)
		return response
	}
}

// processDeleteRequest handles DELETE rule events
func processDeleteRequest(req RuleChangeRequest) RuleChangeResponse {
	// DELETE always processes existing records to clean up rule effects
	response := RuleChangeResponse{ShouldProcess: true, Conditions: []map[string]interface{}{}, NeedsCleanup: false}

	response.ProcessingReason = "DELETE rule - always process existing anomalies to clean up rule effects"
	if req.NewRule != nil {
		response.Conditions = ExtractConditions(req.NewRule)
	}
	return response
}

// processCreateRequest handles CREATE rule events
func processCreateRequest(req RuleChangeRequest) RuleChangeResponse {
	response := RuleChangeResponse{ShouldProcess: false, Conditions: []map[string]interface{}{}, NeedsCleanup: false}

	if !req.ApplyToExisting {
		response.ProcessingReason = "CREATE rule - skipping existing anomalies (applyToExisting=false)"
		return response
	}

	response.ShouldProcess = true
	response.ProcessingReason = "CREATE rule - processing existing anomalies"
	if req.NewRule != nil {
		response.Conditions = ExtractConditions(req.NewRule)
	}
	return response
}

// processUpdateRequest handles UPDATE rule events
func processUpdateRequest(req RuleChangeRequest) RuleChangeResponse {
	// Check ApplyActionsToAll transitions
	return processApplyActionsTransition(req)
}

// processApplyActionsTransition handles ApplyActionsToAll transitions for enabled rules
func processApplyActionsTransition(req RuleChangeRequest) RuleChangeResponse {
	response := RuleChangeResponse{ShouldProcess: false, Conditions: []map[string]interface{}{}, NeedsCleanup: false}

	// Check for ApplyActionsToAll transition from true to false (for enabled rules)
	needsCleanup := false
	if req.OldRule != nil && req.OldRule.ApplyActionsToAll && !req.ApplyToExisting {
		needsCleanup = true
		response.NeedsCleanup = true
	}

	if !req.ApplyToExisting && !needsCleanup {
		response.ProcessingReason = "UPDATE rule - skipping existing anomalies (applyToExisting=false, no cleanup needed)"
		return response
	}

	response.ShouldProcess = true

	// Extract conditions from both old and new rules
	var allConditions []map[string]interface{}
	if req.NewRule != nil {
		allConditions = append(allConditions, ExtractConditions(req.NewRule)...)
	}

	// Add old conditions if they exist and are different
	if req.OldRule != nil {
		allConditions = mergeUniqueConditions(allConditions, ExtractConditions(req.OldRule))
	}

	response.Conditions = allConditions
	if needsCleanup {
		response.ProcessingReason = "UPDATE rule - ApplyActionsToAll changed from true to false, processing for cleanup"
	} else {
		response.ProcessingReason = "UPDATE rule - processing existing anomalies"
	}

	return response
}

// mergeUniqueConditions merges old conditions into new conditions, avoiding duplicates
func mergeUniqueConditions(newConditions, oldConditions []map[string]interface{}) []map[string]interface{} {
	for _, oldCond := range oldConditions {
		isDuplicate := false
		for _, newCond := range newConditions {
			if DeepEqualMaps(oldCond, newCond) {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			newConditions = append(newConditions, oldCond)
		}
	}
	return newConditions
}

// DeepEqualMaps compares two maps for deep equality using reflect for better performance
func DeepEqualMaps(map1, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, val1 := range map1 {
		val2, exists := map2[key]
		if !exists {
			return false
		}

		// Use reflect.DeepEqual instead of JSON marshaling for better performance
		if !reflect.DeepEqual(val1, val2) {
			return false
		}
	}

	return true
}

// ExtractValueFromConditions extracts a value for a specific identifier from rule conditions
func ExtractValueFromConditions(conditions []map[string]interface{}, identifier string) interface{} {
	for _, condition := range conditions {
		if value := extractFromConditionArray(condition, "all", identifier); value != nil {
			return value
		}
		if value := extractFromConditionArray(condition, "any", identifier); value != nil {
			return value
		}
	}
	return nil
}

// extractFromConditionArray extracts value from a specific condition array (all/any)
func extractFromConditionArray(condition map[string]interface{}, arrayKey, identifier string) interface{} {
	if arrayValues, exists := condition[arrayKey]; exists {
		if astSlice, ok := arrayValues.([]AstConditional); ok {
			for _, astCond := range astSlice {
				if astCond.Identifier == identifier {
					return astCond.Value
				}
			}
		}
	}
	return nil
}

type RuleEngineType interface {
	HandleAlertRuleEvent([]byte, TransactionMetadata) (*RuleMsgResult, map[string]*RuleDefinition, error)
	EvaluateRules(Data) RuleLookupResult
	GetRule(string) (RuleDefinition, bool)
	AddRule(string) ([]RuleDefinition, error)
	DeleteRule(string) ([]RuleDefinition, error)
	GetAllRuleInfo() []byte
}
