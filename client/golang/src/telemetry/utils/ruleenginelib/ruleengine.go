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
	SortAscending:      false,
}

// RuleEngine represents the main rule engine with its configuration and state
type RuleEngine struct {
	EvaluatorOptions
	RuleMap         map[string]*RuleDefinition
	PrimaryKeyIndex map[string][]*RuleMatchCondition // Primary key -> sorted conditions by priority
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
	ApplyActionsToAll    bool                             `json:"applyActionsToAll,omitempty"`
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
func (re *RuleEngine) AddRule(rules string) ([]RuleDefinition, error) {
	parsedRules := ParseJSON(rules)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range parsedRules {
		re.RuleMap[rule.AlertRuleUUID] = rule
		re.rebuildIndexes() // Rebuild indexes after adding rules
	}
	resRules := make([]RuleDefinition, 0, len(parsedRules))
	for _, rule := range parsedRules {
		resRules = append(resRules, re.deepCopyRuleDefinition(rule))
	}
	return resRules, nil
}

// DeleteRule removes a rule from the engine by its UUID and updates all indexes
func (re *RuleEngine) DeleteRule(rule string) ([]RuleDefinition, error) {
	parsedRules := ParseJSON(rule)
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	for _, rule := range parsedRules {
		delete(re.RuleMap, rule.AlertRuleUUID)
	}
	re.rebuildIndexes() // Rebuild indexes after deleting rules
	resRules := make([]RuleDefinition, 0, len(parsedRules))
	for _, rule := range parsedRules {
		resRules = append(resRules, re.deepCopyRuleDefinition(rule))
	}
	return resRules, nil
}

// GetRule retrieves a copy of a rule from the engine by its UUID
func (re *RuleEngine) GetRule(ruleUUID string) (RuleDefinition, bool) {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()
	rule, exists := re.RuleMap[ruleUUID]
	if !exists {
		return RuleDefinition{}, false
	}
	// Return a deep copy of the rule to prevent external modifications
	return re.deepCopyRuleDefinition(rule), true
}

// EvaluateRules evaluates rules against the provided data using primary key optimization at condition level
func (re *RuleEngine) EvaluateRules(data Data) RuleLookupResult {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	// Extract primary key from data
	primaryKey := re.extractPrimaryKey(data)

	criteriaCount := 0
	// Look for conditions specific to this primary key
	if conditions, exists := re.PrimaryKeyIndex[primaryKey]; exists && len(conditions) > 0 {
		for _, condition := range conditions { // Already sorted by priority
			criteriaCount++
			// Evaluate the condition against the data
			if re.EvaluateMatchCondition(condition.Condition, data) {
				// Get the parent rule information using AlertRuleUUID
				rule := re.RuleMap[condition.AlertRuleUUID]
				if rule != nil && rule.Enabled {
					return RuleLookupResult{
						IsRuleHit: true,
						RuleUUID:  rule.AlertRuleUUID,
						CriteriaHit: RuleHitCriteria{
							Any: append([]AstConditional{}, condition.Condition.Any...),
							All: append([]AstConditional{}, condition.Condition.All...),
						},
						Actions:            deepCopyActions(rule.Actions),
						CritCount:          criteriaCount,
						ApplyToAllExisting: rule.ApplyActionsToAll,
					}
				}
			}
		}
	}

	// No match found
	return RuleLookupResult{
		CritCount: criteriaCount,
	}
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

// deepCopyRuleDefinition creates a deep copy of a RuleDefinition to prevent external modifications
func (re *RuleEngine) deepCopyRuleDefinition(original *RuleDefinition) RuleDefinition {
	copy := RuleDefinition{
		AlertRuleUUID:     original.AlertRuleUUID,
		Name:              original.Name,
		Priority:          original.Priority,
		Description:       original.Description,
		Enabled:           original.Enabled,
		LastModifiedTime:  original.LastModifiedTime,
		ApplyActionsToAll: original.ApplyActionsToAll,
	}

	// Deep copy MatchCriteriaEntries
	if original.MatchCriteriaEntries != nil {
		copy.MatchCriteriaEntries = make(map[string][]*RuleMatchCondition)
		for key, conditions := range original.MatchCriteriaEntries {
			copiedConditions := make([]*RuleMatchCondition, len(conditions))
			for i, condition := range conditions {
				copiedConditions[i] = &RuleMatchCondition{
					CriteriaUUID:      condition.CriteriaUUID,
					AlertRuleUUID:     condition.AlertRuleUUID,
					PrimaryMatchValue: condition.PrimaryMatchValue,
					Condition: AstCondition{
						Any: append([]AstConditional{}, condition.Condition.Any...),
						All: append([]AstConditional{}, condition.Condition.All...),
					},
				}
			}
			copy.MatchCriteriaEntries[key] = copiedConditions
		}
	}

	// Deep copy Actions
	if original.Actions != nil {
		copy.Actions = make([]*RuleAction, len(original.Actions))
		for i, action := range original.Actions {
			copy.Actions[i] = &RuleAction{
				ActionType:     action.ActionType,
				ActionValueStr: action.ActionValueStr,
			}
		}
	}

	return copy
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

// rebuildIndexes rebuilds primary key indexes at the condition level
// This should be called after any rule modification (add/delete)
func (re *RuleEngine) rebuildIndexes() {
	// Clear existing indexes
	re.PrimaryKeyIndex = make(map[string][]*RuleMatchCondition)

	// Build primary key specific indexes by iterating through all rule conditions
	for _, rule := range re.RuleMap {
		if !rule.Enabled {
			continue
		}

		// Iterate through all match criteria entries in the rule
		for _, matchCriteriaEntries := range rule.MatchCriteriaEntries {
			for _, matchCondition := range matchCriteriaEntries {
				// Ensure AlertRuleUUID and Priority are set for lookup
				// just in case not already set
				matchCondition.AlertRuleUUID = rule.AlertRuleUUID
				matchCondition.Priority = rule.Priority

				// Get primary key for this condition
				primaryKey := matchCondition.PrimaryMatchValue
				if primaryKey == "" {
					primaryKey = PrimaryKeyDefault
				}

				// Add condition to the appropriate primary key index
				re.PrimaryKeyIndex[primaryKey] = append(re.PrimaryKeyIndex[primaryKey], matchCondition)
			}
		}
	}

	// Sort each primary key index by condition priority (order depends on SortAscending flag)
	for primaryKey := range re.PrimaryKeyIndex {
		re.sortConditionsByPriority(re.PrimaryKeyIndex[primaryKey])
	}
}

// sortConditionsByPriority sorts conditions by their priority based on SortAscending flag, then by rule UUID for deterministic ordering
func (re *RuleEngine) sortConditionsByPriority(conditions []*RuleMatchCondition) {
	for i := 0; i < len(conditions)-1; i++ {
		for j := i + 1; j < len(conditions); j++ {
			var shouldSwap bool

			if re.SortAscending {
				// Ascending order: lower priority values come first
				shouldSwap = conditions[i].Priority > conditions[j].Priority
			} else {
				// Descending order: higher priority values come first
				shouldSwap = conditions[i].Priority < conditions[j].Priority
			}

			if shouldSwap {
				conditions[i], conditions[j] = conditions[j], conditions[i]
			} else if conditions[i].Priority == conditions[j].Priority {
				// If priorities are equal, sort by rule UUID for deterministic ordering
				if conditions[i].AlertRuleUUID > conditions[j].AlertRuleUUID {
					conditions[i], conditions[j] = conditions[j], conditions[i]
				}
			}
		}
	}
}

// AddRuleDefinition adds a pre-built rule definition directly
// (useful for testing and advanced use cases)
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
		PrimaryKeyIndex:  make(map[string][]*RuleMatchCondition),
		Logger:           logger,
	}
}

// GetPrimaryKeyStats returns statistics about conditions indexed by primary key
func (re *RuleEngine) GetPrimaryKeyStats() map[string]int {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	stats := make(map[string]int)
	for key, conditions := range re.PrimaryKeyIndex {
		stats[key] = len(conditions)
	}
	return stats
}

// ValidateIndexIntegrity checks that all enabled rule conditions are properly indexed
func (re *RuleEngine) ValidateIndexIntegrity() bool {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	totalIndexedConditions := 0
	for _, conditions := range re.PrimaryKeyIndex {
		totalIndexedConditions += len(conditions)
	}

	expectedConditions := 0
	for _, rule := range re.RuleMap {
		if rule.Enabled {
			for _, matchCriteriaEntries := range rule.MatchCriteriaEntries {
				expectedConditions += len(matchCriteriaEntries)
			}
		}
	}

	// Total indexed conditions should match expected conditions from enabled rules
	return totalIndexedConditions == expectedConditions
}
