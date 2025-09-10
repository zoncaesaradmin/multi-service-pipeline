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
	RuleMap   map[string]*RuleDefinition
	Mutex     sync.Mutex
	Logger    *Logger
	RuleTypes []string
}

type RuleDefinition struct {
	AlertRuleUUID        string                           `json:"alertRuleUUID,omitempty"`
	Name                 string                           `json:"name,omitempty"`
	Priority             int64                            `json:"priority,omitempty"`
	Description          string                           `json:"description,omitempty"`
	Enabled              bool                             `json:"enabled"`
	LastModifiedTime     int64                            `json:"lastModifiedTime,omitempty"`
	MatchCriteriaEntries map[string][]*RuleMatchCondition `json:"matchCriteriaEntries,omitempty"`
	Actions              []*RuleAction                    `json:"actions,omitempty"`
}

type RuleMatchCondition struct {
	CriteriaUUID      string       `json:"criteriaUUID,omitempty"`
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
func (re *RuleEngine) AddRule(rules string) error {
	parsedRules := ParseJSON(rules)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range parsedRules {
		re.RuleMap[rule.AlertRuleUUID] = rule
	}
	return nil
}

// DeleteRule removes a rule from the engine by its UUID
func (re *RuleEngine) DeleteRule(rule string) {
	parsedRules := ParseJSON(rule)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range parsedRules {
		delete(re.RuleMap, rule.AlertRuleUUID)
	}
}

// GetRule retrieves a rule from the engine by its UUID
func (re *RuleEngine) GetRule(ruleUUID string) (*RuleDefinition, bool) {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	rule, exists := re.RuleMap[ruleUUID]
	return rule, exists
}

// EvaluateRules evaluates all rules against the provided data
func (re *RuleEngine) EvaluateRules(data Data) RuleLookupResult {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range re.RuleMap {
		if result := re.evaluateSingleRule(rule, data); result.IsRuleHit {
			return result
		}
	}
	return RuleLookupResult{}
}

// evaluateSingleRule checks all match criteria entries for a rule and returns a RuleLookupResult if a match is found
func (re *RuleEngine) evaluateSingleRule(rule *RuleDefinition, data Data) RuleLookupResult {
	for _, matchCriteriaEntries := range rule.MatchCriteriaEntries {
		for _, matchEntry := range matchCriteriaEntries {
			if re.EvaluateMatchCondition(matchEntry.Condition, data) {
				if defaultOptions.FirstMatch {
					return RuleLookupResult{
						IsRuleHit: true,
						RuleUUID:  rule.AlertRuleUUID,
						// Create a new RuleHitCriteria directly
						CriteriaHit: RuleHitCriteria{
							Any: append([]AstConditional{}, matchEntry.Condition.Any...),
							All: append([]AstConditional{}, matchEntry.Condition.All...),
						},
						Actions: deepCopyActions(rule.Actions),
					}
				}
			}
		}
	}
	return RuleLookupResult{}
}

// Helper function to create a deep copy of actions
func deepCopyActions(actions []*RuleAction) []RuleHitAction {
	actionsCopy := make([]RuleHitAction, len(actions))
	for i, action := range actions {
		actionsCopy[i] = RuleHitAction{
			ActionType:     action.ActionType,
			ActionValueStr: action.ActionValueStr,
		}
	}
	return actionsCopy
}

// NewRuleEngineInstance creates a new instance of RuleEngine with the given options
func NewRuleEngineInstance(options *EvaluatorOptions, logger *Logger) *RuleEngine {
	opts := options
	if opts == nil {
		opts = defaultOptions
	}

	return &RuleEngine{
		EvaluatorOptions: *opts,
		RuleMap:          make(map[string]*RuleDefinition, 0),
		Logger:           logger,
	}
}
