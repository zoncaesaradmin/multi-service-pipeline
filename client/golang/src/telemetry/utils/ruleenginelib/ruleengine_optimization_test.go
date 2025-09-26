package ruleenginelib

import (
	"testing"
)

// =============================================================================
// PRIMARY KEY OPTIMIZATION TESTS
// =============================================================================

// TestPrimaryKeyOptimization tests that rules are evaluated in the correct order based on primary keys
func TestPrimaryKeyOptimization(t *testing.T) {
	// Use descending sort order to match the test expectations
	options := &EvaluatorOptions{
		AllowUndefinedVars: true,
		FirstMatch:         true,
		SortAscending:      false, // Descending order for this test
	}
	re := NewRuleEngineInstance(options, nil)

	// Create rules with different primary keys and priorities
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-1",
		Name:          "High Priority Fabric Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "fabricName", Operator: "eq", Value: "fabric-1"},
							{Identifier: "category", Operator: "eq", Value: "HARDWARE"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "OVERRIDE_SEVERITY", ActionValueStr: "critical"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-2",
		Name:          "Low Priority Fabric Rule",
		Priority:      50,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "fabricName", Operator: "eq", Value: "fabric-1"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "OVERRIDE_SEVERITY", ActionValueStr: "minor"},
		},
	}

	rule3 := &RuleDefinition{
		AlertRuleUUID: "rule-3",
		Name:          "Default Key Rule",
		Priority:      200,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-3": {
				{
					CriteriaUUID:      "criteria-3",
					PrimaryMatchValue: "PRIMARY_KEY_DEFAULT",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "HARDWARE"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "OVERRIDE_SEVERITY", ActionValueStr: "warning"},
		},
	}

	// Add rules to engine
	re.AddRuleDefinition(rule1)
	re.AddRuleDefinition(rule2)
	re.AddRuleDefinition(rule3)

	// Test data that matches fabric-1 primary key
	data := Data{
		"fabricName": "fabric-1",
		"category":   "HARDWARE",
	}

	result := re.EvaluateRules(data)

	// Should match rule1 (highest priority within fabric-1)
	if !result.IsRuleHit {
		t.Error("Expected rule hit for fabric-1 data")
	}

	if result.RuleUUID != "rule-1" {
		t.Errorf("Expected rule-1 (highest priority), got %s", result.RuleUUID)
	}

	if len(result.Actions) != 1 || result.Actions[0].ActionValueStr != "critical" {
		t.Errorf("Expected critical severity action, got %v", result.Actions)
	}
}

// TestPriorityOrdering tests that rules are evaluated in priority order
func TestPriorityOrdering(t *testing.T) {
	// Use descending sort order to match the test expectations
	options := &EvaluatorOptions{
		AllowUndefinedVars: true,
		FirstMatch:         true,
		SortAscending:      false, // Descending order for this test
	}
	re := NewRuleEngineInstance(options, nil)

	// Create rules with same primary key but different priorities
	highPriorityRule := &RuleDefinition{
		AlertRuleUUID: "high-priority",
		Name:          "High Priority Rule",
		Priority:      1000,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-high": {
				{
					CriteriaUUID:      "criteria-high",
					PrimaryMatchValue: "fabric-test",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "high-priority-action"},
		},
	}

	lowPriorityRule := &RuleDefinition{
		AlertRuleUUID: "low-priority",
		Name:          "Low Priority Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-low": {
				{
					CriteriaUUID:      "criteria-low",
					PrimaryMatchValue: "fabric-test",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "low-priority-action"},
		},
	}

	// Add low priority rule first, then high priority
	re.AddRuleDefinition(lowPriorityRule)
	re.AddRuleDefinition(highPriorityRule)

	// Test data
	data := Data{
		"fabricName": "fabric-test",
		"category":   "TEST",
	}

	result := re.EvaluateRules(data)

	// Should match high priority rule despite being added second
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	if result.RuleUUID != "high-priority" {
		t.Errorf("Expected high-priority rule, got %s", result.RuleUUID)
	}

	if len(result.Actions) != 1 || result.Actions[0].ActionValueStr != "high-priority-action" {
		t.Errorf("Expected high-priority-action, got %v", result.Actions)
	}
}

