package ruleenginelib

import (
	"encoding/json"
	"strings"
)

var (
	getAllAlertRulesApi = "/alertRules"
)

var severityMap = map[string]string{
	"EVENT_SEVERITY_WARNING":  "warning",
	"EVENT_SEVERITY_CRITICAL": "critical",
	"EVENT_SEVERITY_MAJOR":    "major",
	"EVENT_SEVERITY_MINOR":    "minor",
}

var fieldNameMap = map[string]string{
	"eventName": "title",
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
	RuleName         string                     `json:"ruleName"`
	Description      string                     `json:"description"`
	State            bool                       `json:"state"`
	LastModifiedTime int64                      `json:"lastModifiedTime"`
	AlertRuleUuid    string                     `json:"alertRuleUuid"`
	SiteId           string                     `json:"siteId,omitempty"`
	Actions          []ToDoAction               `json:"actions,omitempty"`
	CustomMessage    []string                   `json:"customMessage,omitempty"`
	MatchRule        map[string]json.RawMessage `json:"matchRule,omitempty"`
}

type ConditionClause struct {
	Identifier string      `json:"identifier"`
	Operator   string      `json:"operator"`
	Value      interface{} `json:"value"`
}

type NewAction struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type SubPayload struct {
	Condition map[string][]ConditionClause `json:"condition"`
	Actions   []NewAction                  `json:"actions"`
}

type NewRule struct {
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	State            bool         `json:"state"`
	LastModifiedTime int64        `json:"lastModifiedTime"`
	UUID             string       `json:"uuid"`
	Payload          []SubPayload `json:"payload"`
}

func ConvertToRuleEngineFormat(oldData []byte) ([]byte, error) {
	var oldRules []OldRule
	if err := json.Unmarshal(oldData, &oldRules); err != nil {
		return nil, err
	}

	if len(oldRules) == 0 {
		return nil, nil
	}

	newRule := &NewRule{
		Name:             oldRules[0].RuleName,
		Description:      oldRules[0].Description,
		State:            oldRules[0].State,
		LastModifiedTime: oldRules[0].LastModifiedTime,
		UUID:             oldRules[0].AlertRuleUuid,
		Payload:          []SubPayload{},
	}
	for _, rule := range oldRules {
		subPayload := SubPayload{
			Condition: map[string][]ConditionClause{
				"all": {},
				"any": {},
			},
			Actions: []NewAction{},
		}
		if rule.SiteId != "" {
			subPayload.Condition["all"] = append(subPayload.Condition["all"], ConditionClause{
				Identifier: "fabricName",
				Operator:   "anyof",
				Value:      []interface{}{interface{}(rule.SiteId)},
			})
		}

		for _, action := range rule.Actions {
			payload := map[string]interface{}{
				"applyToActiveAnomaly": action.ApplyToActiveAnomaly,
			}
			if action.Action == "CUSTOMIZE_ANOMALY" && len(rule.CustomMessage) > 0 {
				payload["customMessage"] = rule.CustomMessage
			}
			subPayload.Actions = append(subPayload.Actions, NewAction{
				Type:    action.Action,
				Payload: payload,
			})
		}

		for key, raw := range rule.MatchRule {
			if strings.HasSuffix(key, "MatchCriteria") && len(raw) > 0 {
				if key == "affectedObjectMatchCriteria" {
					objects := []AffectedObject{}
					if err := json.Unmarshal(raw, &objects); err != nil {
						continue
					}
					for _, obj := range objects {
						subPayload.Condition["all"] = append(subPayload.Condition["all"], ConditionClause{
							Identifier: obj.ObjectType,
							Operator:   "anyof",
							Value:      obj.ValueEquals,
						})
					}
				} else {
					match := MatchRuleEntry{}
					if err := json.Unmarshal(raw, &match); err != nil {
						continue
					}
					if len(match.ValueEquals) > 0 {
						valueEquals := []interface{}{}
						if key == "severityMatchCriteria" {
							for _, v := range match.ValueEquals {
								if sev, ok := v.(string); ok {
									if severity, exists := severityMap[sev]; exists {
										valueEquals = append(valueEquals, interface{}(severity))
									}
								}
							}
						} else {
							valueEquals = match.ValueEquals
						}

						// Get the identifier by removing "MatchCriteria" suffix
						identifier := strings.TrimSuffix(key, "MatchCriteria")

						// Map field names if needed
						if mappedName, exists := fieldNameMap[identifier]; exists {
							identifier = mappedName
						}

						subPayload.Condition["all"] = append(subPayload.Condition["all"], ConditionClause{
							Identifier: identifier,
							Operator:   "anyof",
							Value:      valueEquals,
						})
					}
				}
			}
		}
		newRule.Payload = append(newRule.Payload, subPayload)
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
