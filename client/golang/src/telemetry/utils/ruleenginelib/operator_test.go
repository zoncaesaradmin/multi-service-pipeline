// ...existing code...
package ruleenginelib

import "testing"

func TestEvaluateOperator(t *testing.T) {
	tests := []struct {
		identifier interface{}
		value      interface{}
		operator   string
		expected   bool
	}{
		{"hi", "hi", "eq", true},
		{"hi", "hi", "=", true},
		{"hi", "his", "=", false},
		{"hi", 4, "=", false},
		{4, 4, "=", true},

		{4, 4, "!=", false},
		{4, 5, "neq", true},

		{4, 5, "<", true},
		{6, 5, "lt", false},

		{4, 5, ">", false},
		{6, 5, "gt", true},

		{4, 5, ">=", false},
		{6, 5, "gte", true},
		{5, 5, "gte", true},

		{4, 5, "<=", true},
		{6, 5, "lte", false},
		{5, 5, "lte", true},
	}

	for i, tt := range tests {
		ok, err := EvaluateOperator(tt.identifier, tt.value, tt.operator)
		if err != nil {
			t.Errorf("tests[%d] - unexpected error (%s)", i, err)
		}
		if ok != tt.expected {
			t.Errorf("tests[%d] - expected EvaluateOperator to be %t, got=%t", i, tt.expected, ok)
		}
	}
}

func TestNumericCompareErrorCases(t *testing.T) {
	// Non-numeric left-hand side should return error
	if _, err := evaluateLessThan("not-a-number", 5); err == nil {
		t.Fatalf("expected error when dataValue is not numeric")
	}
	if _, err := evaluateGreaterThan("not-a-number", 5); err == nil {
		t.Fatalf("expected error when dataValue is not numeric")
	}
	if _, err := evaluateGreaterThanOrEqual("not-a-number", 5); err == nil {
		t.Fatalf("expected error when dataValue is not numeric")
	}
	if _, err := evaluateLessThanOrEqual("not-a-number", 5); err == nil {
		t.Fatalf("expected error when dataValue is not numeric")
	}

	// Non-numeric right-hand side should also return error
	if _, err := evaluateLessThan(5, "not-a-number"); err == nil {
		t.Fatalf("expected error when value is not numeric")
	}
	if _, err := evaluateGreaterThan(5, "not-a-number"); err == nil {
		t.Fatalf("expected error when value is not numeric")
	}
}

func TestEvaluateComparableEqualsSuccess_Alt(t *testing.T) {
	ok, err := evaluateComparableEquals("abc", "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected comparable equals to be true for equal strings")
	}
}

func TestEvaluateComparableEqualsError(t *testing.T) {
	// slices are not comparable in Go, should return an error
	a := []int{1, 2}
	b := []int{1, 2}
	ok, err := evaluateComparableEquals(a, b)
	if err == nil {
		t.Fatalf("expected error for non-comparable types, got ok=%v", ok)
	}
}

func TestNumericComparisonsIntFloat(t *testing.T) {
	// int vs float comparisons via EvaluateOperator
	res, err := EvaluateOperator(5, 5.0, "=")
	if err != nil || !res {
		t.Fatalf("expected 5 == 5.0 to be true, got res=%v err=%v", res, err)
	}

	res, err = EvaluateOperator(5, 6.0, "<")
	if err != nil || !res {
		t.Fatalf("expected 5 < 6.0 to be true, got res=%v err=%v", res, err)
	}

	res, err = EvaluateOperator(7.5, 7, ">")
	if err != nil || !res {
		t.Fatalf("expected 7.5 > 7 to be true, got res=%v err=%v", res, err)
	}
}

func TestEvaluateOperatorUnknownExtra(t *testing.T) {
	_, err := EvaluateOperator("a", "b", "not-op")
	if err == nil {
		t.Fatalf("expected error for unknown operator")
	}
}

func TestEvaluateEqualsStringSliceMismatch(t *testing.T) {
	// dataValue is []string and value is a string not in slice
	ok, err := evaluateEquals([]string{"x", "y"}, "z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected false for missing value in slice")
	}
}
