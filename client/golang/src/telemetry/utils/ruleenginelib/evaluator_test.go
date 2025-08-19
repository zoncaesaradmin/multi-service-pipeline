// ...existing code...
package ruleenginelib

import (
	"testing"
)

func TestEvaluateConditional(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	tests := []struct {
		conditional *AstConditional
		identifier  interface{}
		expected    bool
	}{
		{&AstConditional{
			Fact:     "name",
			Operator: "eq",
			Value:    "Icheka",
		},
			"Icheka",
			true,
		},
		{&AstConditional{
			Fact:     "name",
			Operator: "eq",
			Value:    "Icheka",
		},
			"Ronie",
			false,
		},
	}

	for i, tt := range tests {
		if ok := EvaluateConditional(tt.conditional, tt.identifier); ok != tt.expected {
			t.Errorf("tests[%d] - expected EvaluateConditional to return %t, got=%t", i, tt.expected, ok)
		}
	}
}

func TestEvaluateAllCondition(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	tests := []struct {
		payload struct {
			conditions []AstConditional
			identifier Data
		}
		expected bool
	}{
		{
			payload: struct {
				conditions []AstConditional
				identifier Data
			}{
				conditions: []AstConditional{
					{
						Fact:     "planet",
						Operator: "eq",
						Value:    "Neptune",
					},
					{
						Fact:     "colour",
						Operator: "eq",
						Value:    "black",
					},
				},
				identifier: Data{
					"planet": "Neptune",
					"colour": "black",
				},
			},
			expected: true,
		},
		{
			payload: struct {
				conditions []AstConditional
				identifier Data
			}{
				conditions: []AstConditional{
					{
						Fact:     "planet",
						Operator: "eq",
						Value:    "Saturn",
					},
					{
						Fact:     "colour",
						Operator: "eq",
						Value:    "black",
					},
				},
				identifier: Data{
					"planet": "Neptune",
					"colour": "black",
				},
			},
			expected: false,
		},
	}

	for i, tt := range tests {
		if ok := EvaluateAllCondition(&tt.payload.conditions, tt.payload.identifier); ok != tt.expected {
			t.Errorf("tests[%d] - expected EvaluateAllCondition to be %t, got=%t", i, tt.expected, ok)
		}
	}
}

func TestEvaluateAnyCondition(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	tests := []struct {
		payload struct {
			conditions []AstConditional
			identifier Data
		}
		expected bool
	}{
		{
			payload: struct {
				conditions []AstConditional
				identifier Data
			}{
				conditions: []AstConditional{
					{
						Fact:     "planet",
						Operator: "eq",
						Value:    "Neptune",
					},
					{
						Fact:     "colour",
						Operator: "eq",
						Value:    "black",
					},
				},
				identifier: Data{
					"planet": "Neptune",
					"colour": "black",
				},
			},
			expected: true,
		},
		{
			payload: struct {
				conditions []AstConditional
				identifier Data
			}{
				conditions: []AstConditional{
					{
						Fact:     "planet",
						Operator: "eq",
						Value:    "Saturn",
					},
					{
						Fact:     "colour",
						Operator: "eq",
						Value:    "black",
					},
				},
				identifier: Data{
					"planet": "Neptune",
					"colour": "black",
				},
			},
			expected: true,
		},
		{
			payload: struct {
				conditions []AstConditional
				identifier Data
			}{
				conditions: []AstConditional{
					{
						Fact:     "planet",
						Operator: "eq",
						Value:    "Saturn",
					},
					{
						Fact:     "colour",
						Operator: "eq",
						Value:    "white",
					},
				},
				identifier: Data{
					"planet": "Neptune",
					"colour": "black",
				},
			},
			expected: false,
		},
	}

	for i, tt := range tests {
		if ok := EvaluateAnyCondition(&tt.payload.conditions, tt.payload.identifier); ok != tt.expected {
			t.Errorf("tests[%d] - expected EvaluateAnyCondition to be %t, got=%t", i, tt.expected, ok)
		}
	}
}
