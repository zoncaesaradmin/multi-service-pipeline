package ruleenginelib

import (
	"encoding/json"
)

var severityMap = map[string]string{
	"EVENT_SEVERITY_WARNING":  "warning",
	"EVENT_SEVERITY_CRITICAL": "critical",
	"EVENT_SEVERITY_MAJOR":    "major",
	"EVENT_SEVERITY_MINOR":    "minor",
}

// Add field name mapping to convert eventName to title

var fieldNameMap = map[string]string{
	"eventName": "title",
}

type MatchRuleInput struct {
	SeverityCriteria       MatchCritSeverity         `json:"severityMatchCriteria"`
	CategoryCriteria       MatchCritCategory         `json:"categoryMatchCriteria"`
	EventNameCriteria      MatchCritEventName        `json:"eventNameMatchCriteria"`
	AffectedObjectCriteria []MatchCritAffectedObject `json:"affectedObjectMatchCriteria"`
}

type MatchCritSeverity struct {
	ValueEquals []string `json:"valueEquals"`
}

type MatchCritCategory struct {
	ValueEquals []string `json:"valueEquals"`
}

type MatchCritEventName struct {
	ValueEquals []string `json:"valueEquals"`
}

type MatchCritAffectedObject struct {
	ObjectType  string   `json:"objectType"`
	ValueEquals []string `json:"valueEquals"`
}

type ToDoAction struct {
	Action               string `json:"action"`
	ApplyToActiveAnomaly bool   `json:"applyToActiveAnomaly"`
}

type MatchRuleEntry struct {
	ValueEquals []interface{} `json:"valueEquals"`
}

type AffectedObject struct {
	ObjectType  string        `json:"objectType"`
	ValueEquals []interface{} `json:"valueEquals"`
}

type OldRule struct {
	RuleName         string         `json:"ruleName"`
	Description      string         `json:"ruleDescription"`
	State            bool           `json:"state"`
	LastModifiedTime int64          `json:"lastModifiedTime"`
	AlertRuleUuid    string         `json:"alertRuleUuid"`
	SiteId           string         `json:"siteId,omitempty"`
	Actions          []ToDoAction   `json:"actions,omitempty"`
	CustomMessage    []string       `json:"customMessage,omitempty"`
	MatchRule        MatchRuleInput `json:"matchRule,omitempty"`
	//MatchRule        map[string]json.RawMessage `json:"matchRule,omitempty"`
}

func ConvertToRuleEngineFormat(oldData []byte) ([]byte, error) {
	var oldRules []OldRule
	if err := json.Unmarshal(oldData, &oldRules); err != nil {
		return nil, err
	}

	if len(oldRules) == 0 {
		return nil, nil
	}

	// in rule format received on alertRules topic,
	// multiple match criteria entries are replicated with rule level info.
	// so, just take the common info from the first entry
	newRule := &RuleBlock{
		//Type:             "",
		//SubType:          "",
		Name:             oldRules[0].RuleName,
		Description:      oldRules[0].Description,
		State:            oldRules[0].State,
		LastModifiedTime: oldRules[0].LastModifiedTime,
		UUID:             oldRules[0].AlertRuleUuid,
		RuleEntries:      make([]*RuleEntry, 0),
	}

	for _, rule := range oldRules {

		ruleEntry := RuleEntry{
			Condition: AstCondition{},
			Actions:   make([]Action, 0),
		}

		for _, action := range rule.Actions {
			actionPayload := map[string]interface{}{
				"applyToExisting": action.ApplyToActiveAnomaly,
			}
			actionPayload["ACKNOWLEDGE"] = false
			if action.Action == "CUSTOMIZE_ANOMALY" && len(rule.CustomMessage) > 0 {
				actionPayload["customMessage"] = rule.CustomMessage
			}
			if action.Action == "ACKNOWLEDGE" {
				actionPayload["ACKNOWLEDGE"] = true
			}
			NewAction := Action{
				Type:    action.Action,
				Payload: actionPayload,
				ReApply: action.ApplyToActiveAnomaly,
			}
			ruleEntry.Actions = append(ruleEntry.Actions, NewAction)
		}

		conditionals := make([]AstConditional, 0)
		if rule.SiteId != "" {
			conditionals = append(conditionals, AstConditional{
				Fact:     "fabricName",
				Operator: "anyof",
				Value:    rule.SiteId,
			})
		}
		if len(rule.MatchRule.CategoryCriteria.ValueEquals) > 0 {
			conditionals = append(conditionals, AstConditional{
				Fact:     "category",
				Operator: "anyof",
				Value:    rule.MatchRule.CategoryCriteria.ValueEquals,
			})
		}
		if len(rule.MatchRule.EventNameCriteria.ValueEquals) > 0 {
			conditionals = append(conditionals, AstConditional{
				Fact:     "title",
				Operator: "anyof",
				Value:    rule.MatchRule.EventNameCriteria.ValueEquals,
			})
		}
		if len(rule.MatchRule.SeverityCriteria.ValueEquals) > 0 {
			sevs := []string{}
			for _, sev := range rule.MatchRule.SeverityCriteria.ValueEquals {
				if severity, exists := severityMap[sev]; exists {
					sevs = append(sevs, severity)
				}
			}
			if len(sevs) > 0 {
				conditionals = append(conditionals, AstConditional{
					Fact:     "severity",
					Operator: "anyof",
					Value:    sevs,
				})
			}
		}
		if len(rule.MatchRule.AffectedObjectCriteria) > 0 {
			for _, affObj := range rule.MatchRule.AffectedObjectCriteria {
				conditionals = append(conditionals, AstConditional{
					Fact:     affObj.ObjectType,
					Operator: "anyof",
					Value:    affObj.ValueEquals,
				})
			}
		}

		ruleEntry.Condition.All = conditionals
		newRule.RuleEntries = append(newRule.RuleEntries, &ruleEntry)
	}

	output, err := json.Marshal(newRule)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func GetAllConfiguredAlertRules() [][]byte {
	// This function should return all configured alert rules in the system.
	// The implementation is not provided here, as it depends on the specific
	// context and requirements of the application.
	return nil
}
