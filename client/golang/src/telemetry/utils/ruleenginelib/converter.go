package ruleenginelib

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	ethPattern = "(?:e|et|eth|ether|ethernet)(\\d+/\\d+(\\.\\d+|/\\d+)*)"
	pcPattern  = "(?:p|po|por|port|pc|port-channel)(\\d+(\\.\\d+)*)"
	loPattern  = "(?:l|lo|loo|loop|loopb|loopba|loopbac|loopback)(\\d+)"
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
	SeverityOverride            string                    `json:"overrideSeverity,omitempty"`
	AssociatedInsightGroupUuids []string                  `json:"associatedInsightGroupUuids,omitempty"`
	AlertRuleActions            []RuleActionConfig        `json:"alertRuleActions,omitempty"`
	AlertRuleMatchCriteria      []RuleMatchCriteriaConfig `json:"alertRuleMatchCriteria,omitempty"`
	LastModifiedTime            int64                     `json:"lastModifiedTime"`
	Links                       []interface{}             `json:"links,omitempty"`
	ApplyActionsToAllStr        string                    `json:"applyActionsToActiveAnomalies,omitempty"`
}

type CustomizeAnomalyConfig struct {
	CustomMessage string `json:"customMessage,omitempty"`
}

type RuleActionConfig struct {
	Action string `json:"action"`
}

type RuleMatchCriteriaConfig struct {
	CategoryMatchCriteria       []MatchCriteria `json:"categoryMatchCriteria,omitempty"`
	EventNameMatchCriteria      []MatchCriteria `json:"eventNameMatchCriteria,omitempty"`
	AffectedObjectMatchCriteria []MatchCriteria `json:"affectedObjectMatchCriteria,omitempty"`
	SeverityMatchCriteria       []MatchCriteria `json:"severityMatchCriteria,omitempty"`
	UUID                        string          `json:"uuid"`
	AlertRuleId                 string          `json:"alertRuleId"`
	SiteId                      string          `json:"siteId,omitempty"`
	Scope                       string          `json:"scope,omitempty"`
}

type MatchCriteria struct {
	ObjectType  string `json:"resourceType,omitempty"`
	ValueEquals string `json:"valueEquals,omitempty"`
}

