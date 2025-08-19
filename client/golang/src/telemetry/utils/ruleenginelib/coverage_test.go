package ruleenginelib

import "testing"

func TestEvaluateOperatorErrorBranches(t *testing.T) {
	// eq with non-comparable types
	ok, err := EvaluateOperator(struct{}{}, struct{}{}, "eq")
	if ok || err == nil {
		t.Error("eq with non-comparable types should be false and should error")
	}
	// lt with non-number
	ok, err = EvaluateOperator("a", "b", "lt")
	if ok || err == nil {
		t.Error("lt with non-number should error")
	}
	// unknown operator
	ok, err = EvaluateOperator("a", "b", "unknown")
	if ok || err == nil {
		t.Error("unknown operator should error")
	}
}

func TestAssertIsNumberBranches(t *testing.T) {
	// int
	v, err := assertIsNumber(42)
	if v != 42 || err != nil {
		t.Error("assertIsNumber failed for int")
	}
	// float64
	v, err = assertIsNumber(42.5)
	if v != 42.5 || err != nil {
		t.Error("assertIsNumber failed for float64")
	}
	// string
	_, err = assertIsNumber("notnum")
	if err == nil {
		t.Error("assertIsNumber should error for string")
	}
}

func TestGetFactValueAllowUndefinedVarsTrue(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	cond := &AstConditional{Fact: "missing", Operator: "eq", Value: "val"}
	v := GetFactValue(cond, Data{})
	if v != false {
		t.Error("GetFactValue should return false when AllowUndefinedVars is true and value is missing")
	}
}

func TestEvaluateConditionalNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("EvaluateConditional should panic when Value is nil")
		}
	}()
	EvaluateConditional(&AstConditional{Fact: "planet", Operator: "eq", Value: nil}, "Earth")
}

func TestEvaluateRuleNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("EvaluateRule should panic when rule is nil")
		}
	}()
	EvaluateRule(nil, nil, &Options{AllowUndefinedVars: true})
}
func TestEvaluateOperatorBranches(t *testing.T) {
	// anyof with slice
	ok, err := EvaluateOperator("a", []interface{}{"a", "b"}, "anyof")
	if !ok || err != nil {
		t.Error("anyof with slice failed")
	}
	ok, err = EvaluateOperator("c", []interface{}{"a", "b"}, "anyof")
	if ok || err != nil {
		t.Error("anyof with slice should be false")
	}
	// noneof with slice
	ok, err = EvaluateOperator("a", []interface{}{"a", "b"}, "noneof")
	if ok || err != nil {
		t.Error("noneof with slice should be false")
	}
	ok, err = EvaluateOperator("c", []interface{}{"a", "b"}, "noneof")
	if !ok || err != nil {
		t.Error("noneof with slice should be true")
	}
	// anyof with single value
	ok, err = EvaluateOperator("a", "a", "anyof")
	if !ok || err != nil {
		t.Error("anyof with single value failed")
	}
	ok, err = EvaluateOperator("b", "a", "anyof")
	if ok || err != nil {
		t.Error("anyof with single value should be false")
	}
	// noneof with single value
	ok, err = EvaluateOperator("a", "a", "noneof")
	if ok || err != nil {
		t.Error("noneof with single value should be false")
	}
	ok, err = EvaluateOperator("b", "a", "noneof")
	if !ok || err != nil {
		t.Error("noneof with single value should be true")
	}
	// eq/neq with numbers
	ok, err = EvaluateOperator(4, 4, "eq")
	if !ok || err != nil {
		t.Error("eq with numbers failed")
	}
	ok, err = EvaluateOperator(4, 5, "neq")
	if !ok || err != nil {
		t.Error("neq with numbers failed")
	}
	// lt/gt/gte/lte
	ok, err = EvaluateOperator(4, 5, "lt")
	if !ok || err != nil {
		t.Error("lt failed")
	}
	ok, err = EvaluateOperator(6, 5, "gt")
	if !ok || err != nil {
		t.Error("gt failed")
	}
	ok, err = EvaluateOperator(5, 5, "gte")
	if !ok || err != nil {
		t.Error("gte failed")
	}
	ok, err = EvaluateOperator(4, 5, "lte")
	if !ok || err != nil {
		t.Error("lte failed")
	}
}

func TestRuleEngineMethodsPanics(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	// DeleteRule with invalid JSON
	defer func() {
		if r := recover(); r == nil {
			t.Error("DeleteRule should panic on invalid JSON")
		}
	}()
	re.DeleteRule("not a json")
}

func TestEvaluateRulePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("EvaluateRule should panic on nil options")
		}
	}()
	EvaluateRule(nil, nil, nil)
}

func TestParseJSONMissingFields(t *testing.T) {
	// Should not panic, just return RuleBlock with zero values
	rb := ParseJSON(`{"uuid":"","payload":[],"state":true}`)
	if rb == nil {
		t.Error("ParseJSON should return non-nil RuleBlock")
	}
}
