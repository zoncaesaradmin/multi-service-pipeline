package ruleenginelib

import (
	"strings"
	"sync"
)

type EvaluatorOptions struct {
	AllowUndefinedVars bool
	FirstMatch         bool
	SortAscending      bool
}

var defaultOptions = &EvaluatorOptions{
	AllowUndefinedVars: true,
	FirstMatch:         true,
	SortAscending:      true,
}

// RuleEngine represents the main rule engine with its configuration and state
type RuleEngine struct {
	EvaluatorOptions
	RuleMap         map[string]*RuleDefinition
	PrimaryKeyIndex map[string][]*RuleDefinition // Primary key -> sorted rules by priority
	Mutex           sync.Mutex
	Logger          *Logger
	RuleTypes       []string
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
	// fields from rule
	AlertRuleUUID string `json:"alertRuleUUID,omitempty"` // link to rule holding this
	Priority      int64  `json:"priority,omitempty"`      // priority of the rule this condition belongs to

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

// AddRule adds a new rule to the engine and updates all indexes
func (re *RuleEngine) AddRule(rules string) error {
	parsedRules := ParseJSON(rules)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range parsedRules {
		re.RuleMap[rule.AlertRuleUUID] = rule
		re.rebuildIndexes() // Rebuild indexes after adding rules
	}
	return nil
}

// DeleteRule removes a rule from the engine by its UUID and updates all indexes
func (re *RuleEngine) DeleteRule(rule string) {
	parsedRules := ParseJSON(rule)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range parsedRules {
		delete(re.RuleMap, rule.AlertRuleUUID)
	}
	re.rebuildIndexes() // Rebuild indexes after deleting rules
}

// GetRule retrieves a rule from the engine by its UUID
func (re *RuleEngine) GetRule(ruleUUID string) (*RuleDefinition, bool) {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	rule, exists := re.RuleMap[ruleUUID]
	return rule, exists
}

// EvaluateRules evaluates rules against the provided data using primary key optimization
func (re *RuleEngine) EvaluateRules(data Data) RuleLookupResult {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	// Extract primary key from data
	primaryKey := re.extractPrimaryKey(data)

	// Look for rules specific to this primary key
	if rules, exists := re.PrimaryKeyIndex[primaryKey]; exists && len(rules) > 0 {
		for _, rule := range rules { // Already sorted by priority
			if result := re.evaluateSingleRule(rule, data); result.IsRuleHit {
				return result
			}
		}
	}

	// No fallback needed - if primary key mapping is correct, all rules should be findable
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

// extractPrimaryKey extracts primary key from incoming data for rule lookup optimization
func (re *RuleEngine) extractPrimaryKey(data Data) string {
	// Try to extract fabricName (site) as primary key
	if fabricName, exists := data["fabricName"]; exists {
		if fabricStr, ok := fabricName.(string); ok && fabricStr != "" {
			return fabricStr
		}
	}

	// Check for system scope indicators
	if category, exists := data["category"]; exists {
		if catStr, ok := category.(string); ok && strings.Contains(strings.ToLower(catStr), "system") {
			return PrimaryKeySystem
		}
	}

	// Default fallback
	return PrimaryKeyDefault
}

// rebuildIndexes rebuilds primary key indexes
// This should be called after any rule modification (add/delete)
func (re *RuleEngine) rebuildIndexes() {
	// Clear existing indexes
	re.PrimaryKeyIndex = make(map[string][]*RuleDefinition)

	// Build primary key specific indexes
	for _, rule := range re.RuleMap {
		if !rule.Enabled {
			continue
		}

		// Extract primary keys from rule's match criteria
		primaryKeys := re.extractRulePrimaryKeys(rule)

		// Add rule to each relevant primary key index
		for _, primaryKey := range primaryKeys {
			re.PrimaryKeyIndex[primaryKey] = append(re.PrimaryKeyIndex[primaryKey], rule)
		}
	}

	// Sort each primary key index by priority (order depends on SortAscending flag)
	for primaryKey := range re.PrimaryKeyIndex {
		re.sortRulesByPriority(re.PrimaryKeyIndex[primaryKey])
	}
}

// extractRulePrimaryKeys extracts all primary keys that this rule should be indexed under
func (re *RuleEngine) extractRulePrimaryKeys(rule *RuleDefinition) []string {
	primaryKeys := make(map[string]bool)

	// Look through all match criteria entries to find primary keys
	for _, matchCriteriaEntries := range rule.MatchCriteriaEntries {
		for _, matchCondition := range matchCriteriaEntries {
			if matchCondition.PrimaryMatchValue != "" {
				primaryKeys[matchCondition.PrimaryMatchValue] = true
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(primaryKeys))
	for key := range primaryKeys {
		result = append(result, key)
	}

	// If no specific primary keys found, use default
	if len(result) == 0 {
		result = append(result, PrimaryKeyDefault)
	}

	return result
}

// sortRulesByPriority sorts rules by priority based on SortAscending flag, then by UUID for deterministic ordering
func (re *RuleEngine) sortRulesByPriority(rules []*RuleDefinition) {
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			var shouldSwap bool

			if re.SortAscending {
				// Ascending order: lower priority values come first
				shouldSwap = rules[i].Priority > rules[j].Priority
			} else {
				// Descending order: higher priority values come first
				shouldSwap = rules[i].Priority < rules[j].Priority
			}

			if shouldSwap {
				rules[i], rules[j] = rules[j], rules[i]
			} else if rules[i].Priority == rules[j].Priority {
				// If priorities are equal, sort by UUID for deterministic ordering
				if rules[i].AlertRuleUUID > rules[j].AlertRuleUUID {
					rules[i], rules[j] = rules[j], rules[i]
				}
			}
		}
	}
}

// AddRuleDefinition adds a pre-built rule definition directly (useful for testing)
func (re *RuleEngine) AddRuleDefinition(rule *RuleDefinition) {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	re.RuleMap[rule.AlertRuleUUID] = rule
	re.rebuildIndexes()
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
		PrimaryKeyIndex:  make(map[string][]*RuleDefinition),
		Logger:           logger,
	}
}

// GetPrimaryKeyStats returns statistics about rules indexed by primary key
func (re *RuleEngine) GetPrimaryKeyStats() map[string]int {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	stats := make(map[string]int)
	for key, rules := range re.PrimaryKeyIndex {
		stats[key] = len(rules)
	}
	return stats
}

// ValidateIndexIntegrity checks that all enabled rules are properly indexed
func (re *RuleEngine) ValidateIndexIntegrity() bool {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	totalIndexedRules := 0
	for _, rules := range re.PrimaryKeyIndex {
		totalIndexedRules += len(rules)
	}

	enabledRules := 0
	for _, rule := range re.RuleMap {
		if rule.Enabled {
			enabledRules++
		}
	}

	// Note: totalIndexedRules might be >= enabledRules because rules can be indexed
	// under multiple primary keys
	return totalIndexedRules >= enabledRules
}