// TestGlobalFallback tests that rules with default primary key are found when data has no specific primary key
func TestGlobalFallback(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Rule with specific primary key
	specificRule := &RuleDefinition{
		AlertRuleUUID: "specific-rule",
		Name:          "Specific Fabric Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-specific": {
				{
					CriteriaUUID:      "criteria-specific",
					PrimaryMatchValue: "fabric-specific",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "SPECIFIC"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "specific-action"},
		},
	}

	// Rule with default primary key (fallback)
	fallbackRule := &RuleDefinition{
		AlertRuleUUID: "fallback-rule",
		Name:          "Fallback Rule",
		Priority:      200,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-fallback": {
				{
					CriteriaUUID:      "criteria-fallback",
					PrimaryMatchValue: "PRIMARY_KEY_DEFAULT",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "GENERAL"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "fallback-action"},
		},
	}

	re.AddRuleDefinition(specificRule)
	re.AddRuleDefinition(fallbackRule)

	// Test data that doesn't have a fabricName (so it defaults to PRIMARY_KEY_DEFAULT)
	data := Data{
		"category": "GENERAL",
	}

	result := re.EvaluateRules(data)

	// Should match fallback rule via PRIMARY_KEY_DEFAULT index
	if !result.IsRuleHit {
		t.Error("Expected rule hit from default primary key lookup")
	}

	if result.RuleUUID != "fallback-rule" {
		t.Errorf("Expected fallback-rule, got %s", result.RuleUUID)
	}
}

// TestDisabledRulesExcluded tests that disabled rules are not included in indexes
func TestDisabledRulesExcluded(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Disabled rule
	disabledRule := &RuleDefinition{
		AlertRuleUUID: "disabled-rule",
		Name:          "Disabled Rule",
		Priority:      1000,
		Enabled:       false, // Disabled
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-disabled": {
				{
					CriteriaUUID:      "criteria-disabled",
					PrimaryMatchValue: "fabric-test",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "disabled-action"},
		},
	}

	// Enabled rule with lower priority
	enabledRule := &RuleDefinition{
		AlertRuleUUID: "enabled-rule",
		Name:          "Enabled Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-enabled": {
				{
					CriteriaUUID:      "criteria-enabled",
					PrimaryMatchValue: "fabric-test",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "enabled-action"},
		},
	}

	re.AddRuleDefinition(disabledRule)
	re.AddRuleDefinition(enabledRule)

	// Test data
	data := Data{
		"fabricName": "fabric-test",
		"category":   "TEST",
	}

	result := re.EvaluateRules(data)

	// Should match enabled rule, not disabled one despite higher priority
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	if result.RuleUUID != "enabled-rule" {
		t.Errorf("Expected enabled-rule, got %s", result.RuleUUID)
	}

	if len(result.Actions) != 1 || result.Actions[0].ActionValueStr != "enabled-action" {
		t.Errorf("Expected enabled-action, got %v", result.Actions)
	}
}

// TestPrimaryKeyExtraction tests the primary key extraction logic
func TestPrimaryKeyExtraction(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	testCases := []struct {
		name        string
		data        Data
		expectedKey string
	}{
		{
			name:        "Fabric name extraction",
			data:        Data{"fabricName": "fabric-123"},
			expectedKey: "fabric-123",
		},
		{
			name:        "System category detection",
			data:        Data{"category": "SYSTEM_HEALTH"},
			expectedKey: "PRIMARY_KEY_SYSTEM",
		},
		{
			name:        "Default fallback",
			data:        Data{"title": "Some alert"},
			expectedKey: "PRIMARY_KEY_DEFAULT",
		},
		{
			name:        "Empty fabric name fallback",
			data:        Data{"fabricName": ""},
			expectedKey: "PRIMARY_KEY_DEFAULT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := re.extractPrimaryKey(tc.data)
			if result != tc.expectedKey {
				t.Errorf("Expected primary key %s, got %s", tc.expectedKey, result)
			}
		})
	}
}

