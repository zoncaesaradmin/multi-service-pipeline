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

type ActionSeverity struct {
	SeverityValue string `json:"severityValue,omitempty"`
}

// external type for rule lookup result
type RuleLookupResult struct {
	IsRuleHit bool `json:"isRuleHit"`
	// below are relevant only if IsRuleHit is true
	RuleUUID    string          `json:"ruleUUID"`
	CriteriaHit RuleHitCriteria `json:"criteria"`
	Actions     []RuleHitAction `json:"actions"`
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

func (re *RuleEngine) HandleRuleEvent(msgBytes []byte) (*RuleMsgResult, error) {
	var rInput AlertRuleMsg
	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		re.Logger.Infof("Failed to unmarshal rule message: %v", err.Error())
		// log and ignore invalid messages
		return nil, err
	}

	re.Logger.Debugf("RELIB - received msg unmarshalled: %+v", rInput)

	msgType := rInput.Metadata.RuleEventType
	result := &RuleMsgResult{Action: msgType}

	if !isRelevantRule(rInput.Metadata, re.RuleTypes) {
		re.Logger.Debug("RELIB - ignore invalid/non-relevant rule")
		return result, nil
	}

	re.Logger.Infof("RELIB - processing valid rule for operation: %s", msgType)

	ruleJsonBytes, err := ConvertToRuleEngineFormat(rInput.AlertRules)
	if err != nil {
		re.Logger.Infof("RELIB - failed to convert rule format: %v", err.Error())
		return result, err
	}
	if len(ruleJsonBytes) == 0 {
		re.Logger.Info("RELIB - invalid empty rule to process, ignored")
		return result, fmt.Errorf("empty rule after conversion")
	}

	re.Logger.Infof("RELIB - converted rule: %s", string(ruleJsonBytes))
	re.handleRuleMsgEvents(ruleJsonBytes, msgType)

	result.RuleJSON = ruleJsonBytes
	return result, nil
}

func (re *RuleEngine) handleRuleMsgEvents(rmsg []byte, msgType string) {
	re.Logger.Debugf("RELIB - handleRuleMsgEvents called with msgType: %s, ruleData: %s", msgType, string(rmsg))
	switch msgType {
	case RuleOpCreate:
		// Handle create rule event
		re.Logger.Debugf("RELIB - handling create rule msg event")
		re.AddRule(string(rmsg))
	case RuleOpUpdate:
		// Handle update rule event
		re.Logger.Debugf("RELIB - handling update rule msg event")
		re.AddRule(string(rmsg))
	case RuleOpDelete:
		// Handle delete rule event
		re.Logger.Debugf("RELIB - handling delete rule msg event")
		re.DeleteRule(string(rmsg))
	default:
		re.Logger.Infof("RELIB - unknown rule msg event type: %s", msgType)
	}
	re.Logger.Debugf("RELIB - handleRuleMsgEvents completed for msgType: %s", msgType)
}

// initAlertRules initializes the rules from already existing rules configured
func (re *RuleEngine) initAlertRules() {
	rules := GetAllConfiguredAlertRules()
	for _, rule := range rules {
		re.AddRule(string(rule))
	}
}

type RuleEngineType interface {
	HandleRuleEvent([]byte) (*RuleMsgResult, error)
	EvaluateRules(Data) RuleLookupResult
}