// Helper to process rule match criteria
func processRuleMatchCriteria(rule AlertRuleConfig) map[string][]*RuleMatchCondition {
	matchCriteriaEntries := make(map[string][]*RuleMatchCondition)

	for _, criteria := range rule.AlertRuleMatchCriteria {

		primKey := criteria.SiteId
		if criteria.Scope == ScopeSystem {
			primKey = PrimaryKeySystem
		}
		if primKey == "" {
			// Use default key if no site ID specified or it is not system cscope as well
			primKey = PrimaryKeyDefault
		}
		conditionals := make([]AstConditional, 0)

		// fabricName
		conditionals = append(conditionals, AstConditional{
			Identifier: MatchKeyFabricName,
			Operator:   "anyof",
			Value:      []string{criteria.SiteId},
		})

		// Category
		categories := make([]string, 0)
		for _, cat := range criteria.CategoryMatchCriteria {
			categories = append(categories, strings.ToUpper(cat.ValueEquals))
		}
		if len(categories) > 0 {
			conditionals = append(conditionals, AstConditional{
				Identifier: MatchKeyCategory,
				Operator:   "anyof",
				Value:      categories,
			})
		}

		// EventName / title
		titles := make([]string, 0)
		for _, evt := range criteria.EventNameMatchCriteria {
			titles = append(titles, evt.ValueEquals)
		}
		if len(titles) > 0 {
			conditionals = append(conditionals, AstConditional{
				Identifier: MatchKeyTitle,
				Operator:   "anyof",
				Value:      titles,
			})
		}

		// Severity match
		sevMatches := make([]string, 0)
		for _, sev := range criteria.SeverityMatchCriteria {
			convSeverity := NormalizeSeverity(sev.ValueEquals)
			sevMatches = append(sevMatches, convSeverity)
		}
		if len(sevMatches) > 0 {
			conditionals = append(conditionals, AstConditional{
				Identifier: MatchKeySeverity,
				Operator:   "anyof",
				Value:      sevMatches,
			})
		}

		// object match conditions
		for _, objMatch := range criteria.AffectedObjectMatchCriteria {
			id := convertObjIdentifier(objMatch.ObjectType)
			conditionals = append(conditionals, AstConditional{
				Identifier: id,
				Operator:   "eq",
				Value:      convertObjIdentifierValue(id, objMatch.ValueEquals),
			})
		}

		matchCriteriaEntries[criteria.UUID] = append(matchCriteriaEntries[criteria.UUID],
			&RuleMatchCondition{
				AlertRuleUUID:     rule.UUID,
				Priority:          rule.Priority,
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
		applyToAll := false
		if strings.EqualFold(rule.ApplyActionsToAllStr, "true") {
			applyToAll = true
		}
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
			ApplyActionsToAll:    applyToAll,
		}

		// Process actions
		for _, action := range rule.AlertRuleActions {
			reActionType := convertToActionType(action.Action)
			reActionValue := ""
			switch reActionType {
			case RuleActionCustomizeRecommendation:
				reActionValue = rule.CustomizeAnomaly.CustomMessage
			case RuleActionSeverityOverride:
				reActionValue = NormalizeSeverity(rule.SeverityOverride)
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
	case "OVERRIDE_SEVERITY":
		return RuleActionSeverityOverride
	default:
		return "unknown"
	}
}

func NormalizeSeverity(severity string) string {
	switch strings.ToUpper(severity) {
	case "EVENT_SEVERITY_CRITICAL", "CRITICAL":
		return SeverityCritical
	case "EVENT_SEVERITY_MAJOR", "MAJOR":
		return SeverityMajor
	case "EVENT_SEVERITY_MINOR", "MINOR":
		return SeverityMinor
	case "EVENT_SEVERITY_WARNING", "WARNING":
		return SeverityWarning
	}
	return SeverityDefault
}

func convertObjIdentifier(objectType string) string {
	switch strings.ToLower(objectType) {
	case "interface":
		return MatchKeyInterface
	case "switch", "node", "leaf":
		return MatchKeySwitch
	case "ip", "ipv4", "ipv6", "address":
		return MatchKeyIp
	case "vni":
		return MatchKeyVni
	case "vrf":
		return MatchKeyVrf
	case "tenant":
		return MatchKeyTenant
	case "subnet", "network":
		return MatchKeySubnet
	}
	return objectType
}

func convertObjIdentifierValue(objectType string, value string) string {
	switch strings.ToLower(objectType) {
	case "interface":
		return NormalizeInterfaceName(value)
	}
	return value
}

func NormalizeInterfaceName(intfName string) string {
	convertedStr := strings.ToLower(strings.TrimSpace(intfName))
	if strings.HasPrefix(convertedStr, "e") {
		r, _ := regexp.Compile(ethPattern)
		matched := r.FindStringSubmatch(convertedStr)
		if len(matched) > 1 {
			return "Ethernet" + matched[1]
		}
	} else if strings.HasPrefix(convertedStr, "p") {
		r, _ := regexp.Compile(pcPattern)
		matched := r.FindStringSubmatch(convertedStr)
		if len(matched) > 1 {
			return "Port-channel" + matched[1]
		}
	} else if strings.HasPrefix(convertedStr, "l") {
		r, _ := regexp.Compile(loPattern)
		matched := r.FindStringSubmatch(convertedStr)
		if len(matched) > 1 {
			return "Loopback" + matched[1]
		}
	}
	return intfName
}

func GetAllConfiguredAlertRules() [][]byte {
	return nil
}