// TestIndexRebuildAfterRuleModification tests that indexes are properly rebuilt after rule changes
func TestIndexRebuildAfterRuleModification(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Add initial rule
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-1",
		Name:          "Initial Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "rule-1-action"},
		},
	}

	re.AddRuleDefinition(rule1)

	// Verify rule is in index
	data := Data{"fabricName": "fabric-1", "category": "TEST"}
	result := re.EvaluateRules(data)
	if !result.IsRuleHit || result.RuleUUID != "rule-1" {
		t.Error("Initial rule should be found")
	}

	// Delete rule using JSON format
	deleteJSON := `[{"alertRuleUUID":"rule-1","name":"Initial Rule","enabled":true}]`
	re.DeleteRule(deleteJSON)

	// Verify rule is no longer found
	result = re.EvaluateRules(data)
	if result.IsRuleHit {
		t.Error("Deleted rule should not be found")
	}

	// Add new rule with higher priority
	rule2JSON := `[{"alertRuleUUID":"rule-2","name":"New Rule","priority":200,"enabled":true,"matchCriteriaEntries":{"criteria-2":[{"criteriaUUID":"criteria-2","primaryMatchValue":"fabric-1","condition":{"all":[{"identifier":"category","operator":"eq","value":"TEST"}]}}]},"actions":[{"ActionType":"TEST_ACTION","ActionValueStr":"rule-2-action"}]}]`
	re.AddRule(rule2JSON)

	// Verify new rule is found
	result = re.EvaluateRules(data)
	if !result.IsRuleHit || result.RuleUUID != "rule-2" {
		t.Error("New rule should be found")
	}
}

// =============================================================================
// SORTING ORDER TESTS
// =============================================================================

// TestSortingOrderAscending tests priority sorting in ascending order
func TestSortingOrderAscending(t *testing.T) {
	// Create options with ascending sort
	options := &EvaluatorOptions{
		AllowUndefinedVars: true,
		FirstMatch:         true,
		SortAscending:      true,
	}

	re := NewRuleEngineInstance(options, nil)

	// Create rules with different priorities
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-low-priority",
		Name:          "Low Priority Rule",
		Priority:      100, // Lowest priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "low-priority-action"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-medium-priority",
		Name:          "Medium Priority Rule",
		Priority:      500, // Medium priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "medium-priority-action"},
		},
	}

	rule3 := &RuleDefinition{
		AlertRuleUUID: "rule-high-priority",
		Name:          "High Priority Rule",
		Priority:      1000, // Highest priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-3": {
				{
					CriteriaUUID:      "criteria-3",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "high-priority-action"},
		},
	}

	// Add rules in random order
	re.AddRuleDefinition(rule2) // Medium first
	re.AddRuleDefinition(rule3) // High second
	re.AddRuleDefinition(rule1) // Low third

	// Test data
	data := Data{
		"fabricName": "test-fabric",
		"category":   "TEST",
	}

	result := re.EvaluateRules(data)

	// With ascending sort, lowest priority (100) should be evaluated first
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	if result.RuleUUID != "rule-low-priority" {
		t.Errorf("Expected rule-low-priority to be evaluated first with ascending sort, got %s", result.RuleUUID)
	}

	if len(result.Actions) != 1 || result.Actions[0].ActionValueStr != "low-priority-action" {
		t.Errorf("Expected low-priority-action, got %v", result.Actions)
	}
}

// TestSortingOrderDescending tests priority sorting in descending order
func TestSortingOrderDescending(t *testing.T) {
	// Create options with descending sort
	options := &EvaluatorOptions{
		AllowUndefinedVars: true,
		FirstMatch:         true,
		SortAscending:      false,
	}

	re := NewRuleEngineInstance(options, nil)

	// Create rules with different priorities (same as above)
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-low-priority",
		Name:          "Low Priority Rule",
		Priority:      100, // Lowest priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "low-priority-action"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-medium-priority",
		Name:          "Medium Priority Rule",
		Priority:      500, // Medium priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "medium-priority-action"},
		},
	}

	rule3 := &RuleDefinition{
		AlertRuleUUID: "rule-high-priority",
		Name:          "High Priority Rule",
		Priority:      1000, // Highest priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-3": {
				{
					CriteriaUUID:      "criteria-3",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "high-priority-action"},
		},
	}

	// Add rules in random order
	re.AddRuleDefinition(rule1) // Low first
	re.AddRuleDefinition(rule2) // Medium second
	re.AddRuleDefinition(rule3) // High third

	// Test data
	data := Data{
		"fabricName": "test-fabric",
		"category":   "TEST",
	}

	result := re.EvaluateRules(data)

	// With descending sort, highest priority (1000) should be evaluated first
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	if result.RuleUUID != "rule-high-priority" {
		t.Errorf("Expected rule-high-priority to be evaluated first with descending sort, got %s", result.RuleUUID)
	}

	if len(result.Actions) != 1 || result.Actions[0].ActionValueStr != "high-priority-action" {
		t.Errorf("Expected high-priority-action, got %v", result.Actions)
	}
}

