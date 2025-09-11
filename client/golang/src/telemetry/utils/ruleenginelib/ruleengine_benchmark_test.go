package ruleenginelib

import (
	"fmt"
	"testing"
)

// BenchmarkRuleEvaluation compares performance with different numbers of rules
func BenchmarkRuleEvaluation(b *testing.B) {
	ruleCounts := []int{100, 500, 1000, 5000}

	for _, ruleCount := range ruleCounts {
		b.Run(fmt.Sprintf("Rules_%d", ruleCount), func(b *testing.B) {
			benchmarkWithRuleCount(b, ruleCount)
		})
	}
}

func benchmarkWithRuleCount(b *testing.B, ruleCount int) {
	re := NewRuleEngineInstance(nil, nil)

	// Create rules distributed across different primary keys
	primaryKeys := []string{"fabric-1", "fabric-2", "fabric-3", "PRIMARY_KEY_DEFAULT"}

	for i := 0; i < ruleCount; i++ {
		primaryKey := primaryKeys[i%len(primaryKeys)]
		rule := &RuleDefinition{
			AlertRuleUUID: fmt.Sprintf("rule-%d", i),
			Name:          fmt.Sprintf("Benchmark Rule %d", i),
			Priority:      int64(1000 - (i % 1000)), // Varying priorities
			Enabled:       true,
			MatchCriteriaEntries: map[string][]*RuleMatchCondition{
				fmt.Sprintf("criteria-%d", i): {
					{
						CriteriaUUID:      fmt.Sprintf("criteria-%d", i),
						PrimaryMatchValue: primaryKey,
						Condition: AstCondition{
							All: []AstConditional{
								{Identifier: "category", Operator: "eq", Value: fmt.Sprintf("CATEGORY_%d", i%10)},
								{Identifier: "severity", Operator: "eq", Value: "major"},
							},
						},
					},
				},
			},
			Actions: []*RuleAction{
				{ActionType: "TEST_ACTION", ActionValueStr: fmt.Sprintf("action-%d", i)},
			},
		}
		re.AddRuleDefinition(rule)
	}

	// Test data that will match fabric-1 rules (should be ~25% of total rules)
	testData := Data{
		"fabricName": "fabric-1",
		"category":   "CATEGORY_1", // Will match some rules
		"severity":   "major",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := re.EvaluateRules(testData)
		_ = result // Prevent optimization
	}
}

// BenchmarkIndexRebuild measures the cost of rebuilding indexes
func BenchmarkIndexRebuild(b *testing.B) {
	ruleCounts := []int{100, 500, 1000, 5000}

	for _, ruleCount := range ruleCounts {
		b.Run(fmt.Sprintf("IndexRebuild_%d", ruleCount), func(b *testing.B) {
			re := NewRuleEngineInstance(nil, nil)

			// Pre-populate rules
			for i := 0; i < ruleCount; i++ {
				rule := &RuleDefinition{
					AlertRuleUUID: fmt.Sprintf("rule-%d", i),
					Name:          fmt.Sprintf("Rule %d", i),
					Priority:      int64(i),
					Enabled:       true,
					MatchCriteriaEntries: map[string][]*RuleMatchCondition{
						fmt.Sprintf("criteria-%d", i): {
							{
								CriteriaUUID:      fmt.Sprintf("criteria-%d", i),
								PrimaryMatchValue: fmt.Sprintf("fabric-%d", i%10),
								Condition: AstCondition{
									All: []AstConditional{
										{Identifier: "category", Operator: "eq", Value: "TEST"},
									},
								},
							},
						},
					},
					Actions: []*RuleAction{
						{ActionType: "TEST_ACTION", ActionValueStr: "test"},
					},
				}
				re.RuleMap[rule.AlertRuleUUID] = rule
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				re.rebuildIndexes()
			}
		})
	}
}

// BenchmarkPrimaryKeyExtraction measures the cost of extracting primary keys
func BenchmarkPrimaryKeyExtraction(b *testing.B) {
	re := NewRuleEngineInstance(nil, nil)

	testCases := []Data{
		{"fabricName": "fabric-1", "category": "HARDWARE"},
		{"category": "SYSTEM_HEALTH"},
		{"title": "Some alert"},
		{"fabricName": "", "category": "CONNECTIVITY"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data := testCases[i%len(testCases)]
		key := re.extractPrimaryKey(data)
		_ = key // Prevent optimization
	}
}

// BenchmarkCompareOptimizedVsUnoptimized compares performance with a simulated unoptimized version
func BenchmarkCompareOptimizedVsUnoptimized(b *testing.B) {
	ruleCount := 1000

	// Optimized version
	b.Run("Optimized", func(b *testing.B) {
		re := NewRuleEngineInstance(nil, nil)

		for i := 0; i < ruleCount; i++ {
			rule := &RuleDefinition{
				AlertRuleUUID: fmt.Sprintf("rule-%d", i),
				Name:          fmt.Sprintf("Rule %d", i),
				Priority:      int64(1000 - i),
				Enabled:       true,
				MatchCriteriaEntries: map[string][]*RuleMatchCondition{
					fmt.Sprintf("criteria-%d", i): {
						{
							CriteriaUUID:      fmt.Sprintf("criteria-%d", i),
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
					{ActionType: "TEST_ACTION", ActionValueStr: "test"},
				},
			}
			re.AddRuleDefinition(rule)
		}

		testData := Data{"fabricName": "fabric-1", "category": "TEST"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := re.EvaluateRules(testData)
			_ = result
		}
	})

	// Simulated unoptimized version (iterate through all rules in map)
	b.Run("Unoptimized", func(b *testing.B) {
		re := NewRuleEngineInstance(nil, nil)

		for i := 0; i < ruleCount; i++ {
			rule := &RuleDefinition{
				AlertRuleUUID: fmt.Sprintf("rule-%d", i),
				Name:          fmt.Sprintf("Rule %d", i),
				Priority:      int64(1000 - i),
				Enabled:       true,
				MatchCriteriaEntries: map[string][]*RuleMatchCondition{
					fmt.Sprintf("criteria-%d", i): {
						{
							CriteriaUUID:      fmt.Sprintf("criteria-%d", i),
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
					{ActionType: "TEST_ACTION", ActionValueStr: "test"},
				},
			}
			re.RuleMap[rule.AlertRuleUUID] = rule
		}

		testData := Data{"fabricName": "fabric-1", "category": "TEST"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate unoptimized evaluation - iterate through all rules
			result := evaluateAllRules(re, testData)
			_ = result
		}
	})
}

// evaluateAllRules simulates the old unoptimized approach
func evaluateAllRules(re *RuleEngine, data Data) RuleLookupResult {
	re.Mutex.Lock()
	defer re.Mutex.Unlock()

	// Simulate old behavior - iterate through all rules without indexing
	for _, rule := range re.RuleMap {
		if result := re.evaluateSingleRule(rule, data); result.IsRuleHit {
			return result
		}
	}
	return RuleLookupResult{}
}
