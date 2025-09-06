package ruleenginelib

// Root struct to hold the entire JSON payload of rule message
type AlertRuleMsg struct {
	Metadata   AlertRuleMetadata `json:"metadata"`
	AlertRules []AlertRule       `json:"alertRulePayload,omitempty"`
}

type AlertRuleMetadata struct {
	RuleType      string `json:"ruleType"`
	AlertType     string `json:"alertType"`
	RuleEventType string `json:"action"`
}

type AlertRule struct {
	UUID                        string                   `json:"uuid"`
	Name                        string                   `json:"name"`
	Priority                    int64                    `json:"priority,omitempty"`
	Description                 string                   `json:"description,omitempty"`
	State                       string                   `json:"state,omitempty"`
	CustomizeAnomaly            CustomizeAnomaly         `json:"customizeAnomaly,omitempty"`
	AssociatedInsightGroupUuids []string                 `json:"associatedInsightGroupUuids,omitempty"`
	AlertRuleActions            []AlertRuleAction        `json:"alertRuleActions,omitempty"`
	AlertRuleMatchCriteria      []AlertRuleMatchCriteria `json:"alertRuleMatchCriteria,omitempty"`
	LastModifiedTime            int64                    `json:"lastModifiedTime"`
	Links                       []interface{}            `json:"links,omitempty"`
}

type CustomizeAnomaly struct {
	CustomMessage string `json:"customMessage,omitempty"`
}

type AlertRuleAction struct {
	Action               string `json:"action"`
	ApplyToActiveAnomaly string `json:"applyToActiveAnomaly,omitempty"`
}

type AlertRuleMatchCriteria struct {
	CategoryMatchCriteria  []MatchCriteria `json:"categoryMatchCriteria,omitempty"`
	EventNameMatchCriteria []MatchCriteria `json:"eventNameMatchCriteria,omitempty"`
	UUID                   string          `json:"uuid"`
	AlertRuleId            string          `json:"alertRuleId"`
	SiteId                 string          `json:"siteId,omitempty"`
}

type MatchCriteria struct {
	ValueEquals string `json:"valueEquals"`
}

func ConvertToRuleEngineFormat(rules []AlertRule) ([]byte, error) {
	var newruleBytes []byte
	return newruleBytes, nil
}

func GetAllConfiguredAlertRules() [][]byte {
	return nil
}
