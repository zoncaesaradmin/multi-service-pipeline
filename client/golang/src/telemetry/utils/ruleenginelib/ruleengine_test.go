package ruleenginelib

import (
	"testing"
)

func TestNewRuleEngineInstance(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	if re == nil {
		t.Error("Expected non-nil RuleEngine instance")
	}
}

func TestAddAndDeleteRule(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	ruleJSON := `{"uuid":"test-uuid","payload":[{"condition":{"any":[],"all":[]},"actions":[]}],"state":true}`
	re.AddRule(ruleJSON)
	if _, ok := re.RuleMap["test-uuid"]; !ok {
		t.Error("Rule not added to RuleMap")
	}
	re.DeleteRule(ruleJSON)
	if _, ok := re.RuleMap["test-uuid"]; ok {
		t.Error("Rule not deleted from RuleMap")
	}
}

func TestEvaluateStructAndRule(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	rule := &RuleEntry{
		Condition: AstCondition{
			All: []AstConditional{{Fact: "planet", Operator: "eq", Value: "Earth"}},
		},
		Actions: []Action{},
	}
	data := Data{"planet": "Earth"}
	if !re.EvaluateStruct(rule, data) {
		t.Error("EvaluateStruct should return true for matching data")
	}
	if !EvaluateRule(rule, data, &Options{AllowUndefinedVars: true}) {
		t.Error("EvaluateRule should return true for matching data")
	}
}

func TestEvaluateRules(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	ruleJSON := `{"uuid":"test-uuid","payload":[{"condition":{"any":[],"all":[{"identifier":"planet","operator":"eq","value":"Earth"}]},"actions":[]}],"state":true}`
	re.AddRule(ruleJSON)
	data := Data{"planet": "Earth"}
	matched, uuid, entry := re.EvaluateRules(data)
	if !matched || uuid != "test-uuid" || entry == nil {
		t.Errorf("EvaluateRules should match and return correct uuid and entry, got matched=%v uuid=%v entry=%v", matched, uuid, entry)
	}
}

func TestEvaluateConditionSwitch(t *testing.T) {
	conds := []AstConditional{{Fact: "planet", Operator: "eq", Value: "Earth"}}
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