// TestSortingOrderDefault tests that default behavior is maintained
func TestSortingOrderDefault(t *testing.T) {
	// Use default options (should be ascending=true)
	re := NewRuleEngineInstance(nil, nil)

	// Verify the default setting
	if !re.SortAscending {
		t.Error("Expected default SortAscending to be true")
	}

	// Create two rules with different priorities
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-1",
		Name:          "Rule 1",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "rule-1-action"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-2",
		Name:          "Rule 2",
		Priority:      200,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "rule-2-action"},
		},
	}

	// Add higher priority rule first
	re.AddRuleDefinition(rule2)
	re.AddRuleDefinition(rule1)

	// Test data
	data := Data{
		"fabricName": "test-fabric",
		"category":   "TEST",
	}

	result := re.EvaluateRules(data)

	// With default ascending sort, lower priority (100) should be evaluated first
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	if result.RuleUUID != "rule-1" {
		t.Errorf("Expected rule-1 to be evaluated first with default ascending sort, got %s", result.RuleUUID)
	}
}

// TestSortingOrderWithEqualPriorities tests UUID-based sorting when priorities are equal
func TestSortingOrderWithEqualPriorities(t *testing.T) {
	// Test with descending order
	options := &EvaluatorOptions{
		AllowUndefinedVars: true,
		FirstMatch:         true,
		SortAscending:      false,
	}

	re := NewRuleEngineInstance(options, nil)

	// Create rules with same priority but different UUIDs
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-a", // Alphabetically first
		Name:          "Rule A",
		Priority:      100, // Same priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-a": {
				{
					CriteriaUUID:      "criteria-a",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "rule-a-action"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-z", // Alphabetically last
		Name:          "Rule Z",
		Priority:      100, // Same priority
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-z": {
				{
					CriteriaUUID:      "criteria-z",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "rule-z-action"},
		},
	}

	// Add rules in reverse alphabetical order
	re.AddRuleDefinition(rule2)
	re.AddRuleDefinition(rule1)

	// Test data
	data := Data{
		"fabricName": "test-fabric",
		"category":   "TEST",
	}

	result := re.EvaluateRules(data)

	// When priorities are equal, UUID sorting should be applied (alphabetical order)
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	if result.RuleUUID != "rule-a" {
		t.Errorf("Expected rule-a to be evaluated first (alphabetical UUID order), got %s", result.RuleUUID)
	}
}

// =============================================================================
// MONITORING AND DEBUGGING TESTS
// =============================================================================

// TestPrimaryKeyStats tests the GetPrimaryKeyStats functionality
func TestPrimaryKeyStats(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Create rules for different primary keys
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-fabric-1",
		Name:          "Fabric 1 Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "action-1"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-fabric-2",
		Name:          "Fabric 2 Rule",
		Priority:      200,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "fabric-2",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "action-2"},
		},
	}

	rule3 := &RuleDefinition{
		AlertRuleUUID: "rule-default",
		Name:          "Default Rule",
		Priority:      300,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-3": {
				{
					CriteriaUUID:      "criteria-3",
					PrimaryMatchValue: "PRIMARY_KEY_DEFAULT",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "action-3"},
		},
	}

	// Add rules
	re.AddRuleDefinition(rule1)
	re.AddRuleDefinition(rule2)
	re.AddRuleDefinition(rule3)

	// Get stats
	stats := re.GetPrimaryKeyStats()

	// Verify stats
	if len(stats) != 3 {
		t.Errorf("Expected 3 primary keys, got %d", len(stats))
	}

	if stats["fabric-1"] != 1 {
		t.Errorf("Expected 1 rule for fabric-1, got %d", stats["fabric-1"])
	}

	if stats["fabric-2"] != 1 {
		t.Errorf("Expected 1 rule for fabric-2, got %d", stats["fabric-2"])
	}

	if stats["PRIMARY_KEY_DEFAULT"] != 1 {
		t.Errorf("Expected 1 rule for PRIMARY_KEY_DEFAULT, got %d", stats["PRIMARY_KEY_DEFAULT"])
	}
}

