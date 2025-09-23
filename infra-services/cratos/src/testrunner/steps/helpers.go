package steps

import (
	"encoding/json"
	"io/ioutil"
	"telemetry/utils/alert"
	"testgomodule/types"

	"google.golang.org/protobuf/proto"
)

// Reads JSON data from a file and converts it to an Alert protobuf message
func LoadAlertFromJSON(filename string) ([]byte, types.SentDataMeta, error) {
	inputDataFile := testFilePath(filename)
	data, err := ioutil.ReadFile(inputDataFile)
	meta := types.SentDataMeta{}
	if err != nil {
		return nil, meta, err
	}
	var AlertStream alert.AlertStream
	if err := json.Unmarshal(data, &AlertStream); err != nil {
		return nil, meta, err
	}
	aBytes, err := proto.Marshal(&AlertStream)
	if err != nil {
		return nil, meta, err
	}
	for _, aObj := range AlertStream.AlertObject {
		meta.FabricName = aObj.FabricName
	}
	return aBytes, meta, nil
}

// Reads JSON rule config from a file and converts it to an Alert protobuf message
func LoadRulesFromJSON(filename string) ([]byte, types.SentConfigMeta, *AlertRuleMsg, error) {
	var sentConfigMeta types.SentConfigMeta
	ruleBytes, err := ioutil.ReadFile(testFilePath(filename))
	if err != nil {
		return nil, sentConfigMeta, nil, err
	}
	ruleConfig := AlertRuleMsg{}
	if err := json.Unmarshal(ruleBytes, &ruleConfig); err != nil {
		return nil, sentConfigMeta, nil, err
	}
	for _, rule := range ruleConfig.AlertRules {
		for _, action := range rule.AlertRuleActions {
			switch action.Action {
			case "CUSTOMIZE_ANOMALY":
				sentConfigMeta.ActionCustomRecoValue = rule.CustomizeAnomaly.CustomMessage
			case "SEVERITY_OVERRIDE":
				sentConfigMeta.ActionSeverityValue = rule.SeverityOverride
			case "ACKNOWLEDGE":
				sentConfigMeta.ActionAcknowledged = true
			}
		}
	}
	return ruleBytes, sentConfigMeta, &ruleConfig, nil
}

func testFilePath(file string) string {
	return "testdata/" + file
}

type AlertRuleMsg struct {
	Metadata   AlertRuleMetadata `json:"metadata"`
	AlertRules []AlertRuleConfig `json:"alertRulePayload,omitempty"`
}

type AlertRuleMetadata struct {
	RuleType      string `json:"ruleType"`
	AlertType     string `json:"alertType"`
	RuleEventType string `json:"action"`
}

type AlertRuleConfig struct {
	UUID                        string                    `json:"uuid"`
	Name                        string                    `json:"name"`
	Priority                    int64                     `json:"priority,omitempty"`
	Description                 string                    `json:"description,omitempty"`
	State                       string                    `json:"state,omitempty"`
	CustomizeAnomaly            CustomizeAnomalyConfig    `json:"customizeAnomaly,omitempty"`
	SeverityOverride            string                    `json:"severityOverride,omitempty"`
	AssociatedInsightGroupUuids []string                  `json:"associatedInsightGroupUuids,omitempty"`
	AlertRuleActions            []RuleActionConfig        `json:"alertRuleActions,omitempty"`
	AlertRuleMatchCriteria      []RuleMatchCriteriaConfig `json:"alertRuleMatchCriteria,omitempty"`
	LastModifiedTime            int64                     `json:"lastModifiedTime"`
	Links                       []interface{}             `json:"links,omitempty"`
	ApplyActionsToAll           bool                      `json:"applyToExistingActiveAnomalies,omitempty"`
}

type CustomizeAnomalyConfig struct {
	CustomMessage string `json:"customMessage,omitempty"`
}

type RuleActionConfig struct {
	Action               string `json:"action"`
	ApplyToActiveAnomaly string `json:"applyToActiveAnomaly,omitempty"`
}

type RuleMatchCriteriaConfig struct {
	CategoryMatchCriteria       []MatchCriteria `json:"categoryMatchCriteria,omitempty"`
	EventNameMatchCriteria      []MatchCriteria `json:"eventNameMatchCriteria,omitempty"`
	AffectedObjectMatchCriteria []MatchCriteria `json:"affectedObjectMatchCriteria,omitempty"`
	SeverityMatchCriteria       []MatchCriteria `json:"severityMatchCriteria,omitempty"`
	UUID                        string          `json:"uuid"`
	AlertRuleId                 string          `json:"alertRuleId"`
	SiteId                      string          `json:"siteId,omitempty"`
}

type MatchCriteria struct {
	ObjectType  string `json:"objectType,omitempty"`
	ValueEquals string `json:"valueEquals,omitempty"`
}
