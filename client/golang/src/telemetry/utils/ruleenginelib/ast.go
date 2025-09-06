package ruleenginelib

import (
	"encoding/json"
)

// Conditionals are the basic units of rules
type AstConditional struct {
	Fact     string      `json:"identifier"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// A Condition is a group of conditionals within a binding context
// that determines how the group will be evaluated.
type AstCondition struct {
	Any []AstConditional `json:"any"`
	All []AstConditional `json:"all"`
}

// Fired when a identifier matches a rule
type RuleAction struct {
	ActionType     string
	ActionValueStr string
}

type RuleBlock struct {
	Type             string       `json:"ruleType,omitempty"`
	SubType          string       `json:"ruleSubType,omitempty"`
	Name             string       `json:"name,omitempty"`
	UUID             string       `json:"uuid,omitempty"`
	Description      string       `json:"description,omitempty"`
	LastModifiedTime int64        `json:"lastModifiedTime,omitempty"`
	State            bool         `json:"state"`
	RuleEntries      []*RuleEntry `json:"payload,omitempty"`
}

type RuleEntry struct {
	Condition AstCondition `json:"condition"`
	Actions   []RuleAction `json:"actions"`
}

// parse JSON string as Rule
func ParseJSON(j string) *RuleBlock {
	var rule *RuleBlock
	if err := json.Unmarshal([]byte(j), &rule); err != nil {
		panic("expected valid JSON")
	}
	return rule
}
