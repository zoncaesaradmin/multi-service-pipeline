package ruleenginelib

import (
	"testing"
)

const testRuleUUID = "test-uuid"

func TestNewRuleEngineInstance(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	if re == nil {
		t.Error("Expected non-nil RuleEngine instance")
	}
}

func TestEvaluateStructAndRule(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	condition := AstCondition{
		All: []AstConditional{{Identifier: "planet", Operator: "eq", Value: "Earth"}},
	}
	data := Data{"planet": "Earth"}
	if !re.EvaluateMatchCondition(condition, data) {
		t.Error("EvaluateMatchCondition should return true for matching data")
	}
	if !EvaluateAstCondition(condition, data, &Options{AllowUndefinedVars: true}) {
		t.Error("EvaluateAstCondition should return true for matching data")
	}
}

func TestEvaluateRules(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	// Create a test rule definition manually
	ruleDefinition := &RuleDefinition{
		AlertRuleUUID: testRuleUUID,
		Name:          "Test Rule",
		Priority:      100, // Add priority for proper indexing
		Enabled:       true,
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"default": {
				{
					AlertRuleUUID:     testRuleUUID,
					Priority:          100,
					CriteriaUUID:      "condition-1",
					PrimaryMatchValue: "PRIMARY_KEY_DEFAULT", // Add primary key for indexing
					Condition: AstCondition{
						All: []AstConditional{
							{Identifier: "planet", Operator: "eq", Value: "Earth"},
						},
					},
				},
			},
		},
		Actions: []*RuleAction{
			{ActionType: "TEST_ACTION", ActionValueStr: "test-value"},
		},
	}

	// Add the rule to the engine using the new method that rebuilds indexes
	re.AddRuleDefinition(ruleDefinition)

	// Test with matching data
	data := Data{"planet": "Earth"}
	result := re.EvaluateRules(data)

	if !result.IsRuleHit {
		t.Error("EvaluateRules should return IsRuleHit=true for matching data")
	}

	if result.RuleUUID != testRuleUUID {
		t.Errorf("EvaluateRules should return correct UUID, got %s", result.RuleUUID)
	}

	if len(result.Actions) != 1 {
		t.Errorf("EvaluateRules should return one action, got %d", len(result.Actions))
	} else if result.Actions[0].ActionType != "TEST_ACTION" {
		t.Errorf("EvaluateRules should return correct action type, got %s", result.Actions[0].ActionType)
	}
}

func TestEvaluateConditionSwitch(t *testing.T) {
	conds := []AstConditional{{Identifier: "planet", Operator: "eq", Value: "Earth"}}
	data := Data{"planet": "Earth"}
	if !EvaluateCondition(&conds, "all", data) {
		t.Error("EvaluateCondition 'all' should return true")
	}
	if !EvaluateCondition(&conds, "any", data) {
		t.Error("EvaluateCondition 'any' should return true")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Error("EvaluateCondition should panic on invalid kind")
		}
	}()
	EvaluateCondition(&conds, "invalid", data)
}

func TestDeepCopyActions(t *testing.T) {
	// Create original actions
	originalActions := []*RuleAction{
		{ActionType: "ACTION1", ActionValueStr: "value1"},
		{ActionType: "ACTION2", ActionValueStr: "value2"},
	}

	// Copy the actions
	copiedActions := deepCopyActions(originalActions)

	// Verify the copied actions have the same values
	if len(copiedActions) != len(originalActions) {
		t.Errorf("deepCopyActions should create same number of actions, got %d expected %d",
			len(copiedActions), len(originalActions))
	}

	for i, orig := range originalActions {
		if copiedActions[i].ActionType != orig.ActionType {
			t.Errorf("Copied action %d should have same ActionType, got %s expected %s",
				i, copiedActions[i].ActionType, orig.ActionType)
		}
		if copiedActions[i].ActionValueStr != orig.ActionValueStr {
			t.Errorf("Copied action %d should have same ActionValueStr, got %s expected %s",
				i, copiedActions[i].ActionValueStr, orig.ActionValueStr)
		}
	}

	// Modify original to verify deep copy
	originalActions[0].ActionType = "MODIFIED"
	if copiedActions[0].ActionType == "MODIFIED" {
		t.Error("Modifying original action should not affect copied actions")
	}
}
