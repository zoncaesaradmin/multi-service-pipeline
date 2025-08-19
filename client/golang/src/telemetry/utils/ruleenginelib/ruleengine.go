package ruleenginelib

import (
	"sync"
)

type MatchedResults []Action

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
	Results   MatchedResults
	Mutex     sync.Mutex
	Logger    *Logger
	RuleTypes []string
}

// EvaluateStruct evaluates a single rule against the provided data
func (re *RuleEngine) EvaluateStruct(rule *RuleEntry, dataMap Data) bool {
	return EvaluateRule(rule, dataMap, &Options{
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
			if re.EvaluateStruct(rule, data) {
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
