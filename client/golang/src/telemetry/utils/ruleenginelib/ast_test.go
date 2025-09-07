package ruleenginelib

import (
	"encoding/json"
	"reflect"
	"testing"
)

const testActionValue = "test value"

// testAstConditionalSerialization tests JSON serialization of AstConditional
func testAstConditionalSerialization(t *testing.T, cond AstConditional, expectedJSON string) {
	bytes, err := json.Marshal(cond)
	if err != nil {
		t.Fatalf("Failed to marshal AstConditional: %v", err)
	}
	if string(bytes) != expectedJSON {
		t.Errorf("JSON serialization mismatch: got %s, want %s", string(bytes), expectedJSON)
	}
}

// testAstConditionalDeserialization tests JSON deserialization of AstConditional
func testAstConditionalDeserialization(t *testing.T, cond AstConditional, jsonString string) {
	var decoded AstConditional
	if err := json.Unmarshal([]byte(jsonString), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AstConditional: %v", err)
	}

	// For numeric values, JSON unmarshaling may convert to float64
	if reflect.TypeOf(cond.Value) == reflect.TypeOf(int(0)) {
		if decodedVal, ok := decoded.Value.(float64); ok {
			decoded.Value = int(decodedVal)
		}
	}

	validateAstConditionalFields(t, decoded, cond)
}

// validateAstConditionalFields validates the fields of an AstConditional
func validateAstConditionalFields(t *testing.T, decoded, expected AstConditional) {
	if decoded.Identifier != expected.Identifier {
		t.Errorf("Identifier mismatch: got %s, want %s", decoded.Identifier, expected.Identifier)
	}
	if decoded.Operator != expected.Operator {
		t.Errorf("Operator mismatch: got %s, want %s", decoded.Operator, expected.Operator)
	}
	if !reflect.DeepEqual(decoded.Value, expected.Value) {
		t.Errorf("Value mismatch: got %v (%T), want %v (%T)",
			decoded.Value, decoded.Value, expected.Value, expected.Value)
	}
}

