package ruleenginelib

import (
	"encoding/json"
	"fmt"
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
)

// Constants for rule types
const (
	RuleTypeMgmt      = "ALERT_RULES"
	RuleTypeSource    = "GLOBAL_CATEGORY_RULES"
	ScopeSystem       = "platform-system"
	PrimaryKeyDefault = "PRIMARY_KEY_DEFAULT"
	PrimaryKeySystem  = "PRIMARY_KEY_SYSTEM"
)

// Constants for CRUD operations on rule
const (
	RuleOpCreate = "CREATE_ALERT_RULE"
	RuleOpUpdate = "UPDATE_ALERT_RULE"
	RuleOpDelete = "DELETE_ALERT_RULE"
)

// action types supported and used in RuleAction.ActionType
const (
	RuleActionAcknowledge             = "ACKNOWLEDGE"
	RuleActionSeverityOverride        = "SEVERITY_OVERRIDE"
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

type ActionSeverity struct {
	SeverityValue string `json:"severityValue,omitempty"`
}

// external type for rule lookup result
type RuleLookupResult struct {
	IsRuleHit bool `json:"isRuleHit"`
	// below are relevant only if IsRuleHit is true
	RuleUUID           string          `json:"ruleUUID"`
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
	Action   string
	RuleJSON []byte
}

type UserMetaData struct {
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
	case RuleOpCreate, RuleOpUpdate, RuleOpDelete:
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

func (re *RuleEngine) HandleRuleEvent(msgBytes []byte, userMeta UserMetaData) (*RuleMsgResult, error) {
	var rInput AlertRuleMsg
	trLogger := re.Logger.WithField("traceId", userMeta.TraceId)
	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		trLogger.Infof("Failed to unmarshal rule message: %v", err.Error())
		// log and ignore invalid messages
		return nil, err
	}

	trLogger.Debugf("RELIB - received msg unmarshalled: %+v", rInput)

	msgType := rInput.Metadata.RuleEventType
	result := &RuleMsgResult{Action: msgType}

	if !isRelevantRule(rInput.Metadata, re.RuleTypes) {
		trLogger.Debug("RELIB - ignore invalid/non-relevant rule")
		return result, nil
	}

	trLogger.Infof("RELIB - processing valid rule for operation: %s", msgType)

	ruleJsonBytes, err := ConvertToRuleEngineFormat(rInput.AlertRules)
	if err != nil {
		trLogger.Infof("RELIB - failed to convert rule format: %v", err.Error())
		return result, err
	}
	if len(ruleJsonBytes) == 0 {
		trLogger.Info("RELIB - invalid empty rule to process, ignored")
		return result, fmt.Errorf("empty rule after conversion")
	}

	trLogger.Infof("RELIB - converted rule: %s", string(ruleJsonBytes))
	re.handleRuleMsgEvents(trLogger, ruleJsonBytes, msgType)

	result.RuleJSON = ruleJsonBytes
	return result, nil
}

func (re *RuleEngine) handleRuleMsgEvents(l *Logger, rmsg []byte, msgType string) {
	l.Debugf("RELIB - handleRuleMsgEvents called with msgType: %s, ruleData: %s", msgType, string(rmsg))
	switch msgType {
	case RuleOpCreate:
		// Handle create rule event
		l.Debugf("RELIB - handling create rule msg event")
		re.AddRule(string(rmsg))
	case RuleOpUpdate:
		// Handle update rule event
		l.Debugf("RELIB - handling update rule msg event")
		re.AddRule(string(rmsg))
	case RuleOpDelete:
		// Handle delete rule event
		l.Debugf("RELIB - handling delete rule msg event")
		re.DeleteRule(string(rmsg))
	default:
		l.Infof("RELIB - unknown rule msg event type: %s", msgType)
	}
	l.Debugf("RELIB - handleRuleMsgEvents completed for msgType: %s", msgType)
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

// CheckApplyToExistingFlag parses the rule JSON to check if any action has applyToExisting=true
func CheckApplyToExistingFlag(ruleJSON []byte) (bool, error) {
	var rules []map[string]interface{}
	if err := json.Unmarshal(ruleJSON, &rules); err != nil {
		return false, fmt.Errorf("failed to unmarshal rule JSON: %w", err)
	}

	for _, rule := range rules {
		// Check actions array for applyToExisting flag
		if actions, exists := rule["actions"]; exists {
			if actionArray, ok := actions.([]interface{}); ok {
				for _, actionItem := range actionArray {
					if actionMap, ok := actionItem.(map[string]interface{}); ok {
						if applyVal, hasApply := actionMap["applyToExisting"]; hasApply {
							if applyBool, ok := applyVal.(bool); ok && applyBool {
								return true, nil // Found at least one action with applyToExisting=true
							}
						}
					}
				}
			}
		}
	}

	return false, nil // No applyToExisting=true found
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

type RuleEngineType interface {
	HandleRuleEvent([]byte, UserMetaData) (*RuleMsgResult, error)
	EvaluateRules(Data) RuleLookupResult
	GetRule(string) (RuleDefinition, bool)
	AddRule(string) error
	DeleteRule(string)
}