// TestValidateIndexIntegrity tests the index integrity validation
func TestValidateIndexIntegrity(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Create some rules
	rule1 := &RuleDefinition{
		AlertRuleUUID: "rule-1",
		Name:          "Rule 1",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "action-1"},
		},
	}

	rule2 := &RuleDefinition{
		AlertRuleUUID: "rule-2",
		Name:          "Rule 2",
		Priority:      200,
		Enabled:       false, // Disabled rule
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "fabric-2",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "action-2"},
		},
	}

	rule3 := &RuleDefinition{
		AlertRuleUUID: "rule-3",
		Name:          "Rule 3",
		Priority:      300,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-3": {
				{
					CriteriaUUID:      "criteria-3",
					PrimaryMatchValue: "fabric-1", // Same primary key as rule1
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "severity", Operator: "eq", Value: "critical"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "action-3"},
		},
	}

	// Add rules
	re.AddRuleDefinition(rule1)
	re.AddRuleDefinition(rule2)
	re.AddRuleDefinition(rule3)

	// Validate integrity
	isValid := re.ValidateIndexIntegrity()

	if !isValid {
		t.Error("Expected index integrity to be valid")
	}

	// Verify the actual stats make sense
	stats := re.GetPrimaryKeyStats()

	// Should have 2 enabled rules total
	// rule1 and rule3 both indexed under fabric-1 = 2 rules
	// rule2 is disabled so not indexed
	expectedFabric1Count := 2
	if stats["fabric-1"] != expectedFabric1Count {
		t.Errorf("Expected %d rules for fabric-1, got %d", expectedFabric1Count, stats["fabric-1"])
	}

	// fabric-2 should not appear since rule2 is disabled
	if count, exists := stats["fabric-2"]; exists {
		t.Errorf("Expected fabric-2 to not be indexed (rule is disabled), but found %d rules", count)
	}
}

// TestMultiplePrimaryKeys tests rules that are indexed under multiple primary keys
func TestMultiplePrimaryKeys(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Create a rule that should be indexed under multiple primary keys
	rule := &RuleDefinition{
		AlertRuleUUID: "multi-key-rule",
		Name:          "Multi Primary Key Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					CriteriaUUID:      "criteria-1",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
			"criteria-2": {
				{
					CriteriaUUID:      "criteria-2",
					PrimaryMatchValue: "fabric-2",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "category", Operator: "eq", Value: "TEST"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "multi-action"},
		},
	}

	re.AddRuleDefinition(rule)

	// Check stats
	stats := re.GetPrimaryKeyStats()

	// Rule should be indexed under both fabric-1 and fabric-2
	if stats["fabric-1"] != 1 {
		t.Errorf("Expected 1 rule for fabric-1, got %d", stats["fabric-1"])
	}

	if stats["fabric-2"] != 1 {
		t.Errorf("Expected 1 rule for fabric-2, got %d", stats["fabric-2"])
	}

	// Test that the rule can be found via both primary keys
	data1 := Data{"fabricName": "fabric-1", "category": "TEST"}
	result1 := re.EvaluateRules(data1)
	if !result1.IsRuleHit || result1.RuleUUID != "multi-key-rule" {
		t.Error("Expected to find rule via fabric-1 primary key")
	}

	data2 := Data{"fabricName": "fabric-2", "category": "TEST"}
	result2 := re.EvaluateRules(data2)
	if !result2.IsRuleHit || result2.RuleUUID != "multi-key-rule" {
		t.Error("Expected to find rule via fabric-2 primary key")
	}

	// Validate integrity should still pass
	if !re.ValidateIndexIntegrity() {
		t.Error("Expected index integrity to be valid for multi-key rule")
	}
}

