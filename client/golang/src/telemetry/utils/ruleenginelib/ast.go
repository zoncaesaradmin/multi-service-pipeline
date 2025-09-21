package ruleenginelib

import (
	"encoding/json"
)

// Conditionals are the basic units of rules
type AstConditional struct {
	Identifier string      `json:"identifier"`
	Operator   string      `json:"operator"`
	Value      interface{} `json:"value"`
}

// A Condition is a group of conditionals within a binding context
// that determines how the group will be evaluated.
type AstCondition struct {
	Any []AstConditional `json:"any"`
	All []AstConditional `json:"all"`
}

// Fired when a identifier matches a rule
type RuleAction struct {
	ActionType     string `json:"actionType"`
	ActionValueStr string `json:"actionValueStr"`
}

// parse JSON string as Rule
func ParseJSON(j string) []*RuleDefinition {
	var rules []*RuleDefinition
	if err := json.Unmarshal([]byte(j), &rules); err != nil {
		panic("expected valid JSON")
	}
	return rules
}
