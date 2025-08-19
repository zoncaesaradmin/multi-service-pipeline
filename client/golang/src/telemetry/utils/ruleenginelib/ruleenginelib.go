package ruleenginelib

import "encoding/json"

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

func (re *RuleEngine) HandleRuleEvent(msgBytes []byte) error {
	var rInput RuleMessage
	err := json.Unmarshal(msgBytes, &rInput)
	re.Logger.Infof("RELIB - received rule msg: %v", string(msgBytes))
	if err != nil {
		re.Logger.Infof("Failed to unmarshal rule message: %v", err.Error())
		// log and ignore invalid messages
		return nil
	}

	re.Logger.Debugf("RELIB - received msg unmarshalled: %+v", rInput)

	msgType := rInput.Metadata.Action
	if IsValidRuleMsg(msgType) {
		re.Logger.Infof("RELIB - processing valid rule for operation: %s", msgType)
	} else {
		re.Logger.Debug("RELIB - ignore invalid/non-relevant rule")
	}

	return nil
}

func (re *RuleEngine) initAlertRules() {
	re.Logger.Info("Initializing alert rules")
	for _, ruleType := range re.RuleTypes {
		re.Logger.Infof("Loading rules for type: %s", ruleType)
		rules, err := re.LoadRules(ruleType)
		if err != nil {
			re.Logger.Errorf("Failed to load rules for type %s: %v", ruleType, err)
			continue
		}
		for _, rule := range rules {
			re.AddRule(rule)
		}
	}
	re.Logger.Info("Alert rules initialization complete")
}