func TestAstConditionalSerialization(t *testing.T) {
	tests := []struct {
		name       string
		cond       AstConditional
		jsonString string
	}{
		{
			name: "string value",
			cond: AstConditional{
				Identifier: "category",
				Operator:   "equal",
				Value:      "networking",
			},
			jsonString: `{"identifier":"category","operator":"equal","value":"networking"}`,
		},
		{
			name: "numeric value",
			cond: AstConditional{
				Identifier: "priority",
				Operator:   "greater_than",
				Value:      5,
			},
			jsonString: `{"identifier":"priority","operator":"greater_than","value":5}`,
		},
		{
			name: "boolean value",
			cond: AstConditional{
				Identifier: "active",
				Operator:   "equal",
				Value:      true,
			},
			jsonString: `{"identifier":"active","operator":"equal","value":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAstConditionalSerialization(t, tt.cond, tt.jsonString)
			testAstConditionalDeserialization(t, tt.cond, tt.jsonString)
		})
	}
}

func TestAstConditionSerialization(t *testing.T) {
	condition := AstCondition{
		Any: []AstConditional{
			{Identifier: "device", Operator: "equal", Value: "switch"},
			{Identifier: "device", Operator: "equal", Value: "router"},
		},
		All: []AstConditional{
			{Identifier: "status", Operator: "equal", Value: "down"},
		},
	}

	// Test serialization
	bytes, err := json.Marshal(condition)
	if err != nil {
		t.Fatalf("Failed to marshal AstCondition: %v", err)
	}

	// Test deserialization
	var decoded AstCondition
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AstCondition: %v", err)
	}

	if len(decoded.Any) != len(condition.Any) {
		t.Errorf("Any length mismatch: got %d, want %d", len(decoded.Any), len(condition.Any))
	}
	if len(decoded.All) != len(condition.All) {
		t.Errorf("All length mismatch: got %d, want %d", len(decoded.All), len(condition.All))
	}

	// Compare first "Any" condition
	if len(decoded.Any) > 0 && len(condition.Any) > 0 {
		if decoded.Any[0].Identifier != condition.Any[0].Identifier {
			t.Errorf("Any[0] Identifier mismatch: got %s, want %s",
				decoded.Any[0].Identifier, condition.Any[0].Identifier)
		}
		if decoded.Any[0].Operator != condition.Any[0].Operator {
			t.Errorf("Any[0] Operator mismatch: got %s, want %s",
				decoded.Any[0].Operator, condition.Any[0].Operator)
		}
		if !reflect.DeepEqual(decoded.Any[0].Value, condition.Any[0].Value) {
			t.Errorf("Any[0] Value mismatch: got %v, want %v",
				decoded.Any[0].Value, condition.Any[0].Value)
		}
	}
}

func TestEmptyConditions(t *testing.T) {
	// Test empty Any, non-empty All
	condition1 := AstCondition{
		Any: []AstConditional{},
		All: []AstConditional{
			{Identifier: "status", Operator: "equal", Value: "down"},
		},
	}

	// Test non-empty Any, empty All
	condition2 := AstCondition{
		Any: []AstConditional{
			{Identifier: "device", Operator: "equal", Value: "switch"},
		},
		All: []AstConditional{},
	}

	// Test both empty
	condition3 := AstCondition{
		Any: []AstConditional{},
		All: []AstConditional{},
	}

	// Serialize and deserialize all conditions
	for i, cond := range []AstCondition{condition1, condition2, condition3} {
		bytes, err := json.Marshal(cond)
		if err != nil {
			t.Fatalf("Case %d: Failed to marshal AstCondition: %v", i+1, err)
		}

		var decoded AstCondition
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("Case %d: Failed to unmarshal AstCondition: %v", i+1, err)
		}

		if len(decoded.Any) != len(cond.Any) {
			t.Errorf("Case %d: Any length mismatch: got %d, want %d", i+1, len(decoded.Any), len(cond.Any))
		}
		if len(decoded.All) != len(cond.All) {
			t.Errorf("Case %d: All length mismatch: got %d, want %d", i+1, len(decoded.All), len(cond.All))
		}
	}
}

func TestParseJSON(t *testing.T) {
	// Valid JSON for RuleDefinition
	validJSON := `[
		{
			"alertRuleUUID": "rule1",
			"name": "Test Rule",
			"priority": 5,
			"description": "Test rule description",
			"enabled": true
		}
	]`

	// Test successful parsing
	rules := ParseJSON(validJSON)
	if rules == nil {
		t.Fatal("ParseJSON returned nil for valid JSON")
	}
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(rules))
	}
	rule := rules[0]
	if rule.AlertRuleUUID != "rule1" {
		t.Errorf("AlertRuleUUID mismatch: got %s, want %s", rule.AlertRuleUUID, "rule1")
	}
	if rule.Name != "Test Rule" {
		t.Errorf("Name mismatch: got %s, want %s", rule.Name, "Test Rule")
	}
	if rule.Priority != 5 {
		t.Errorf("Priority mismatch: got %d, want %d", rule.Priority, 5)
	}
	if rule.Description != "Test rule description" {
		t.Errorf("Description mismatch: got %s, want %s", rule.Description, "Test rule description")
	}
	if !rule.Enabled {
		t.Error("Enabled mismatch: got false, want true")
	}

	// Test invalid JSON
	invalidJSON := `{not valid json`
	defer func() {
		if r := recover(); r == nil {
			t.Error("ParseJSON should panic on invalid JSON")
		}
	}()
	ParseJSON(invalidJSON) // Should panic
}

func TestRuleAction(t *testing.T) {
	action := RuleAction{
		ActionType:     "ACKNOWLEDGE",
		ActionValueStr: testActionValue,
	}

	// Just test basic property access
	if action.ActionType != "ACKNOWLEDGE" {
		t.Errorf("ActionType mismatch: got %s, want %s", action.ActionType, "ACKNOWLEDGE")
	}
	if action.ActionValueStr != testActionValue {
		t.Errorf("ActionValueStr mismatch: got %s, want %s", action.ActionValueStr, testActionValue)
	}
}
