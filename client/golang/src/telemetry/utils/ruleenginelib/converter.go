package ruleenginelib

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Root struct to hold the entire JSON payload of rule message
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
	UUID                        string          `json:"uuid"`
	AlertRuleId                 string          `json:"alertRuleId"`
	SiteId                      string          `json:"siteId,omitempty"`
}

type MatchCriteria struct {
	ObjectType  string `json:"objectType,omitempty"`
	ValueEquals string `json:"valueEquals,omitempty"`
}

// Helper to process rule match criteria
func processRuleMatchCriteria(rule AlertRuleConfig) map[string][]*RuleMatchCondition {
	matchCriteriaEntries := make(map[string][]*RuleMatchCondition)

	for _, criteria := range rule.AlertRuleMatchCriteria {

		primKey := criteria.SiteId
		if primKey == "" {
			primKey = DefaultPrimaryKey // Use default key if no site ID specified
		}
		conditionals := make([]AstConditional, 0)

		// fabricName
		conditionals = append(conditionals, AstConditional{
			Identifier: "fabricName",
			Operator:   "anyof",
			Value:      []string{criteria.SiteId},
		})

		// Category
		categories := make([]string, 0)
		for _, cat := range criteria.CategoryMatchCriteria {
			categories = append(categories, cat.ValueEquals)
		}
		conditionals = append(conditionals, AstConditional{
			Identifier: "category",
			Operator:   "anyof",
			Value:      categories,
		})

		// EventName / title
		titles := make([]string, 0)
		for _, evt := range criteria.EventNameMatchCriteria {
			titles = append(titles, evt.ValueEquals)
		}
		conditionals = append(conditionals, AstConditional{
			Identifier: "title",
			Operator:   "anyof",
			Value:      titles,
		})

		// object match conditions
		for _, objMatch := range criteria.AffectedObjectMatchCriteria {
			conditionals = append(conditionals, AstConditional{
				Identifier: objMatch.ObjectType,
				Operator:   "eq",
				Value:      objMatch.ValueEquals,
			})
		}

		matchCriteriaEntries[criteria.UUID] = append(matchCriteriaEntries[criteria.UUID],
			&RuleMatchCondition{
				CriteriaUUID:      criteria.UUID,
				PrimaryMatchValue: primKey,
				Condition: AstCondition{
					All: conditionals,
				},
			},
		)
	}

	return matchCriteriaEntries
}

// convert rule msg format to rule engine format
func ConvertToRuleEngineFormat(rules []AlertRuleConfig) ([]byte, error) {
	ruleDefinitions := make([]RuleDefinition, 0, len(rules))

	for _, rule := range rules {
		// Create a new RuleDefinition for each AlertRuleConfig
		ruleDef := RuleDefinition{
			AlertRuleUUID:        rule.UUID,
			Name:                 rule.Name,
			Priority:             rule.Priority,
			Description:          rule.Description,
			Enabled:              strings.ToLower(rule.State) == "true",
			LastModifiedTime:     rule.LastModifiedTime,
			MatchCriteriaEntries: processRuleMatchCriteria(rule),
			Actions:              make([]*RuleAction, 0, len(rule.AlertRuleActions)),
		}

		// Process actions
		for _, action := range rule.AlertRuleActions {
			reActionType := convertToActionType(action.Action)
			reActionValue := ""
			switch reActionType {
			case RuleActionCustomizeRecommendation:
				reActionValue = rule.CustomizeAnomaly.CustomMessage
			case RuleActionSeverityOverride:
				reActionValue = rule.SeverityOverride
			}
			ruleAction := &RuleAction{
				ActionType:     reActionType,
				ActionValueStr: reActionValue,
			}
			ruleDef.Actions = append(ruleDef.Actions, ruleAction)
		}

		ruleDefinitions = append(ruleDefinitions, ruleDef)
	}

	// Marshal the rule definitions to JSON
	newruleBytes, err := json.Marshal(ruleDefinitions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rule definitions: %w", err)
	}

	return newruleBytes, nil
}

func convertToActionType(action string) string {
	switch action {
	case "CUSTOMIZE_ANOMALY":
		return RuleActionCustomizeRecommendation
	case "ACKNOWLEDGE":
		return RuleActionAcknowledge
	case "SEVERITY_OVERRIDE":
		return RuleActionSeverityOverride
	default:
		return "unknown"
	}
}

func GetAllConfiguredAlertRules() [][]byte {
	return nil
}
