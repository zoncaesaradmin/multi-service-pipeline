package ruleenginelib

import (
	"sync"
)

type EvaluatorOptions struct {
	AllowUndefinedVars bool
	FirstMatch         bool
}

var defaultOptions = &EvaluatorOptions{
	AllowUndefinedVars: true,
	FirstMatch:         true,
}

// RuleEngine represents the main rule engine with its configuration and state
type RuleEngine struct {
	EvaluatorOptions
	RuleMap   map[string]RuleBlock
	Mutex     sync.Mutex
	Logger    *Logger
	RuleTypes []string
	rules     []*RuleDefinition // sorted by Priority ascending
}

type RuleDefinition struct {
	AlertRuleUUID         string `json:"alertRuleUUID,omitempty"`
	Name                  string `json:"name,omitempty"`
	Priority              int64  `json:"priority,omitempty"`
	Description           string `json:"description,omitempty"`
	Enabled               bool   `json:"enabled"`
	LastModifiedTime      int64  `json:"lastModifiedTime,omitempty"`
	MatchCriteriaBySiteId map[string][]*RuleMatchCondition
	Actions               []*RuleAction
}

type RuleMatchCondition struct {
	ConditionUUID     string       `json:"conditionUUID,omitempty"`
	PrimaryMatchValue string       `json:"primaryMatchValue,omitempty"`
	Condition         AstCondition `json:"condition"`
}

// EvaluateMatchCondition evaluates a single match criteria entry against the provided data
func (re *RuleEngine) EvaluateMatchCondition(aCond AstCondition, dataMap Data) bool {
	return EvaluateAstCondition(aCond, dataMap, &Options{
		AllowUndefinedVars: re.AllowUndefinedVars,
	})
}

// AddRule adds a new rule to the engine
func (re *RuleEngine) AddRule(rule string) *RuleEngine {
	ruleBlock := ParseJSON(rule)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	re.RuleMap[ruleBlock.UUID] = *ruleBlock
	return re
}

// DeleteRule removes a rule from the engine by its UUID
func (re *RuleEngine) DeleteRule(rule string) {
	ruleBlock := ParseJSON(rule)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	delete(re.RuleMap, ruleBlock.UUID)
}

// EvaluateRules evaluates all rules against the provided data
func (re *RuleEngine) EvaluateRules(data Data) (bool, string, *RuleEntry) {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, ruleBlock := range re.RuleMap {
		for _, rule := range ruleBlock.RuleEntries {
			if re.EvaluateMatchCondition(rule.Condition, data) {
				if defaultOptions.FirstMatch {
					return true, ruleBlock.UUID, rule
				}
			}
		}
	}
	return false, "", nil
}

// NewRuleEngineInstance creates a new instance of RuleEngine with the given options
func NewRuleEngineInstance(options *EvaluatorOptions, logger *Logger) *RuleEngine {
	opts := options
	if opts == nil {
		opts = defaultOptions
	}

	return &RuleEngine{
		EvaluatorOptions: *opts,
		RuleMap:          make(map[string]RuleBlock, 0),
		Logger:           logger,
	}
}
