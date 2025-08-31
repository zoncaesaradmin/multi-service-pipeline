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
	MatchKeyLeaf       = "leaf"
	MatchKeyInterface  = "interface"
	MatchKeyVrf        = "vrf"
	MatchKeyVni        = "vni"
	MatchKeySubnet     = "subnet"
)

// Constants for rule types
const (
	RuleTypeMgmt   = "RULE_TYPE_MGMT"
	RuleTypeSource = "RULE_TYPE_SOURCE"
)

// Constants for CRUD operations on rule
const (
	RuleOpCreate = "CREATE_ALERT_RULE"
	RuleOpUpdate = "UPDATE_ALERT_RULE"
	RuleOpDelete = "DELETE_ALERT_RULE"
)

type AlertRuleConfigMetadata struct {
	Action string `json:"action"`
}

type ActionSeverity struct {
	SeverityValue string `json:"severityValue,omitempty"`
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

type RuleMessage struct {
	Metadata AlertRuleConfigMetadata `json:"metadata"`
	Rules    []any                   `json:"payload"`
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

func IsValidRuleMsg(msgType string) bool {
	switch msgType {
	case RuleOpCreate, RuleOpUpdate, RuleOpDelete:
		return true
	default:
		return false
	}
}

func (re *RuleEngine) HandleRuleEvent(msgBytes []byte) (*RuleMsgResult, error) {
	var rInput RuleMessage
	re.Logger.Infof("RELIB - received rule msg: %v", string(msgBytes))

	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		re.Logger.Infof("Failed to unmarshal rule message: %v", err.Error())
		// log and ignore invalid messages
		return nil, err
	}

	re.Logger.Debugf("RELIB - received msg unmarshalled: %+v", rInput)

	msgType := rInput.Metadata.Action
	result := &RuleMsgResult{Action: msgType}

	if !IsValidRuleMsg(msgType) {
		re.Logger.Debug("RELIB - ignore invalid/non-relevant rule")
		return result, nil
	}

	re.Logger.Infof("RELIB - processing valid rule for operation: %s", msgType)

	jsonBytes, err := json.Marshal(rInput.Rules)
	if err != nil {
		re.Logger.Infof("RELIB - failed to marshal rule payload: %v", err.Error())
		return result, err
	}

	re.Logger.Infof("RELIB - processed rule payload: %s", string(jsonBytes))

	ruleJsonBytes, err := ConvertToRuleEngineFormat(jsonBytes)
	if err != nil {
		re.Logger.Infof("RELIB - failed to convert rule format: %v", err.Error())
		return result, err
	}
	if len(ruleJsonBytes) == 0 {
		re.Logger.Info("RELIB - invalid empty rule to process, ignored")
		return result, fmt.Errorf("empty rule after conversion")
	}

	re.Logger.Infof("RELIB - rule to be processed %s", string(ruleJsonBytes))
	re.handleRuleMsgEvents(ruleJsonBytes, msgType)

	result.RuleJSON = ruleJsonBytes
	return result, nil
}

func (re *RuleEngine) handleRuleMsgEvents(rmsg []byte, msgType string) {
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
	EvaluateRules(Data) (bool, string, *RuleEntry)
}