// TestConditionLevelOptimization validates that the rule engine works at condition level instead of rule level
func TestConditionLevelOptimization(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)

	// Create a rule with multiple conditions having different PrimaryMatchValues
	rule := &RuleDefinition{
		AlertRuleUUID: "condition-test-rule",
		Name:          "Multi-Condition Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					AlertRuleUUID:     "condition-test-rule",
					Priority:          100,
					CriteriaUUID:      "crit-fabric1",
					PrimaryMatchValue: "fabric-1",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "fabricName", Operator: "eq", Value: "fabric-1"},
							{Identifier: "category", Operator: "eq", Value: "HARDWARE"},
						},
					},
				},
			},
			"criteria-2": {
				{
					AlertRuleUUID:     "condition-test-rule",
					Priority:          100,
					CriteriaUUID:      "crit-fabric2",
					PrimaryMatchValue: "fabric-2",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "fabricName", Operator: "eq", Value: "fabric-2"},
							{Identifier: "category", Operator: "eq", Value: "SOFTWARE"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "OVERRIDE_SEVERITY", ActionValueStr: "critical"},
		},
	}

	re.AddRuleDefinition(rule)

	// Verify that conditions are indexed separately by primary key
	stats := re.GetPrimaryKeyStats()
	if stats["fabric-1"] != 1 {
		t.Errorf("Expected 1 condition for fabric-1, got %d", stats["fabric-1"])
	}
	if stats["fabric-2"] != 1 {
		t.Errorf("Expected 1 condition for fabric-2, got %d", stats["fabric-2"])
	}

	// Test evaluation for fabric-1 data
	data1 := Data{
		"fabricName": "fabric-1",
		"category":   "HARDWARE",
	}

	result1 := re.EvaluateRules(data1)
	if !result1.IsRuleHit {
		t.Error("Expected rule hit for fabric-1 data")
	}
	if result1.RuleUUID != "condition-test-rule" {
		t.Errorf("Expected rule UUID 'condition-test-rule', got '%s'", result1.RuleUUID)
	}

	// Test evaluation for fabric-2 data
	data2 := Data{
		"fabricName": "fabric-2",
		"category":   "SOFTWARE",
	}

	result2 := re.EvaluateRules(data2)
	if !result2.IsRuleHit {
		t.Error("Expected rule hit for fabric-2 data")
	}
	if result2.RuleUUID != "condition-test-rule" {
		t.Errorf("Expected rule UUID 'condition-test-rule', got '%s'", result2.RuleUUID)
	}

	// Test that incorrect data doesn't match
	data3 := Data{
		"fabricName": "fabric-1",
		"category":   "SOFTWARE", // Wrong category for fabric-1
	}

	result3 := re.EvaluateRules(data3)
	if result3.IsRuleHit {
		t.Error("Expected no rule hit for mismatched fabric-1 data")
	}

	t.Log("Condition-level optimization test passed: conditions indexed and evaluated independently")
}

// TestConditionPriorityOrdering validates that conditions with same primary key are ordered by priority
func TestConditionPriorityOrdering(t *testing.T) {
	re := NewRuleEngineInstance(&EvaluatorOptions{SortAscending: false}, nil)

	// Create two rules with same primary key but different priorities
	highPriorityRule := &RuleDefinition{
		AlertRuleUUID: "high-priority-rule",
		Name:          "High Priority Rule",
		Priority:      200,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					AlertRuleUUID:     "high-priority-rule",
					Priority:          200,
					CriteriaUUID:      "high-crit",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "fabricName", Operator: "eq", Value: "test-fabric"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "ESCALATE", ActionValueStr: "high-priority-action"},
		},
	}

	lowPriorityRule := &RuleDefinition{
		AlertRuleUUID: "low-priority-rule",
		Name:          "Low Priority Rule",
		Priority:      100,
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"criteria-1": {
				{
					AlertRuleUUID:     "low-priority-rule",
					Priority:          100,
					CriteriaUUID:      "low-crit",
					PrimaryMatchValue: "test-fabric",
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "fabricName", Operator: "eq", Value: "test-fabric"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "ALERT", ActionValueStr: "low-priority-action"},
		},
	}

	// Add rules in reverse order to test sorting
	re.AddRuleDefinition(lowPriorityRule)
	re.AddRuleDefinition(highPriorityRule)

	// Test that high priority rule is evaluated first
	data := Data{
		"fabricName": "test-fabric",
	}

	result := re.EvaluateRules(data)
	if !result.IsRuleHit {
		t.Error("Expected rule hit")
	}

	// With descending sort, high priority (200) should be evaluated first
	if result.RuleUUID != "high-priority-rule" {
		t.Errorf("Expected high priority rule to be evaluated first, got '%s'", result.RuleUUID)
	}

	t.Log("Condition priority ordering test passed: higher priority conditions evaluated first")
}
